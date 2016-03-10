/*
	The Cassandra Index Writer/Reader

	The table should have this schema to match the repr item

		keyspace: base keyspace name (default: metric)
		path_table: base table name (default: path)
		segment_table: base table name (default: segment)
		write_consistency: "one"
		read_consistency: "one"
		port: 9042
		numcons: 5  (connection pool size)
		timeout: "30s"
		user: ""
		pass: ""


*/

package indexer

import (
	"regexp"
)

import (
	"consthash/server/dispatch"
	"consthash/server/lrucache"
	"consthash/server/repr"
	stats "consthash/server/stats"
	"consthash/server/writers/dbs"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"strings"
	"sync"
	"time"
)

const CASSANDRA_RESULT_CACHE_SIZE = 1024 * 1024 * 100
const CASSANDRA_RESULT_CACHE_TTL = 10 * time.Second

/** Being Cassandra we need some mappings to match the schemas **/

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
		has_data bool,
		PRIMARY KEY (segment, path, has_data)
*/
type CassPath struct {
	Segment CassSegment
	Path    string
	Length  int
	Hasdata bool
}

/****************** Writer *********************/
type CassandraIndexer struct {
	db   *dbs.CassandraDB
	conn *gocql.Session

	write_list     []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	max_write_size int             // size of that buffer before a flush
	max_idle       time.Duration   // either max_write_size will trigger a write or this time passing will
	write_lock     sync.Mutex
	paths_inserted map[string]bool // just to not do "extra" work on paths we've already indexed

	write_queue      chan dispatch.IJob
	dispatch_queue   chan chan dispatch.IJob
	write_dispatcher *dispatch.Dispatch

	log *logging.Logger

	findcache *lrucache.TTLLRUCache
}

func NewCassandraIndexer() *CassandraIndexer {
	cass := new(CassandraIndexer)
	cass.log = logging.MustGetLogger("indexer.cassandra")
	cass.paths_inserted = make(map[string]bool)
	cass.findcache = lrucache.NewTTLLRUCache(CASSANDRA_RESULT_CACHE_SIZE, CASSANDRA_RESULT_CACHE_TTL)
	return cass
}

