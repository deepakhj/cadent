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
	The Cassandra Metric Blob Reader/Writer
	XXX TODO work in progress

	This cassandra blob writer takes one of the "series" blobs

	These blobs are stored in ram until "max_blob_chunk_size" is reached (default of 16kb)

	8kb for the "gob" blob type is about 130 metrics (at a flush window of 10s, about .3 hours
	Blobs are flushed every "max_time_in_ram" (default 1 hour)

	json are much larger (2-3x), but are more generic to other system backends (i.e. if you need
	to read from the DB from things other then cadent)

	protobuf are about the same size as the Gob ones, but since it's not a go-specific item, has more portability

	gorilla is a highly compressed format, but only supports foward in time points which for writing is
	probably a good thing

	Since these things are stored in Ram for a while, the size of the blobs can become important

	Unlike the "flat" cassandra where there are zillion writes happening, this one trades off the writes
	for ram.  So we don't need to use the crazy worker queue mechanism as adding metrics
	to the ram pools is fast enough, a separate slow processor periodically marches through the
	cache pool and flushes those items that need flushing to cassandra in a more serial fashion


	[graphite-cassandra.accumulator.writer.metrics]
	driver="cassandra"
	dsn="my.cassandra.com"
	[graphite-cassandra.accumulator.writer.metrics.options]
		keyspace="metric"
		metric_table="metric"
		path_table="path"
		segment_table="segment"
		write_consistency="one"
		read_consistency="one"
		port=9042
		cache_metric_size=102400  # the "internal carbon-like-cache" size (ram is your friend)
		cache_byte_size=1024 # number of points per metric to cache above to keep before we drop (this * cache_metric_size * 32 * 128 bytes == your better have that ram)
		cache_low_fruit_rate=0.25 # every 1/4 of the time write "low count" metrics to at least persist them
		writes_per_second=5000 # allowed insert queries per second

		series_encoding_type="protobuf" # gob, gorilla, json, protobuf .. the data binary blob type

		numcons=5  # cassandra connection pool size
		timeout="30s" # query timeout
		user: ""
		pass: ""
		write_workers=32  # dispatch workers to write
		write_queue_length=102400  # buffered queue size before we start blocking


	Schema

	CREATE TYPE metric_id (
    		uid varchar,   # repr.StatName.UniqueIDString()
    		res int  # resolution
	);

	CREATE TABLE metric (
    		id frozen<metric_id>,
    		stime bigint,
    		etime bigint,
    		points blob,
    		PRIMARY KEY (id, stime, etime)
	) WITH CLUSTER ORDERING stime DESC



*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/broadcast"
	"cadent/server/dispatch"
	"cadent/server/series"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/indexer"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	CASSANDRA_DEFAULT_RENDER_TIMEOUT   = "5s"
	CASSANDRA_DEFAULT_ROLLUP_TYPE      = "cached"
	CASSANDRA_DEFAULT_METRIC_WORKERS   = 16
	CASSANDRA_DEFAULT_METRIC_QUEUE_LEN = 1024 * 100
	CASSANDRA_DEFAULT_METRIC_RETRIES   = 2
)

/*** set up "one" real writer (per dsn) .. need just a single cassandra DB connection for all the time resoltuions

 */

// the singleton
var _CASS_WRITER_SINGLETON map[string]*CassandraWriter
var _cass_set_mutex sync.Mutex

