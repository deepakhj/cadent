/*
Copyright 2016 Under Armour, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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



	a "brief" schema ..

	CREATE TYPE metric.segment_pos (
    		pos int,
    		segment text
	);

	CREATE TABLE metric.segment (
   		pos int,
   		segment text,
   		PRIMARY KEY (pos, segment)
	) WITH COMPACT STORAGE AND CLUSTERING ORDER BY (segment ASC)


	CREATE TABLE metric.path (
    		segment frozen<segment_pos>,
    		path text,
    		length int,
    		has_data boolean,
 		id varchar,  # repr.StatName.UniqueIDString()
  		PRIMARY KEY ((segment, length), path, id)
	) WITH CLUSTERING ORDER BY (path ASC)

	CREATE INDEX ON metric.path (id);

	CREATE TABLE metric.tag (
    		id varchar,  # see repr.StatName.UniqueId()
    		tags list<text>  # this will be a [ "name=val", "name=val", ...] so we can do `IN "moo=goo" in tag`
    		PRIMARY KEY (id)
	);
	CREATE INDEX ON metric.tag (tags);

	# an index of tags basically to do name="{regex}" things
	# get a list of name=values and then Q the metric.tag for id lists
	CREATE TABLE metric.tag_list (
		name text
    		value text
    		PRIMARY KEY (name)
	);
	CREATE INDEX ON metric.tag_list (value);


*/

package indexer

import (
	"cadent/server/dispatch"
	"cadent/server/repr"
	stats "cadent/server/stats"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	CASSANDRA_INDEXER_QUEUE_LEN = 1024 * 1024
	CASSANDRA_INDEXER_WORKERS   = 128
	CASSANDRA_WRITES_PER_SECOND = 1000
)

/****************** Writer *********************/
type CassandraIndexer struct {
	db        *dbs.CassandraDB
	conn      *gocql.Session
	indexerId string

	writeList    []repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	maxWriteSize int             // size of that buffer before a flush
	maxIdle      time.Duration   // either maxWriteSize will trigger a write or this time passing will
	writeLock    sync.Mutex
	numWorkers   int
	queueLen     int
	shutitdown   uint32 //shtdown notice
	startstop    utils.StartStop

	writeQueue      chan dispatch.IJob
	dispatchQueue   chan chan dispatch.IJob
	writeDispatcher *dispatch.Dispatch

	cache           *Cacher // simple cache to rate limit and buffer writes
	writesPerSecond int     // rate limit writer

	log *logging.Logger

	tagCache   *TagCache
	indexCache *IndexReadCache

	// general fix queries strings
	selectPathQ    string
	insertPathQ    string
	selectSegmentQ string
}

func NewCassandraIndexer() *CassandraIndexer {
	cass := new(CassandraIndexer)
	cass.log = logging.MustGetLogger("indexer.cassandra")
	cass.indexCache = NewIndexCache(10000)
	cass.tagCache = NewTagCache()
	atomic.SwapUint32(&cass.shutitdown, 0)
	return cass
}

func (cass *CassandraIndexer) Start() {
	cass.startstop.Start(func() {
		cass.log.Notice("starting up cassandra indexer: %s", cass.Name())
		cass.log.Notice("Adding index tables ...")

		schems := NewCassandraIndexerSchema(cass.conn, cass.db.Keyspace(), cass.db.PathTable(), cass.db.SegmentTable())
		err := schems.AddIndexerTables()
		if err != nil {
			panic(err)
		}

		workers := cass.numWorkers
		cass.writeQueue = make(chan dispatch.IJob, cass.queueLen)
		cass.dispatchQueue = make(chan chan dispatch.IJob, workers)
		cass.writeDispatcher = dispatch.NewDispatch(workers, cass.dispatchQueue, cass.writeQueue)
		cass.writeDispatcher.SetRetries(2)
		cass.writeDispatcher.Run()

		cass.cache.Start() //start cacher

		// make the nice query strings
		cass.selectPathQ = fmt.Sprintf(
			"SELECT id,path,length,has_data FROM %s WHERE segment={pos: ?, segment: ?} ",
			cass.db.PathTable(),
		)

		cass.insertPathQ = fmt.Sprintf(
			"INSERT INTO %s (segment, path, id, length, has_data) VALUES  ({pos: ?, segment: ?}, ?, ?, ?, ?)",
			cass.db.PathTable(),
		)

		cass.selectSegmentQ = fmt.Sprintf(
			"SELECT segment FROM %s WHERE pos=? AND segment=?",
			cass.db.SegmentTable(),
		)

		go cass.sendToWriters() // the dispatcher
	})
}