func (cass *CassandraIndexer) Config(conf map[string]interface{}) (err error) {
	gots := conf["dsn"]
	if gots == nil {
		return fmt.Errorf("Indexer: `dsn` (server1,server2,server3) is needed for cassandra config")
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

func (cass *CassandraIndexer) WriteOne(skey string) error {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("indexer.cassandra.write.path-time-ns"), time.Now())
	stats.StatsdClientSlow.Incr("indexer.cassandra.noncached-writes-path", 1)

	s_parts := strings.Split(skey, ".")
	p_len := len(s_parts)

	// we are going to assume that if the path is already in the system, we've indexed it and therefore
	// do not need to do the super loop (which is very expensive)
	SelQ := fmt.Sprintf(
		"SELECT path, length, has_data FROM %s WHERE segment={pos: ?, segment: ?}",
		cass.db.PathTable(),
	)

	var _pth string
	var _len int
	var _dd bool
	gerr := cass.conn.Query(SelQ,
		p_len-1, skey,
	).Scan(&_pth, &_len, &_dd)
	// got it
	if gerr == nil {
		if _pth == skey && _dd && _len == p_len-1 {
			return nil
		}
	}
	//	cass.log.Notice("Indexer pre-get check fail, on to indexing ... : '%s'", gerr)

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
			stats.StatsdClientSlow.Incr("indexer.cassandra.segment-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.cassandra.segment-writes", 1)
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
		consthash.zipperwork.local.writer.cassandra -> consthash.zipperwork.local.writer.cassandra.write
		consthash.zipperwork.local.writer.cassandra.write -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns

		as data-less nodes

		*/
		if skey != seg.Segment && idx < len(paths)-2 {
			Q = fmt.Sprintf(
				"INSERT INTO %s (segment, path, length, has_data) VALUES  ({pos: ?, segment: ?}, ?, ?, ?)",
				cass.db.PathTable(),
			)
			err = cass.conn.Query(Q,
				seg.Pos, seg.Segment, seg.Segment+"."+s_parts[idx+1], seg.Pos+1, false,
			).Exec()
			//cass.log.Critical("NODATA:: Seg INS: %s PATH: %s Len: %d", seg.Segment, seg.Segment+"."+s_parts[idx+1], seg.Pos)
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
			consthash.zipperwork.local.writer.cassandra.write.metric-time-ns -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99

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
		//cass.log.Critical("DATA:: Seg INS: %s PATH: %s Len: %d", seg.Segment, skey, p_len-1)

		if err != nil {
			cass.log.Error("Could not insert path %v :: %v", last_path, err)
			stats.StatsdClientSlow.Incr("indexer.cassandra.path-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.cassandra.path-writes", 1)
		}
	}
	return nil
}

/*
func (cass *CassandraIndexer) writeLoop() {
	for {
		select {
		case key := <-cass.write_queue:
			cass.WriteOne(key)
		}
	}
	return
}
*/

// keep an index of the stat keys and their fragments so we can look up
func (cass *CassandraIndexer) Write(skey string) error {

	cass.write_lock.Lock()

	if _, ok := cass.paths_inserted[skey]; ok {
		stats.StatsdClientSlow.Incr("indexer.cassandra.cached-writes-path", 1)
		cass.write_lock.Unlock()
		return nil
	}
	cass.paths_inserted[skey] = true

	cass.write_lock.Unlock()

	if cass.write_queue == nil {
		workers := 128
		cass.write_queue = make(chan dispatch.IJob, 100000)
		cass.dispatch_queue = make(chan chan dispatch.IJob, workers)
		cass.write_dispatcher = dispatch.NewDispatch(workers, cass.dispatch_queue, cass.write_queue)
		cass.write_dispatcher.Run()
	}
	cass.write_queue <- CassandraIndexerJob{Cass: cass, Stat: skey}

	return nil
	/*
		if cass.write_queue == nil {
			cass.write_queue = make(chan string, 10000)
			for i := 0; i < cass.db.Cluster().NumConns; i++ {
				go cass.writeLoop()
			}
		}
		cass.write_queue <- skey
		return nil
	*/

}

/** reader methods **/

func (cass *CassandraIndexer) ExpandNonRegex(metric string) (MetricExpandItem, error) {
	paths := strings.Split(metric, ".")
	m_len := len(paths)
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=? AND segment=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		m_len-1, metric,
	).Iter()

	var on_pth string

	var me MetricExpandItem
	// just grab the "n+1" length ones
	for iter.Scan(&on_pth) {
		me.Results = append(me.Results, on_pth)
	}
	if err := iter.Close(); err != nil {
		return me, err
	}
	return me, nil
}

// Expand simply pulls out any regexes into full form
func (cass *CassandraIndexer) Expand(metric string) (MetricExpandItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.expand.get-time-ns", time.Now())

	has_reg := regexp.MustCompile(`\*|\{|\}|\[|\]`)
	needs_regex := has_reg.Match([]byte(metric))
	if !needs_regex {
		return cass.ExpandNonRegex(metric)
	}
	paths := strings.Split(metric, ".")
	m_len := len(paths)

	var me MetricExpandItem

	// convert the "graphite regex" into something golang understands (just the "."s really)
	regable := strings.Replace(metric, ".", "\\.", 0)
	the_reg, err := regexp.Compile(regable)
	if err != nil {
		return me, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		m_len-1,
	).Iter()

	var seg string
	// just grab the "n+1" length ones
	for iter.Scan(&seg) {
		//cass.log.Debug("SEG: %s", seg)

		if !the_reg.Match([]byte(seg)) {
			continue
		}
		me.Results = append(me.Results, seg)
	}
	if err := iter.Close(); err != nil {
		return me, err
	}
	return me, nil

}