func _get_cass_signelton(conf map[string]interface{}) (*CassandraWriter, error) {
	_cass_set_mutex.Lock()
	defer _cass_set_mutex.Unlock()
	gots := conf["dsn"]
	if gots == nil {
		return nil, fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	// unique per dns:port:keyspace:metrics_table
	keysp := conf["keyspace"]
	tbl := conf["metrics_table"]
	port := conf["port"]
	dsn := fmt.Sprintf("%v:%v:%v:%v", gots, port, keysp, tbl)

	if val, ok := _CASS_WRITER_SINGLETON[dsn]; ok {
		return val, nil
	}

	writer, err := NewCassandraWriter(conf)
	if err != nil {
		return nil, err
	}
	_CASS_WRITER_SINGLETON[dsn] = writer
	return writer, nil
}

// special onload init
func init() {
	_CASS_WRITER_SINGLETON = make(map[string]*CassandraWriter)
}

type CassandraWriter struct {
	// the writer connections for this
	db   *dbs.CassandraDB
	conn *gocql.Session

	// shutdowners
	shutitdown bool
	shutdown   chan bool

	log *logging.Logger
}

func NewCassandraWriter(conf map[string]interface{}) (*CassandraWriter, error) {
	cass := new(CassandraWriter)
	cass.log = logging.MustGetLogger("metrics.cassandra")
	cass.shutdown = make(chan bool)
	cass.shutitdown = false

	gots := conf["dsn"]
	if gots == nil {
		return nil, fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	conn_key := fmt.Sprintf("%v:%v/%v/%v", gots, conf["port"], conf["keyspace"], conf["metrics_table"])
	cass.log.Notice("Connecting Metrics to Cassandra (%s)", conn_key)

	db, err := dbs.NewDB("cassandra", conn_key, conf)
	if err != nil {
		return nil, err
	}
	// need to cast for real usage
	cass.db = db.(*dbs.CassandraDB)
	cass.conn = db.Connection().(*gocql.Session)

	if err != nil {
		return nil, err
	}

	return cass, nil
}

func (cass *CassandraWriter) InsertSeries(name *repr.StatName, timeseries series.TimeSeries) (int, error) {
	if name == nil {
		return 0, errNameIsNil
	}
	if timeseries == nil {
		return 0, errSeriesIsNil
	}
	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.insert.metric-time-ns"), time.Now())

	l := timeseries.Count()
	if l == 0 {
		return 0, nil
	}

	DO_Q := fmt.Sprintf(
		"INSERT INTO %s (mid, etime, stime, ptype, points) VALUES  ({id: ?, res: ?}, ?, ?, ?, ?)",
		cass.db.MetricTable(),
	)
	if name.TTL > 0 {
		DO_Q += fmt.Sprintf(" USING TTL %d", name.TTL)
	}
	blob, err := timeseries.MarshalBinary()
	if err != nil {
		return 0, err
	}
	err = cass.conn.Query(
		DO_Q,
		name.UniqueIdString(),
		int64(name.Resolution),
		timeseries.LastTime(),
		timeseries.StartTime(),
		series.IdFromName(timeseries.Name()),
		blob,
	).Exec()

	if err != nil {
		cass.log.Error("Cassandra Driver:Metric insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandra.insert.metric-failures", 1)
		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandra.batch.writes", 1)
	stats.StatsdClientSlow.GaugeAvg("writer.cassandra.insert.metrics-per-writes", int64(l))

	return l, nil
}

func (cass *CassandraWriter) Stop() {
	if cass.shutitdown {
		return
	}
	cass.shutitdown = true
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type CassandraBlobMetricJob struct {
	Cass  *CassandraMetric
	Ts    *TotalTimeSeries // where the point list live
	retry int
}

func (j CassandraBlobMetricJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j CassandraBlobMetricJob) OnRetry() int {
	return j.retry
}

func (j CassandraBlobMetricJob) DoWork() error {
	err := j.Cass.doInsert(j.Ts)
	return err
}

/****************** Metrics Writer *********************/
type CassandraMetric struct {
	driver            string
	resolutions       [][]int
	currentResolution int
	static_tags       repr.SortingTags
	indexer           indexer.Indexer
	writer            *CassandraWriter

	// this is for Render where we may have several caches, but only "one"
	// cacher get picked for the default render (things share the cache from writers
	// and the api render, but not all the caches, so we need to be able to get the
	// the cache singleton keys
	// `cache:series:seriesMaxMetrics:seriesEncoding:seriesMaxBytes:maxTimeInCache`
	cacherPrefix  string
	cacher        *Cacher
	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

	// if the rolluptype == cached, then we this just uses the internal RAM caches
	// otherwise if "trigger" we only have the lowest res cache, and trigger rollups on write
	rollupType string
	rollup     *RollupMetric
	doRollup   bool

	render_wg     sync.WaitGroup
	render_mu     sync.Mutex
	renderTimeout time.Duration

	// dispatch writer worker queue
	num_workers      int
	queue_len        int
	dispatch_retries int
	dispatcher       *dispatch.DispatchQueue

	shutitdown bool
	shutdown   chan bool
	startstop  utils.StartStop
}

func NewCassandraMetrics() *CassandraMetric {
	cass := new(CassandraMetric)
	cass.driver = "cassandra"
	return cass
}

func NewCassandraTriggerMetrics() *CassandraMetric {
	cass := new(CassandraMetric)
	cass.driver = "cassandra-triggered"
	return cass
}

func (cass *CassandraMetric) Config(conf map[string]interface{}) (err error) {

	// only need one real "writer DB" here as we are writing to the same metrics table
	gots, err := _get_cass_signelton(conf)
	if err != nil {
		return err
	}
	cass.writer = gots

	_dsn := conf["dsn"]
	if _dsn == nil {
		return fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	resolution := conf["resolution"]
	if resolution == nil {
		return fmt.Errorf("resolution needed for cassandra writer")
	}

	g_tag, ok := conf["tags"]
	if ok {
		cass.static_tags = repr.SortingTagsFromString(g_tag.(string))
	}

	// tweak queus and worker sizes
	_workers := conf["write_workers"]
	cass.num_workers = CASSANDRA_DEFAULT_METRIC_WORKERS
	if _workers != nil {
		cass.num_workers = int(_workers.(int64))
	}

	_qs := conf["write_queue_length"]
	cass.queue_len = CASSANDRA_DEFAULT_METRIC_QUEUE_LEN
	if _qs != nil {
		cass.queue_len = int(_qs.(int64))
	}
	_rt := conf["write_queue_retries"]
	cass.dispatch_retries = CASSANDRA_DEFAULT_METRIC_RETRIES
	if _rt != nil {
		cass.dispatch_retries = int(_rt.(int64))
	}

	rdur, err := time.ParseDuration(CASSANDRA_DEFAULT_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	cass.renderTimeout = rdur

	// rolluptype
	cass.rollupType = CASSANDRA_DEFAULT_ROLLUP_TYPE
	_rot, ok := conf["rollup_type"]
	if ok {
		cass.rollupType = _rot.(string)
	}

	_cache := conf["cache"]
	if _cache == nil {
		return errMetricsCacheRequired
	}
	cass.cacher = _cache.(*Cacher)
	cass.cacherPrefix = cass.cacher.Prefix

	if cass.rollupType == "triggered" {
		cass.driver = "cassandra-triggered" // reset the name
		cass.rollup = NewRollupMetric(cass, cass.cacher.maxBytes)
	}

	return nil
}

func (cass *CassandraMetric) Driver() string {
	return cass.driver
}

func (cass *CassandraMetric) Start() {
	cass.startstop.Start(func() {
		/**** dispatcher queue ***/
		cass.writer.log.Notice("Starting cassandra series writer for %s at %d bytes per series", cass.writer.db.MetricTable(), cass.cacher.maxBytes)
		cass.cacher.overFlowMethod = "chan" // need to force this issue
		cass.cacher.Start()

		// register the overflower
		cass.cacheOverFlow = cass.cacher.GetOverFlowChan()

		cass.shutitdown = false
		go cass.overFlowWrite()

		// if the resolutions list is just "one" there is no triggered rollups
		if len(cass.resolutions) == 1 {
			cass.rollupType = "cached"
		}
		cass.doRollup = cass.rollupType == "triggered" && cass.currentResolution == cass.resolutions[0][0]
		// start the rollupper if needed
		if cass.doRollup {
			// all but the lowest one
			cass.rollup.blobMaxBytes = cass.cacher.maxBytes
			cass.rollup.SetResolutions(cass.resolutions[1:])
			go cass.rollup.Start()
		}

		//start up the dispatcher
		cass.dispatcher = dispatch.NewDispatchQueue(
			cass.num_workers,
			cass.queue_len,
			cass.dispatch_retries,
		)
		cass.dispatcher.Start()
	})
}

func (cass *CassandraMetric) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		cass.writer.log.Warning("Starting Shutdown of cassandra series writer")

		if cass.shutitdown {
			return // already did
		}
		cass.shutitdown = true

		cass.cacher.Stop()

		mets := cass.cacher.Cache
		mets_l := len(mets)
		cass.writer.log.Warning("Shutting down %s and exhausting the queue (%d items) and quiting", cass.cacher.Name, mets_l)

		// full tilt write out
		go_do := make(chan *TotalTimeSeries, 16)
		done := make(chan bool, 1)
		go func() {
			for {
				select {
				case s, more := <-go_do:
					if !more {
						return
					}
					stats.StatsdClient.Incr(fmt.Sprintf("writer.cache.shutdown.send-to-writers"), 1)
					cass.writer.InsertSeries(s.Name, s.Series)
					if cass.doRollup {
						cass.rollup.DoRollup(s)
					}
				case <-done:
					return
				}
			}
		}()

		did := 0
		for _, queueitem := range mets {
			if did%100 == 0 {
				cass.writer.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
			}
			if queueitem.Series != nil {
				go_do <- &TotalTimeSeries{Name: queueitem.Name, Series: queueitem.Series}
			}
			did++
		}
		close(done)
		close(go_do)
		cass.writer.log.Warning("shutdown purge: written %d/%d...", did, mets_l)

		if cass.doRollup {
			cass.rollup.Stop()
		}
		if cass.dispatcher != nil {
			cass.dispatcher.Stop()
		}

		cass.writer.log.Warning("Shutdown finished ... quiting cassandra series writer")
		return
	})
}

func (cass *CassandraMetric) SetIndexer(idx indexer.Indexer) error {
	cass.indexer = idx
	return nil
}

// Resolutions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (cass *CassandraMetric) SetResolutions(res [][]int) int {
	cass.resolutions = res
	return len(res) // need as many writers as bins
}

func (cass *CassandraMetric) SetCurrentResolution(res int) {
	cass.currentResolution = res
}

func (cass *CassandraMetric) doInsert(ts *TotalTimeSeries) error {
	stats.StatsdClientSlow.Incr("writer.cassandra.consume.add", 1)
	_, err := cass.writer.InsertSeries(ts.Name, ts.Series)
	if err == nil && cass.doRollup {
		cass.rollup.Add(ts)
	}
	return err
}

// listen to the overflow chan from the cache and attempt to write "now"
func (cass *CassandraMetric) overFlowWrite() {
	for {
		statitem, more := <-cass.cacheOverFlow.Ch
		if !more {
			return
		}
		stats.StatsdClientSlow.Incr("writer.cassandra.queue.add", 1)
		cass.dispatcher.Add(&CassandraBlobMetricJob{Cass: cass, Ts: statitem.(*TotalTimeSeries)})
	}
}

// simple proxy to the cacher
func (cass *CassandraMetric) Write(stat repr.StatRepr) error {
	if cass.shutitdown {
		return nil
	}
	stat.Name.MergeMetric2Tags(cass.static_tags)
	// only need to do this if the first resolution
	if cass.currentResolution == cass.resolutions[0][0] {
		cass.indexer.Write(stat.Name)
	}
	if cass.rollupType == "triggered" {
		if cass.currentResolution == cass.resolutions[0][0] {
			return cass.cacher.Add(&stat.Name, &stat)
		}
	} else {
		return cass.cacher.Add(&stat.Name, &stat)
	}
	return nil
}

/************************ READERS ****************/

// based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
// from and to needs to be > then the TTL as well
func (cass *CassandraMetric) getResolution(from int64, to int64) uint32 {
	diff := int(math.Abs(float64(to - from)))
	n := int(time.Now().Unix())
	back_f := n - int(from)
	back_t := n - int(to)
	for _, res := range cass.resolutions {
		if diff < res[1] && back_f < res[1] && back_t < res[1] {
			return uint32(res[0])
		}
	}
	return uint32(cass.resolutions[len(cass.resolutions)-1][0])
}

func (cass *CassandraMetric) GetFromReadCache(metric string, start int64, end int64) (rawd *RawRenderItem, got bool) {

	// check read cache
	r_cache := GetReadCache()
	if r_cache == nil {
		stats.StatsdClient.Incr("reader.cassandra.render.cache.miss", 1)
		return rawd, false
	}

	t_start := time.Unix(int64(start), 0)
	t_end := time.Unix(int64(end), 0)
	cached_stats, _, _ := r_cache.Get(metric, t_start, t_end)
	var d_points []RawDataPoint
	step := uint32(0)

	// the ReadCache will only have the "sum" point in the mix as that's
	// the designated cached point
	if cached_stats != nil && len(cached_stats) > 0 {
		stats.StatsdClient.Incr("reader.cassandra.render.cache.hits", 1)

		f_t := uint32(0)
		for _, stat := range cached_stats {
			t := uint32(stat.Time.Unix())
			d_points = append(d_points, RawDataPoint{
				Count: 1,
				Sum:   float64(stat.Sum),
				Time:  t,
			})
			if f_t <= 0 {
				f_t = t
			}
			if step <= 0 && f_t >= 0 {
				step = t - f_t
			}
		}
		rawd.AggFunc = repr.GuessReprValueFromKey(metric)
		rawd.RealEnd = d_points[len(d_points)-1].Time
		rawd.RealStart = d_points[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.RealEnd + step
		rawd.Metric = metric
		rawd.Step = step
		rawd.Data = d_points
		return rawd, len(d_points) > 0
	} else {
		stats.StatsdClient.Incr("reader.cassandra.render.cache.miss", 1)
	}

	return rawd, false
}

// grab the time series from the DBs
func (cass *CassandraMetric) GetFromDatabase(metric *indexer.MetricFindItem, resolution uint32, start int64, end int64) (rawd *RawRenderItem, err error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.database.get-time-ns", time.Now())
	rawd = new(RawRenderItem)

	Q := fmt.Sprintf(
		"SELECT ptype, points FROM %s WHERE mid={id: ?, res: ?} AND etime >= ? AND etime <= ?",
		cass.writer.db.MetricTable(),
	)

	// times need to be in Nanos, but comming as a epoch
	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nano_end := end * nano
	nano_start := start * nano

	iter := cass.writer.conn.Query(
		Q,
		metric.UniqueId, resolution, nano_start, nano_end,
	).Iter()

	// cass.writer.log.Debug("Select Q for %s: %s (%v, %v, %v, %v)", metric.Id, Q, metric.UniqueId, resolution, nano_start, nano_end)

	// for each "series" we get make a list of points
	u_start := uint32(start)
	u_end := uint32(end)
	rawd.Start = u_start
	rawd.End = u_end
	rawd.Id = metric.UniqueId
	rawd.Metric = metric.Path

	rawd.AggFunc = repr.GuessReprValueFromKey(metric.Id)
	var p_type uint8
	var p_bytes []byte

	for iter.Scan(&p_type, &p_bytes) {
		s_name := series.NameFromId(p_type)
		s_iter, err := series.NewIter(s_name, p_bytes)
		if err != nil {
			return rawd, err
		}

		for s_iter.Next() {
			to, mi, mx, ls, su, ct := s_iter.Values()

			t := uint32(time.Unix(0, to).Unix())

			// skip if not in range
			if t > u_end || t < u_start {
				continue
			}

			rawd.Data = append(rawd.Data, RawDataPoint{
				Count: ct,
				Sum:   su,
				Max:   mx,
				Min:   mi,
				Last:  ls,
				Time:  t,
			})

			if rawd.RealEnd < t {
				rawd.RealEnd = t
			}
			if rawd.RealStart > t || rawd.RealStart == 0 {
				rawd.RealStart = t
			}
		}
		if s_iter.Error() != nil {
			return rawd, s_iter.Error()
		}
	}

	if err := iter.Close(); err != nil {
		cass.writer.log.Error("Database: Failure closing iterator: %s: %v", Q, err)
	}

	return rawd, nil

}

func (cass *CassandraMetric) GetFromWriteCache(metric *indexer.MetricFindItem, start uint32, end uint32, resolution uint32) (*RawRenderItem, error) {

	// grab data from the write inflight cache
	// need to pick the "proper" cache
	cache_db := fmt.Sprintf("%s:%d", cass.cacherPrefix, resolution)
	if cass.rollupType == "triggered" {
		cache_db = fmt.Sprintf("%s:%d", cass.cacherPrefix, cass.resolutions[0][0])
	}

	use_cache := GetCacherByName(cache_db)
	if use_cache == nil {
		use_cache = cass.cacher
	}
	inflight, err := use_cache.GetAsRawRenderItem(metric.StatName())

	if err != nil {
		return nil, err
	}
	if inflight == nil {
		return nil, nil
	}
	inflight.Metric = metric.Path
	inflight.Id = metric.UniqueId
	inflight.Step = resolution
	inflight.Start = start
	inflight.End = end
	inflight.Tags = metric.Tags
	inflight.MetaTags = metric.MetaTags
	return inflight, nil
}

func (cass *CassandraMetric) RawDataRenderOne(metric *indexer.MetricFindItem, start int64, end int64) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.renderraw.get-time-ns", time.Now())
	rawd := new(RawRenderItem)

	//figure out the best res
	resolution := cass.getResolution(start, end)

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	u_start := uint32(start)
	u_end := uint32(end)

	rawd.Step = resolution
	rawd.Metric = metric.Path
	rawd.Id = metric.UniqueId
	rawd.RealEnd = u_end
	rawd.RealStart = u_start
	rawd.Start = rawd.RealStart
	rawd.End = rawd.RealEnd
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.AggFunc = repr.GuessReprValueFromKey(metric.Id)

	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return rawd, errNotADataNode
	}

	b_len := (u_end - u_start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, errTimeTooSmall
	}

	inflight, err := cass.GetFromWriteCache(metric, u_start, u_end, resolution)

	// need at LEAST 2 points to get the proper step size
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// all the data we need is in the inflight
		in_range := inflight.DataInRange(u_start, u_end)

		// if all the data is in this list we don't need to go any further
		if in_range {
			// move the times to the "requested" ones and quantize the list
			inflight.RealEnd = u_end
			inflight.RealStart = u_start
			err = inflight.Quantize()
			return inflight, err
		}
	}
	if err != nil {
		cass.writer.log.Error("Cassandra: Erroring getting inflight data: %v", err)
	}

	// and now for the mysql Query otherwise
	cass_data, err := cass.GetFromDatabase(metric, resolution, start, end)
	if err != nil {
		cass.writer.log.Error("Cassandra: Error getting from DB: %v", err)
		return rawd, err
	}

	cass_data.Step = resolution
	cass_data.Start = u_start
	cass_data.End = u_end
	cass_data.Tags = metric.Tags
	cass_data.MetaTags = metric.MetaTags

	if inflight == nil {
		cass_data.Quantize()
		return cass_data, nil
	}

	if len(cass_data.Data) > 0 && len(inflight.Data) > 1 {
		inflight.Merge(cass_data)
		return inflight, nil
	}
	inflight.Quantize()
	return inflight, nil
}

