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
	"consthash/server/repr"
	stats "consthash/server/stats"
	"fmt"
	"github.com/gocql/gocql"
	logging "github.com/op/go-logging"

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
		PRIMARY KEY (segment, path)
*/
type CassPath struct {
	Segment CassSegment
	Path    string
	Length  int
}

/****************** Writer *********************/
type CassandraWriter struct {
	conn              *gocql.Session
	keyspace          string
	metric_table      string
	path_table        string
	segment_table     string
	read_consistency  gocql.Consistency
	write_consistency gocql.Consistency

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

	port := 9042
	_port := conf["port"]
	if _port != nil {
		port = _port.(int)
	}

	cass.keyspace = "metric"
	_keyspace := conf["keyspace"]
	if _keyspace != nil {
		cass.keyspace = _keyspace.(string)
	}

	cass.metric_table = "metric"
	_table := conf["metrics_table"]
	if _table != nil {
		cass.metric_table = _table.(string)
	}

	cass.path_table = "path"
	_ptable := conf["path_table"]
	if _ptable != nil {
		cass.path_table = _ptable.(string)
	}
	cass.segment_table = "segment"
	_stable := conf["segment_table"]
	if _stable != nil {
		cass.segment_table = _stable.(string)
	}

	w_consistency := "one"
	cass.write_consistency = gocql.LocalOne
	_wconsistency := conf["write_consistency"]
	if _wconsistency != nil {
		w_consistency = _wconsistency.(string)
		if w_consistency == "local_quorum" {
			cass.write_consistency = gocql.LocalQuorum
		} else if w_consistency == "quorum" {
			cass.write_consistency = gocql.Quorum
		}
	}

	r_consistency := "one"
	cass.read_consistency = gocql.LocalOne
	_rconsistency := conf["read_consistency"]
	if _rconsistency != nil {
		r_consistency = _rconsistency.(string)
		if r_consistency == "local_quorum" {
			cass.read_consistency = gocql.LocalQuorum
		} else if r_consistency == "quorum" {
			cass.read_consistency = gocql.Quorum
		}
	}

	timeout := time.Duration(30 * time.Second)
	_timeout := conf["timeout"]
	if _rconsistency != nil {
		ok, err := time.ParseDuration(_timeout.(string))
		if err != nil {
			return err
		}
		timeout = ok
	}

	numcons := 5
	_numcons := conf["numcons"]
	if _numcons != nil {
		numcons = _numcons.(int)
	}

	// cassandra does not like batch of large proportions .. so this is rather "not relevent
	// and we need to insert things One At A Time
	/*
		_wr_buffer := conf["batch_count"]
		if _wr_buffer == nil {
			cass.max_write_size = 100
		} else {
			// toml things generic ints are int64
			cass.max_write_size = int(_wr_buffer.(int64))
			//cassandra has a batch limit (on input queries) so we force 100
			if cass.max_write_size > 100 {
				cass.log.Warning("Batch size too large, need to reduce to 100")
				cass.max_write_size = 100
			}
		}
	*/

	_pr_flush := conf["periodic_flush"]
	cass.max_idle = time.Duration(time.Second)
	if _pr_flush != nil {
		dur, err := time.ParseDuration(_pr_flush.(string))
		if err == nil {
			cass.max_idle = dur
		} else {
			cass.log.Error("Cassandra Driver: Invalid Duration `%v`", _pr_flush)
		}
	}
	cass.log.Notice("Connecting to Cassandra (can take a bit of time) ... %s", dsn)

	servers := strings.Split(dsn, ",")
	cluster := gocql.NewCluster(servers...)
	cluster.Port = port
	cluster.Keyspace = cass.keyspace
	cluster.Consistency = cass.write_consistency
	cluster.Timeout = timeout
	cluster.NumConns = numcons
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: 3}

	// auth
	user := ""
	_user := conf["user"]
	if _numcons != nil {
		user = _user.(string)
	}
	pass := ""
	_pass := conf["pass"]
	if _numcons != nil {
		pass = _pass.(string)
	}

	if user != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: user,
			Password: pass,
		}
	}

	/*
		if *compress {
			cluster.Compressor = new(gocql.SnappyCompressor)
		}

		if *policy == "token" {
			cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
		}
	*/
	cass.conn, err = cluster.CreateSession()
	if err != nil {
		return err
	}
	cass.log.Notice("Connected to Cassandra: %v", servers)

	// Not needed go cass.PeriodFlush()

	return nil
}

func (cass *CassandraWriter) PeriodFlush() {
	for {
		time.Sleep(cass.max_idle)
		cass.Flush()
	}
	return
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
			Length:  p_len,
		}

		paths = append(paths, on_path)
	}

	// now to upsert them all (inserts in cass are upserts)
	for _, seg := range segments {
		Q := fmt.Sprintf(
			"INSERT INTO %s (pos, segment) VALUES  (?, ?) ",
			cass.segment_table,
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
	}

	for _, pth := range paths {
		/*
			segment frozen<segment_pos>,
			path text,
			length int
		*/
		Q := fmt.Sprintf(
			"INSERT INTO %s (segment, path, length) VALUES  ({pos: ?, segment: ?}, ?, ?) ",
			cass.path_table,
		)
		err := cass.conn.Query(Q,
			pth.Segment.Pos, pth.Segment.Segment, skey, p_len,
		).Exec()

		if err != nil {
			cass.log.Error("Could not insert path %v", pth)
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
		cass.metric_table,
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

func (cass *CassandraWriter) Flush() (int, error) {
	cass.write_lock.Lock()
	defer cass.write_lock.Unlock()

	l := len(cass.write_list)
	if l == 0 {
		return 0, nil
	}

	Q := "BEGIN BATCH "

	vals := []interface{}{}

	ttl := int64(0)
	for _, stat := range cass.write_list {
		if stat.TTL > 0 {
			ttl = stat.TTL
		}

		Q += fmt.Sprintf(
			"INSERT INTO %s (id, time, point) VALUES  ({path: ?, resolution: ?}, ?, {sum: ?, mean: ?, min: ?, max: ?, count: ?}) ",
			cass.metric_table,
		)
		if ttl > 0 {
			Q += fmt.Sprintf(" USING TTL %d", ttl)
		}
		Q += "\n"

		// note need to cast back from jsonFloat64
		vals = append(
			vals, stat.Key, int64(stat.Resolution), stat.Time.UnixNano(), float64(stat.Sum), float64(stat.Mean), float64(stat.Min), float64(stat.Max), stat.Count,
		)
	}

	Q += " APPLY BATCH"
	//cass.log.Debug(Q)
	//cass.log.Debug("%v", vals)
	//format all vals at once
	err := cass.conn.Query(Q, vals...).Exec()

	if err != nil {
		cass.log.Error("Cassandra Driver: insert failed, %v", err)
		return 0, err
	}
	cass.write_list = nil
	cass.write_list = []repr.StatRepr{}
	return l, nil
}

func (cass *CassandraWriter) Write(stat repr.StatRepr) error {

	// sadly can only do one at a time for cassandra
	_, err := cass.InsertOne(stat)
	return err

	/*
		if len(cass.write_list) > cass.max_write_size {
			_, err := cass.Flush()
			if err != nil {
				return err
			}
		}

		// Flush can cause double locking
		cass.write_lock.Lock()
		defer cass.write_lock.Unlock()
		cass.write_list = append(cass.write_list, stat)
		return nil
	*/
}