func (cass *CassandraIndexer) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		if atomic.SwapUint32(&cass.shutitdown, 1) == 1 {
			return // already did
		}
		cass.log.Notice("shutting down cassandra indexer: %s", cass.Name())

		cass.cache.Stop()
		if cass.writeQueue != nil {
			cass.writeDispatcher.Shutdown()
		}
	})
}

func (cass *CassandraIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Indexer: `dsn` (server1,server2,server3) is needed for cassandra config")
	}
	cass.indexerId = fmt.Sprintf(
		"%v:%v/%v|%v|%v",
		dsn,
		conf.Int64("port", 9042),
		conf.String("keyspace", "metric"),
		conf.String("path_table", "path"),
		conf.String("segment_table", "segment"),
	)

	cass.log.Notice("Connecting Indexer to Cassandra %s", cass.indexerId)
	db, err := dbs.NewDB("cassandra", cass.indexerId, conf)
	if err != nil {
		return err
	}
	cass.db = db.(*dbs.CassandraDB)
	cass.conn = db.Connection().(*gocql.Session)

	// tweak queues and worker sizes
	cass.numWorkers = int(conf.Int64("write_workers", CASSANDRA_INDEXER_WORKERS))
	cass.queueLen = int(conf.Int64("writeQueueLength", CASSANDRA_INDEXER_QUEUE_LEN))

	c_key := "indexer:cassandra:" + cass.indexerId
	cass.cache, err = getCacherSingleton(c_key)
	if err != nil {
		return err
	}
	cass.cache.maxKeys = int(conf.Int64("cache_index_size", CACHER_METRICS_KEYS))
	cass.writesPerSecond = int(conf.Int64("writesPerSecond", CASSANDRA_WRITES_PER_SECOND))

	return nil
}

func (cass *CassandraIndexer) Name() string {
	return cass.indexerId
}

func (cass *CassandraIndexer) WriteOne(inname repr.StatName) error {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("indexer.cassandra.write.path-time-ns"), time.Now())
	stats.StatsdClientSlow.Incr("indexer.cassandra.noncached-writes-path", 1)

	skey := inname.Key

	// we are going to assume that if the path is already in the system, we've indexed it and therefore
	// do not need to do the super loop (which is very expensive)

	uid := inname.UniqueIdString()
	pth := NewParsedPath(skey, uid)

	/* Skip this for now
	SelQ := fmt.Sprintf(
		"SELECT path, length, has_data FROM %s WHERE segment={pos: ?, segment: ?}",
		cass.db.PathTable(),
	)
	var _pth string
	var _len int
	var _dd bool
	gerr := cass.conn.Query(SelQ, pth.Len-1, skey).Scan(&_pth, &_len, &_dd)

	// got it
	if gerr == nil {
		if _pth == skey && _dd && _len == pth.Len-1 {
			return nil
		}
	}*/

	last_path := pth.Last()
	// now to upsert them all (inserts in cass are upserts)
	for idx, seg := range pth.Segments {
		Q := fmt.Sprintf(
			"INSERT INTO %s (pos, segment) VALUES  (?, ?) ",
			cass.db.SegmentTable(),
		)
		err := cass.conn.Query(Q,
			seg.Pos, seg.Segment,
		).Exec()

		if err != nil {
			cass.log.Error("Could not insert segment %v (%s) : %v", seg, skey, err)
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

		consthash.zipperwork.local.writer.cassandra.write.metric-time-ns -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns

		as "data'ed" nodes

		*/

		if skey != seg.Segment && idx < pth.Len-1 {
			err = cass.conn.Query(
				cass.insertPathQ,
				seg.Pos, seg.Segment, seg.Segment+"."+pth.Parts[idx+1], "", seg.Pos+1, false,
			).Exec()
		} else {
			//the "raw data" path
			err = cass.conn.Query(
				cass.insertPathQ,
				seg.Pos, seg.Segment, skey, uid, pth.Len-1, true,
			).Exec()
		}

		//cass.log.Critical("DATA:: Seg INS: %s PATH: %s Len: %d", seg.Segment, skey, p_len-1)

		if err != nil {
			cass.log.Error("Could not insert path %v (%v) :: %v", last_path, uid, err)
			stats.StatsdClientSlow.Incr("indexer.cassandra.path-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.cassandra.path-writes", 1)
		}
	}
	return nil
}