// after the "raw" render we need to yank just the "point" we need from the data which
// will make the read-cache much smaller (will compress just the Mean value as the count is 1)
func (cass *CassandraMetric) RawRenderOne(metric *indexer.MetricFindItem, from int64, to int64) (*RawRenderItem, error) {
	return cass.RawDataRenderOne(metric, from, to)
}

func (cass *CassandraMetric) Render(path string, start int64, end int64) (WhisperRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.render.get-time-ns", time.Now())

	raw_data, err := cass.RawRender(path, start, end)

	if err != nil {
		return WhisperRenderItem{}, err
	}

	var whis WhisperRenderItem
	whis.Series = make(map[string][]DataPoint)
	for _, data := range raw_data {
		whis.End = data.End
		whis.Start = data.Start
		whis.Step = data.Step
		whis.RealEnd = data.RealEnd
		whis.RealStart = data.RealStart

		d_points := make([]DataPoint, 0)
		for _, d := range data.Data {
			v := d.AggValue(data.AggFunc)
			d_points = append(d_points, DataPoint{Time: d.Time, Value: &v})
		}
		whis.Series[data.Metric] = d_points
	}
	return whis, err
}

func (cass *CassandraMetric) RawRender(path string, start int64, end int64) ([]*RawRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.rawrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	for _, pth := range paths {
		mets, err := cass.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*RawRenderItem, len(metrics), len(metrics))

	// ye old fan out technique
	render_one := func(metric *indexer.MetricFindItem, idx int) {
		defer render_wg.Done()
		timeout := time.NewTimer(cass.renderTimeout)
		for {
			select {
			case <-timeout.C:
				cass.writer.log.Error("Render Timeout for %s (%s->%s)", path, start, end)
				timeout.Stop()
				return
			default:
				_ri, err := cass.RawRenderOne(metric, start, end)

				if err != nil {
					cass.writer.log.Error("Read Error for %s (%s->%s) : %v", path, start, end, err)
					return
				}
				rawd[idx] = _ri
				return
			}
		}

	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(&metric, idx)
	}
	render_wg.Wait()
	return rawd, nil
}

