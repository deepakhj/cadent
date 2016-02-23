/*
	The Cassandra write

	The table should have this schema to match the repr item


		keyspace: base keyspace name (default: metric)
		metric_table: base table name (default: metric)
		path_table: base table name (default: path)
		segment_table: base table name (default: segment)
		write_consistency: "one"
		read_consistency: "one"
		port: 9042
		numcons: 5  (connection pool size)
		timeout: "30s"
		user: ""
		pass: ""
		# NOPE: batch_count: batch this many inserts for much faster insert performance (default 1000)
		# NOPE: periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package writers

import (
	"consthash/server/dbs"
	"consthash/server/repr"
	stats "consthash/server/stats"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"strings"
	"sync"
	"time"
)

/** Being Cassandra we need some mappings to match the schemas **/

/**
	CREATE TYPE metric_point (
        max double,
        mean double,
        min double,
        sum double,
        count int
    );
*/

type CassMetricPoint struct {
	Max   float64
	Mean  float64
	Min   float64
	Sum   float64
	Count int
}

/*
	CREATE TYPE metric_id (
        path text,
        resolution int
    );
*/

type CassMetricID struct {
	Path       string
	Resolution int
}

/*
 CREATE TABLE metric (
        id frozen<metric_id>,
        time bigint,
        point frozen<metric_point>
 )
*/
type CassMetric struct {
	Id         CassMetricID
	Time       int64
	Resolution CassMetricPoint
}

/*
 	metric.segment (
       pos int,
       segment text,
   		PRIMARY KEY (pos, segment)
    )
*/
type CassSegment struct {
	Pos     int
	Segment string
}

/*
 	metric.path (
		segment frozen<segment_pos>,
		path text,
		length int,
		has_data int,
		PRIMARY KEY (segment, path)
*/
type CassPath struct {
	Segment CassSegment
	Path    string
	Length  int
	Hasdata bool
}

/****************** Writer *********************/
type CassandraWriter struct {
	db   *dbs.CassandraDB
	conn *gocql.Session

	write_list     []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	max_write_size int             // size of that buffer before a flush
	max_idle       time.Duration   // either max_write_size will trigger a write or this time passing will
	write_lock     sync.Mutex
	paths_inserted map[string]bool // just to not do "extra" work on paths we've already indexed

	log *logging.Logger
}

func NewCassandraWriter() *CassandraWriter {
	cass := new(CassandraWriter)
	cass.log = logging.MustGetLogger("writers.cassandra")
	cass.paths_inserted = make(map[string]bool)
	return cass
}

func (cass *CassandraWriter) Config(conf map[string]interface{}) (err error) {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("`dsn` (server1,server2,server3) is needed for cassandra config")
	}
	dsn := gots.(string)

	db, err := dbs.NewDB("cassandra", dsn, conf)
	if err != nil {
		return err
	}
	cass.db = db.(*dbs.CassandraDB)
	cass.conn = db.Connection().(*gocql.Session)

	return nil
}