// pop from the cache and send to actual writers
func (cass *CassandraIndexer) sendToWriters() error {
	// this may not be the "greatest" ratelimiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backedup
	// it will block on the writeQueue stage

	if cass.writesPerSecond <= 0 {
		cass.log.Notice("Starting indexer writer: No rate limiting enabled")
		for {
			if cass.shutitdown == 1 {
				return nil
			}
			skey := cass.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.cassandra.write.send-to-writers"), 1)
				cass.writeQueue <- &cassandraIndexerJob{Cass: cass, Stat: skey}
			}
		}
	} else {
		sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(cass.writesPerSecond))
		cass.log.Notice("Starting indexer writer: limiter every %f nanoseconds (%d writes per second)", sleep_t, cass.writesPerSecond)
		dur := time.Duration(int(sleep_t))
		for {
			if cass.shutitdown == 1 {
				return nil
			}
			skey := cass.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.cassandra.write.send-to-writers"), 1)
				cass.writeQueue <- &cassandraIndexerJob{Cass: cass, Stat: skey}
				time.Sleep(dur)
			}
		}
	}
}

// keep an index of the stat keys and their fragments so we can look up
func (cass *CassandraIndexer) Write(skey repr.StatName) error {
	return cass.cache.Add(skey)
}

/** reader methods **/