func (cass *CassandraMetric) CacheRender(path string, start int64, end int64, tags repr.SortingTags) (rawd []*RawRenderItem, err error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.cacherender.get-time-ns", time.Now())

	//figure out the best res
	resolution := cass.getResolution(start, end)

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	for _, pth := range paths {
		mets, err := cass.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd = make([]*RawRenderItem, len(metrics), len(metrics))

	// ye old fan out technique
	render_one := func(metric *indexer.MetricFindItem, idx int) {
		defer render_wg.Done()
		_ri, err := cass.GetFromWriteCache(metric, uint32(start), uint32(end), resolution)

		if err != nil {
			cass.writer.log.Error("Read Error for %s (%s->%s) : %v", path, start, end, err)
			return
		}
		rawd[idx] = _ri
		return
	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(&metric, idx)
	}
	render_wg.Wait()
	return rawd, nil
}

func (cass *CassandraMetric) CachedSeries(path string, start int64, end int64, tags repr.SortingTags) (series *TotalTimeSeries, err error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.seriesrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	if len(paths) > 1 {
		return series, errMultiTargetsNotAllowed
	}

	metric := &repr.StatName{Key: path}
	metric.MergeMetric2Tags(tags)
	metric.MergeMetric2Tags(cass.static_tags)

	resolution := cass.getResolution(start, end)
	cache_db := fmt.Sprintf("%s:%v", cass.cacherPrefix, resolution)
	use_cache := GetCacherByName(cache_db)
	if use_cache == nil {
		use_cache = cass.cacher
	}
	name, inflight, err := use_cache.GetSeries(metric)
	if err != nil {
		return nil, err
	}
	if inflight == nil {
		// try the the path as unique ID
		gots_int := metric.StringToUniqueId(path)
		if gots_int != 0 {
			name, inflight, err = use_cache.GetSeriesById(gots_int)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, nil
		}
	}

	return &TotalTimeSeries{Name: name, Series: inflight}, nil
}

/*************** Match the DBMetrics Interface ***********************/

// given a name get the latest metric series
func (cass *CassandraMetric) GetLatestFromDB(name *repr.StatName, resolution uint32) (DBSeriesList, error) {

	Q := fmt.Sprintf(
		"SELECT mid.id, stime, etime, ptype, points FROM %s WHERE mid={id: ?, res: ?} ORDER BY etime DESC LIMIT 1",
		cass.writer.db.MetricTable(),
	)

	iter := cass.writer.conn.Query(
		Q,
		name.UniqueIdString(), resolution,
	).Iter()

	defer iter.Close()

	rawd := make(DBSeriesList, 0)
	var p_type uint8
	var p_bytes []byte
	var uid string
	var start, end int64

	for iter.Scan(&uid, &start, &end, &p_type, &p_bytes) {
		dataums := &DBSeries{
			Uid:        uid,
			Start:      start,
			End:        end,
			Ptype:      p_type,
			Pbytes:     p_bytes,
			Resolution: resolution,
		}
		rawd = append(rawd, dataums)

	}
	if err := iter.Close(); err != nil {
		cass.writer.log.Error("Database: Failure closing iterator: %s: %v", Q, err)
	}

	return rawd, nil
}

// given a name get the latest metric series
func (cass *CassandraMetric) GetRangeFromDB(name *repr.StatName, start uint32, end uint32, resolution uint32) (DBSeriesList, error) {

	Q := fmt.Sprintf(
		"SELECT mid.id, stime, etime, ptype, points FROM %s WHERE mid={id: ?, res: ?} AND etime >= ? AND etime <= ?",
		cass.writer.db.MetricTable(),
	)
	// need to convert second time to nan time
	nano := int64(time.Second)
	nano_end := int64(end) * nano
	nano_start := int64(start) * nano

	iter := cass.writer.conn.Query(
		Q,
		name.UniqueIdString(), resolution, nano_start, nano_end,
	).Iter()

	defer iter.Close()

	rawd := make(DBSeriesList, 0)
	var p_type uint8
	var p_bytes []byte
	var uid string
	var tstart, tend int64

	for iter.Scan(&uid, &tstart, &tend, &p_type, &p_bytes) {
		dataums := &DBSeries{
			Uid:        uid,
			Start:      tstart,
			End:        tend,
			Ptype:      p_type,
			Pbytes:     p_bytes,
			Resolution: resolution,
		}
		rawd = append(rawd, dataums)

	}
	if err := iter.Close(); err != nil {
		cass.writer.log.Error("Database: Failure closing iterator: %s: %v", Q, err)
	}

	return rawd, nil
}

// update the row defined in dbs w/ the new bytes from the new Timeseries
// for cassandara we need to use the "old" start ansd end times
// as the "Uniqueid" uid, res, etime, stime
func (cass *CassandraMetric) UpdateDBSeries(dbs *DBSeries, ts series.TimeSeries) error {

	Q := fmt.Sprintf(
		"UPDATE %s SET stime=?, etime=?, ptype=?, points=? WHERE mid={id: ?, res:?} AND stime=? AND etime=?",
		cass.writer.db.MetricTable(),
	)

	ptype := series.IdFromName(ts.Name())

	blob, err := ts.MarshalBinary()
	if err != nil {
		return err
	}
	err = cass.writer.conn.Query(
		Q,
		ts.StartTime(),
		ts.LastTime(),
		ptype,
		blob,
		dbs.Uid,
		dbs.Resolution,
		dbs.Start,
		dbs.End,
	).Exec()

	return err
}

// update the row defined in dbs w/ the new bytes from the new Timeseries
func (cass *CassandraMetric) InsertDBSeries(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (added int, err error) {
	return cass.writer.InsertSeries(name, timeseries)
}