// keep an index of the stat keys and their fragments so we can look up
func (cass *CassandraWriter) InsertStatKey(skey string) error {

	if _, ok := cass.paths_inserted[skey]; ok {
		stats.StatsdClient.Incr("writer.cassandra.cached-writes-path", 1)
		return nil
	}

	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.write.path-time-ns"), time.Now())
	stats.StatsdClient.Incr("writer.cassandra.noncached-writes-path", 1)

	s_parts := strings.Split(skey, ".")
	p_len := len(s_parts)
	cur_part := ""
	segments := []CassSegment{}
	paths := []CassPath{}

	for idx, part := range s_parts {
		if len(cur_part) > 1 {
			cur_part += "."
		}
		cur_part += part
		on_segment := CassSegment{
			Segment: cur_part,
			Pos:     idx,
		}
		segments = append(segments, on_segment)

		on_path := CassPath{
			Segment: on_segment,
			Path:    skey,
			Length:  p_len - 1, // starts at 0
		}

		paths = append(paths, on_path)
	}

	last_path := paths[len(paths)-1]
	// now to upsert them all (inserts in cass are upserts)
	for idx, seg := range segments {
		Q := fmt.Sprintf(
			"INSERT INTO %s (pos, segment) VALUES  (?, ?) ",
			cass.db.SegmentTable(),
		)
		err := cass.conn.Query(Q,
			seg.Pos, seg.Segment,
		).Exec()

		if err != nil {
			cass.log.Error("Could not insert segment %v", seg)
			stats.StatsdClient.Incr("writer.cassandra.segment-failures", 1)
		} else {
			stats.StatsdClient.Incr("writer.cassandra.segment-writes", 1)
		}

		// now for each "partial path" add in the fact that it's not a "data" node
		// for each "segment" add in the path to do a segment to path(s) lookup
		// the skey one obviously has data
		// if key is consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
		/* insert

		consthash -> consthash.zipperwork
		consthash.zipperwork -> consthash.zipperwork.local
		consthash.zipperwork.local -> consthash.zipperwork.local.writer
		consthash.zipperwork.local.writer -> consthash.zipperwork.local.writer.cassandra

		as dataless nodes

		*/
		if skey != seg.Segment && idx < len(paths)-1 {
			Q = fmt.Sprintf(
				"INSERT INTO %s (segment, path, length, has_data) VALUES  ({pos: ?, segment: ?}, ?, ?, ?)",
				cass.db.PathTable(),
			)
			err = cass.conn.Query(Q,
				seg.Pos, seg.Segment, seg.Segment+"."+s_parts[idx+1], seg.Pos+1, false,
			).Exec()
			cass.log.Debug("Seg INS: %s PATH: %s Len: %d", seg.Segment, seg.Segment+"."+s_parts[idx+1], seg.Pos)
		}

		// for each "segment" add in the path to do a segment to path(s) lookup
		/*
			for key consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			insert
			consthash -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local.writer -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local.writer.cassandra -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
			consthash.zipperwork.local.writer.cassandra.write -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99

		*/
		Q = fmt.Sprintf(
			"INSERT INTO %s (segment, path, length, has_data) VALUES  ({pos: ?, segment: ?}, ?, ?, ?)",
			cass.db.PathTable(),
		)
		/*
			segment frozen<segment_pos>,
			path text,
			length int
		*/
		err = cass.conn.Query(Q,
			seg.Pos, seg.Segment, skey, p_len-1, true,
		).Exec()

		if err != nil {
			cass.log.Error("Could not insert path %v :: %v", last_path, err)
			stats.StatsdClient.Incr("writer.cassandra.path-failures", 1)
		} else {
			stats.StatsdClient.Incr("writer.cassandra.path-writes", 1)
		}
	}

	//cached bits
	cass.paths_inserted[skey] = true
	return nil

}

func (cass *CassandraWriter) InsertOne(stat repr.StatRepr) (int, error) {

	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.write.metric-time-ns"), time.Now())

	ttl := int64(0)
	if stat.TTL > 0 {
		ttl = stat.TTL
	}

	cass.InsertStatKey(stat.Key)

	Q := fmt.Sprintf(
		"INSERT INTO %s (id, time, point) VALUES  ({path: ?, resolution: ?}, ?, {sum: ?, mean: ?, min: ?, max: ?, count: ?}) ",
		cass.db.MetricTable(),
	)
	if ttl > 0 {
		Q += fmt.Sprintf(" USING TTL %d", ttl)
	}

	err := cass.conn.Query(Q,
		stat.Key, int64(stat.Resolution), stat.Time.UnixNano(), float64(stat.Sum), float64(stat.Mean), float64(stat.Min), float64(stat.Max), stat.Count,
	).Exec()

	if err != nil {
		cass.log.Error("Cassandra Driver: insert failed, %v", err)
		stats.StatsdClient.Incr("writer.cassandra.metric-failures", 1)

		return 0, err
	}
	stats.StatsdClient.Incr("writer.cassandra.metric-writes", 1)

	return 1, nil
}

func (cass *CassandraWriter) Write(stat repr.StatRepr) error {

	_, err := cass.InsertOne(stat)
	return err

}