func (cass *CassandraIndexer) ExpandNonRegex(metric string) (MetricExpandItem, error) {
	paths := strings.Split(metric, ".")
	m_len := len(paths)

	iter := cass.conn.Query(cass.selectSegmentQ, m_len-1, metric).Iter()

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

	needs_regex := needRegex(metric)
	//cass.log.Debug("REGSS: %v, %s", needs_regex, metric)

	if !needs_regex {
		return cass.ExpandNonRegex(metric)
	}
	paths := strings.Split(metric, ".")
	m_len := len(paths)

	var me MetricExpandItem

	the_reg, err := regifyKey(metric)

	if err != nil {
		return me, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	//cass.log.Debug("POSPOSPOS: %s", m_len-1)

	iter := cass.conn.Query(cass_Q,
		m_len-1,
	).Iter()

	var seg string
	// just grab the "n+1" length ones
	for iter.Scan(&seg) {
		//cass.log.Debug("SEG: %s", seg)

		if !the_reg.MatchString(seg) {
			continue
		}
		me.Results = append(me.Results, seg)
	}
	if err := iter.Close(); err != nil {
		return me, err
	}
	return me, nil

}

func (cass *CassandraIndexer) List(has_data bool, page int) (MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.list.get-time-ns", time.Now())

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone down

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes

	//cassandra does "auto pagination" but we really do not want to send back all the rows
	// so we instead need to walk the itterator
	cass_Q := fmt.Sprintf(
		"SELECT id,path FROM %s WHERE has_data=? ALLOW FILTERING",
		cass.db.PathTable(),
	)
	iter := cass.conn.Query(cass_Q, has_data).PageSize(MAX_PER_PAGE).Iter()

	var mt MetricFindItems
	var ms MetricFindItem
	var on_pth string
	var id string

	cur_page := 0
	// just grab the "n+1" length ones
	for iter.Scan(&id, &on_pth) {

		if iter.WillSwitchPage() {
			cur_page += 1
		}
		if cur_page < page {
			continue
		}

		//cass.log.Critical("NON REG:::::PATH %s LEN %d m_len: %d", on_pth, pth_len, m_len)
		spl := strings.Split(on_pth, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = on_pth
		ms.Path = on_pth

		ms.Expandable = 0
		ms.Leaf = 1
		ms.AllowChildren = 0
		ms.UniqueId = id

		mt = append(mt, ms)
	}

	if err := iter.Close(); err != nil {
		return mt, err
	}

	return mt, nil
}

// basic find for non-regex items
func (cass *CassandraIndexer) FindNonRegex(metric string, tags repr.SortingTags, exact bool) (MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.findnoregex.get-time-ns", time.Now())

	// check cache
	items := cass.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.cassandra.findnoregex.cached", 1)
		return *items, nil
	}

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone down

	paths := strings.Split(metric, ".")
	m_len := len(paths)

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes
	cass_Q := fmt.Sprintf(
		"SELECT id,path,length,has_data FROM %s WHERE segment={pos: ?, segment: ?} ",
		cass.db.PathTable(),
	)

	if exact {
		cass_Q += " LIMIT 1"
	}

	iter := cass.conn.Query(cass_Q, m_len-1, metric).Iter()

	var mt MetricFindItems
	var ms MetricFindItem
	var on_pth string
	var pth_len int
	var id string
	var has_data bool

	for iter.Scan(&id, &on_pth, &pth_len, &has_data) {
		if pth_len > m_len {
			continue
		}

		// we are looking to see if this is a "data" node or "expandable"
		if exact {
			ms.Text = paths[len(paths)-1]
			ms.Id = metric
			ms.Path = metric
			if on_pth == metric && has_data {
				ms.Expandable = 0
				ms.Leaf = 1
				ms.AllowChildren = 0
				ms.UniqueId = id

				// grab ze tags
				ms.Tags, ms.MetaTags, _ = cass.GetTagsByUid(id)
				mt = append(mt, ms)
			} else {
				ms.Expandable = 1
				ms.Leaf = 0
				ms.AllowChildren = 1
				mt = append(mt, ms)
			}
			iter.Close()
			cass.indexCache.Add(metric, tags, &mt)
			return mt, nil
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
			ms.UniqueId = id
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

	// set it
	cass.indexCache.Add(metric, tags, &mt)

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
		ms.UniqueId = ""
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
func (cass *CassandraIndexer) Find(metric string, tags repr.SortingTags) (MetricFindItems, error) {
	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide regex abilities
	// on the server side

	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.find.get-time-ns", time.Now())

	// special case for "root" == "*"
	// check cache
	items := cass.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.cassandra.find.cached", 1)
		return *items, nil
	}

	if metric == "*" {
		return cass.FindRoot()
	}

	needs_regex := needRegex(metric)

	//cass.log.Debug("HasReg: %v Metric: %s", needs_regex, metric)
	if !needs_regex {
		return cass.FindNonRegex(metric, tags, true)
	}

	paths := strings.Split(metric, ".")
	m_len := len(paths)

	// convert the "graphite regex" into something golang understands (just the "."s really)
	// need to replace things like "moo*" -> "moo.*" but careful not to do "..*"
	// the "graphite" globs of {moo,goo} we can do with (moo|goo) so convert { -> (, , -> |, } -> )
	the_reg, err := regifyKey(metric)

	if err != nil {
		return nil, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q, m_len-1).PageSize(MAX_PER_PAGE).Iter()

	var mt MetricFindItems

	var seg string
	// just grab the "n+1" length ones
	for iter.Scan(&seg) {
		//cass.log.Debug("REG:::::PATH %s :: REG: %s MATCH %v", seg, regable, the_reg.Match([]byte(seg)))

		if !the_reg.MatchString(seg) {
			continue
		}
		items, err := cass.FindNonRegex(seg, tags, true)
		if err != nil {
			cass.log.Warning("could not get segments for `%s` :: %v", seg, err)
			continue
		}
		if items != nil && len(items) > 0 {
			mt = append(mt, items...)
		}
	}
	cass.indexCache.Add(metric, tags, &mt)

	if err := iter.Close(); err != nil {
		return mt, err
	}

	return mt, nil
}

// delete a path from the index
// TODO
func (cass *CassandraIndexer) Delete(name *repr.StatName) (err error) {
	return nil
	/*
		// just remove the
		cass_Q := fmt.Sprintf(
			"DELETE FROM %s WHERE segment={pos: ?, segment: ?} AND has_data=1 AND length=? AND id=?",
			cass.db.PathTable(),
		)

		err = cass.conn.Query(cass_Q,
			m_len-1, metric,
		).Exec()
		return err
	*/
}

/*************** TAG STUBS ************************/

func (my *CassandraIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {
	return tags, metatags, errNotYetImplimented
}

func (my *CassandraIndexer) GetTagsByName(name string, page int) (tags MetricTagItems, err error) {
	return tags, errNotYetImplimented
}

func (my *CassandraIndexer) GetTagsByNameValue(name string, value string, page int) (tags MetricTagItems, err error) {
	return tags, errNotYetImplimented
}

func (my *CassandraIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	return uids, errNotYetImplimented
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type cassandraIndexerJob struct {
	Cass  *CassandraIndexer
	Stat  repr.StatName
	retry int
}

func (j *cassandraIndexerJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j *cassandraIndexerJob) OnRetry() int {
	return j.retry
}

func (j *cassandraIndexerJob) DoWork() error {
	err := j.Cass.WriteOne(j.Stat)
	if err != nil {
		j.Cass.log.Error("Insert failed for Index: %v retrying ...", j.Stat)
	}
	return err
}