// basic find for non-regex items
func (cass *CassandraIndexer) FindNonRegex(metric string) (MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.findnoregex.get-time-ns", time.Now())

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone down

	paths := strings.Split(metric, ".")
	m_len := len(paths)

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes
	cass_Q := fmt.Sprintf(
		"SELECT path,length,has_data FROM %s WHERE segment={pos: ?, segment: ?} ",
		cass.db.PathTable(),
	)
	iter := cass.conn.Query(cass_Q,
		m_len-1, metric,
	).Iter()

	var mt MetricFindItems
	var ms MetricFindItem
	var on_pth string
	var pth_len int
	var has_data bool

	// just grab the "n+1" length ones
	for iter.Scan(&on_pth, &pth_len, &has_data) {
		if pth_len > m_len {
			continue
		}
		//cass.log.Critical("NON REG:::::PATH %s LEN %d m_len: %d", on_pth, pth_len, m_len)
		spl := strings.Split(on_pth, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = on_pth
		ms.Path = on_pth

		if has_data {
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
		} else {
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
		}

		mt = append(mt, ms)
	}

	if err := iter.Close(); err != nil {
		return mt, err
	}

	return mt, nil
}

// special case for "root" == "*" finder
func (cass *CassandraIndexer) FindRoot() (MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.findroot.get-time-ns", time.Now())

	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		0,
	).Iter()

	var mt MetricFindItems
	var seg string

	for iter.Scan(&seg) {
		var ms MetricFindItem
		ms.Text = seg
		ms.Id = seg
		ms.Path = seg

		ms.Expandable = 1
		ms.Leaf = 0
		ms.AllowChildren = 1

		mt = append(mt, ms)
	}

	if err := iter.Close(); err != nil {
		return mt, err

	}

	return mt, nil
}

// to allow for multiple targets
func (cass *CassandraIndexer) Find(metric string) (MetricFindItems, error) {
	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide regex abilities
	// on the server side
	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.find.get-time-ns", time.Now())

	// special case for "root" == "*"

	if metric == "*" {
		return cass.FindRoot()
	}

	paths := strings.Split(metric, ".")
	m_len := len(paths)

	//if the last fragment is "*" then we realy mean just then next level, not another "." level
	// this is the graphite /find?query=consthash.zipperwork.local which will mean the same as
	// /find?query=consthash.zipperwork.local.* for us
	if paths[len(paths)-1] == "*" {
		metric = strings.Join(paths[:len(paths)-1], ".")
		paths = strings.Split(metric, ".")
		m_len = len(paths)
	}

	has_reg := regexp.MustCompile(`\*|\{|\}|\[|\]`)
	needs_regex := has_reg.Match([]byte(metric))

	//cass.log.Debug("HasReg: %v Metric: %s", needs_regex, metric)
	if !needs_regex {
		return cass.FindNonRegex(metric)
	}

	// convert the "graphite regex" into something golang understands (just the "."s really)

	regable := strings.Replace(metric, ".", "\\.", 0)
	the_reg, err := regexp.Compile(regable)

	if err != nil {
		return nil, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		m_len-1,
	).Iter()

	var mt MetricFindItems

	var seg string
	// just grab the "n+1" length ones
	for iter.Scan(&seg) {
		//cass.log.Debug("REG:::::PATH %s", seg)

		if !the_reg.Match([]byte(seg)) {
			continue
		}
		items, err := cass.FindNonRegex(seg)
		if err != nil {
			cass.log.Warning("could not get segments for `%s` :: %v", seg, err)
			continue
		}
		if items != nil && len(items) > 0 {
			for _, ms := range items {
				mt = append(mt, ms)
			}
			items = nil
		}
	}

	return mt, nil
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type CassandraIndexerJob struct {
	Cass *CassandraIndexer
	Stat string
}

func (j CassandraIndexerJob) DoWork() {
	j.Cass.WriteOne(j.Stat)
}
