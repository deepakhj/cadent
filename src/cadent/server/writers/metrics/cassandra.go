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

		blob_type="protobuf" # gob, gorilla, json, protobuf .. the data binary blob type

		numcons=5  # cassandra connection pool size
		timeout="30s" # query timeout
		user: ""
		pass: ""
		write_workers=32  # dispatch workers to write
		write_queue_length=102400  # buffered queue size before we start blocking


	Schema

	CREATE TYPE metric_id (
    		id varint,   # repr.StatName.UniqueID()
    		resolution int
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
	"os"
	"os/signal"
	"syscall"

	"cadent/server/dispatch"
	"cadent/server/writers/indexer"
	"cadent/server/writers/series"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	CASSANDRA_METRIC_WORKERS       = 32
	CASSANDRA_METRIC_QUEUE_LEN     = 1024 * 1024 * 10
	CASSANDRA_WRITES_PER_SECOND    = 5000
	CASSANDRA_DEFAULT_SERIES_TYPE  = "protobuf"
	CASSANDRA_DEFAULT_SERIES_CHUNK = 16 * 1024 * 1024 // 16kb
)

/*** set up "one" real writer (per dsn) .. and writer queue .. no
  no need to get multiqueues/channel/etc of these per resolution
  as we are all sharing the same DB pool and should write things in as they come in
  multiple writer pools tend to lead to bad lock contention behavior on input channels
  and output channels as well as the cassandra writer (gocql) itself.
  Having a "single" real writer for all resolutions saves all of that.

  this, and the "non-channel" Queue in "writer.go", comes from ALOT of performance testing and tuning
  while not the "go'est" way of doing things.  It works with many 100s of thousands of metrics being flushed
  one a single machine.

  We don't need to do this for the "indexer" portion of the cassandra writer, as there is only "one" instance
  of that per DSN and it also maintains it's own "hot" cache check, which after 1-3 flushes will fill up and
  basically never write anymore

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

	dsn := gots.(string)
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

/***** caching singletons (as readers need to see this as well) ***/

// the singleton
var _CASS_CACHER_SINGLETON map[string]*Cacher
var _cass_cacher_mutex sync.Mutex

func _get_cacher_signelton(nm string) (*Cacher, error) {
	_cass_cacher_mutex.Lock()
	defer _cass_cacher_mutex.Unlock()

	if val, ok := _CASS_CACHER_SINGLETON[nm]; ok {
		return val, nil
	}

	cacher := NewCacher()
	_CASS_CACHER_SINGLETON[nm] = cacher
	return cacher, nil
}

// special onload init
func init() {
	_CASS_WRITER_SINGLETON = make(map[string]*CassandraWriter)
	_CASS_CACHER_SINGLETON = make(map[string]*Cacher)
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type CassandraBlobMetricJob struct {
	Cass   *CassandraWriter
	Series series.TimeSeries // where the point list live
	Name   *repr.StatName
	retry  int
}

func (j CassandraBlobMetricJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j CassandraBlobMetricJob) OnRetry() int {
	return j.retry
}

func (j CassandraBlobMetricJob) DoWork() error {
	_, err := j.Cass.InsertSeries(j.Name, j.Series)
	return err
}

type CassandraWriter struct {
	// the writer connections for this
	db   *dbs.CassandraDB
	conn *gocql.Session

	write_list       []*repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	write_queue      chan dispatch.IJob
	dispatch_queue   chan chan dispatch.IJob
	write_dispatcher *dispatch.Dispatch
	cacher           *Cacher
	blob_series_type string

	shutdown          chan bool // when triggered, we skip the rate limiter and go full out till the queue is done
	shutitdown        bool      // just a flag
	writes_per_second int       // allowed writes per second
	num_workers       int
	queue_len         int
	max_write_size    int           // size of that buffer before a flush
	max_idle          time.Duration // either max_write_size will trigger a write or this time passing will
	write_lock        sync.Mutex
	log               *logging.Logger

	_insert_query      string //render once
	_select_time_query string //render once
	_get_query         string //render once
}

func NewCassandraWriter(conf map[string]interface{}) (*CassandraWriter, error) {
	cass := new(CassandraWriter)
	cass.log = logging.MustGetLogger("metrics.cassandra")
	cass.shutdown = make(chan bool)

	gots := conf["dsn"]
	if gots == nil {
		return nil, fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}
	dsn := gots.(string)

	db, err := dbs.NewDB("cassandra", dsn, conf)
	if err != nil {
		return nil, err
	}
	// need to cast for real usage
	cass.db = db.(*dbs.CassandraDB)
	cass.conn = db.Connection().(*gocql.Session)

	_wr_buffer := conf["batch_count"]
	if _wr_buffer == nil {
		cass.max_write_size = 50
	} else {
		// toml things generic ints are int64
		cass.max_write_size = int(_wr_buffer.(int64))
	}
	if gocql.BatchSizeMaximum < cass.max_write_size {
		cass.log.Warning("Cassandra Driver: Setting batch size to %d, as it's the largest allowed", gocql.BatchSizeMaximum)
		cass.max_write_size = gocql.BatchSizeMaximum
	}

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

	// tweak queus and worker sizes
	_workers := conf["write_workers"]
	cass.num_workers = CASSANDRA_METRIC_WORKERS
	if _workers != nil {
		cass.num_workers = int(_workers.(int64))
	}

	_qs := conf["write_queue_length"]
	cass.queue_len = CASSANDRA_METRIC_QUEUE_LEN
	if _qs != nil {
		cass.queue_len = int(_qs.(int64))
	}

	_rs := conf["writes_per_second"]
	cass.writes_per_second = CASSANDRA_WRITES_PER_SECOND
	if _rs != nil {
		cass.writes_per_second = int(_rs.(int64))
	}

	cass.blob_series_type = CASSANDRA_DEFAULT_SERIES_TYPE
	_bt := conf["blob_type"]
	if _bt != nil {
		cass.blob_series_type = _bt.(string)
	}

	cass._insert_query = fmt.Sprintf(
		"INSERT INTO %s (id, stime, etime, points) VALUES  ({id: ?, path: ?, resolution: ?}, ?, ?, ?)",
		cass.db.MetricTable(),
	)

	cass._select_time_query = fmt.Sprintf(
		"SELECT points, stime, etime FROM %s WHERE id={id: ?, path: ?, resolution: ?} AND etime <= ? and stime >= ?",
		cass.db.MetricTable(),
	)

	cass._get_query = fmt.Sprintf(
		"SELECT points, stime, etime FROM %s WHERE id={id: ?, path: ?, resolution: ?} and stime >= ? and etime <= ?",
		cass.db.MetricTable(),
	)

	go cass.TrapExit()
	return cass, nil
}

// we can use the batcher effectively for single metric multi point writes as they share the
// the same token
func (cass *CassandraWriter) InsertSeries(name *repr.StatName, series series.TimeSeries) (int, error) {

	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandraflat.batch.metric-time-ns"), time.Now())

	l := series.Count()
	if l == 0 {
		return 0, nil
	}
	batch := cass.conn.NewBatch(gocql.LoggedBatch)
	DO_Q := cass._insert_query
	if name.TTL > 0 {
		DO_Q += fmt.Sprintf(" USING TTL %d", name.TTL)
	}
	blob, err := series.MarshalBinary()
	if err != nil {
		return 0, err
	}
	batch.Query(
		DO_Q,
		name.UniqueId(),
		name.Key,
		int64(name.Resolution),
		series.StartTime(),
		series.LastTime(),
		blob,
	)

	err = cass.conn.ExecuteBatch(batch)
	if err != nil {
		cass.log.Error("Cassandra Driver:Batch Metric insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandraflat.batch.metric-failures", 1)
		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandraflat.batch.writes", 1)
	stats.StatsdClientSlow.GaugeAvg("writer.cassandraflat.batch.metrics-per-writes", int64(l))

	return l, nil
}

func (cass *CassandraWriter) Stop() {

	if cass.shutitdown {
		return // already did
	}
	cass.shutitdown = true
	cass.shutdown <- true
	cass.cacher.Stop()

	mets := cass.cacher.Queue
	mets_l := len(mets)
	cass.log.Warning("Shutting down, exhausting the queue (%d items) and quiting", mets_l)
	// full tilt write out
	did := 0
	for _, queueitem := range mets {
		if did%100 == 0 {
			cass.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
		}
		name, points, _ := cass.cacher.GetSeriesById(queueitem.metric)
		if points != nil {
			stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandra.write.send-to-writers"), 1)
			cass.InsertSeries(name, points)
		}
		did++
	}
	cass.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
	cass.log.Warning("Shutdown finished ... quiting cassandra writer")
	return
}

func (cass *CassandraWriter) Start() {
	/**** dispatcher queue ***/
	if cass.write_queue == nil {
		workers := cass.num_workers
		cass.write_queue = make(chan dispatch.IJob, cass.queue_len)
		cass.dispatch_queue = make(chan chan dispatch.IJob, workers)
		cass.write_dispatcher = dispatch.NewDispatch(workers, cass.dispatch_queue, cass.write_queue)
		cass.write_dispatcher.SetRetries(2)
		cass.write_dispatcher.Run()

		cass.cacher.Start()
		go cass.sendToWriters() // the dispatcher
	}
}

func (cass *CassandraWriter) TrapExit() {
	//trap kills to flush queues and close connections
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func(ins *CassandraWriter) {
		s := <-sc
		cass.log.Warning("Caught %s: Flushing remaining points out before quit ", s)

		cass.Stop()
		signal.Stop(sc)
		close(sc)

		// re-raise it
		process, _ := os.FindProcess(os.Getpid())
		process.Signal(s)
		return
	}(cass)
	return
}

// pop from the cache and send to actual writers
func (cass *CassandraWriter) sendToWriters() error {
	// this may not be the "greatest" rate-limiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backed-up
	// it will block on the write_queue stage

	//ye old unlimited
	if cass.writes_per_second <= 0 {
		cass.log.Notice("Starting metric writer: No Write limiter")

		for {
			if cass.shutitdown {
				return nil
			}

			_, tseries := cass.cacher.PopSeries()
			switch tseries {
			case nil:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandra.write.send-to-writers"), 1)
				cass.write_queue <- CassandraBlobMetricJob{Cass: cass, Series: tseries}
			}
		}
	} else {

		sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(cass.writes_per_second))
		cass.log.Notice("Starting metric writer: limiter every %f nanoseconds (%d writes per second)", sleep_t, cass.writes_per_second)
		dur := time.Duration(int(sleep_t))

		for {
			if cass.shutitdown {
				return nil
			}

			_, tseries := cass.cacher.PopSeries()

			switch tseries {
			case nil:
				time.Sleep(time.Second)
			default:

				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandra.write.send-to-writers"), 1)
				cass.write_queue <- CassandraBlobMetricJob{Cass: cass, Series: tseries}
				time.Sleep(dur)
			}

		}
	}
}

func (cass *CassandraWriter) Write(stat repr.StatRepr) error {

	//cache keys needs metric + resolution
	// turning off
	if !cass.shutitdown {
		cass.cacher.Add(&stat.Name, &stat)
	}

	return nil

}

/****************** Metrics Writer *********************/
type CassandraMetric struct {
	resolutions [][]int
	indexer     indexer.Indexer
	writer      *CassandraWriter
	render_wg   sync.WaitGroup
	render_mu   sync.Mutex
	shutonce    sync.Once
}

func NewCassandraMetrics() *CassandraMetric {
	cass := new(CassandraMetric)
	return cass
}

func (cass *CassandraMetric) Stop() {
	cass.shutonce.Do(cass.writer.Stop)
}

func (cass *CassandraMetric) Start() {
	cass.writer.Start()
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

func (cass *CassandraMetric) Config(conf map[string]interface{}) (err error) {
	gots, err := _get_cass_signelton(conf)
	if err != nil {
		return err
	}
	cass.writer = gots

	resolution := conf["resolution"]
	if resolution == nil {
		return fmt.Errorf("Resulotuion needed for cassandra writer")
	}
	cache_str := conf["dsn"].(string) + gots.blob_series_type
	gots.cacher, err = _get_cacher_signelton(cache_str)
	if err != nil {
		return err
	}

	// set the cacher bits
	_ms := conf["cache_metric_size"]
	if _ms != nil {
		gots.cacher.maxKeys = int(_ms.(int64))
	}

	_ps := conf["cache_byte_size"]
	if _ps != nil {
		gots.cacher.maxBytes = int(_ps.(int64))
	}

	_lf := conf["cache_low_fruit_rate"]
	if _lf != nil {
		gots.cacher.lowFruitRate = _lf.(float64)
	}

	// match blob types
	gots.cacher.seriesType = gots.blob_series_type

	cass.writer.Start() //start up

	return nil
}

// simple proxy to the cacher
func (cass *CassandraMetric) Write(stat repr.StatRepr) error {
	// write the index from the cache as indexing can be slooowwww
	// keep note of this, when things are not yet "warm" (the indexer should
	// keep tabs on what it's already indexed for speed sake,
	// the push "push" of stats will cause things to get pretty slow for a while
	cass.indexer.Write(stat.Name)
	return cass.writer.Write(stat)
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

// based on the resolution attempt to round start/end nicely by the resolutions
func (cass *CassandraMetric) truncateTo(num int64, mod int) int64 {
	_mods := int(math.Mod(float64(num), float64(mod)))
	if _mods < mod/2 {
		return num - int64(_mods)
	}
	return num + int64(mod-_mods)
}

func (cass *CassandraMetric) RawRenderOne(metric indexer.MetricFindItem, from string, to string) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.renderraw.get-time-ns", time.Now())

	rawd := new(RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("Cassandra: RawRenderOne: Not a data node")
	}

	start, err := ParseTime(from)
	if err != nil {
		cass.writer.log.Error("Invalid from time `%s` :: %v", from, err)
		return rawd, err
	}

	end, err := ParseTime(to)
	if err != nil {
		cass.writer.log.Error("Invalid from time `%s` :: %v", to, err)
		return rawd, err
	}
	if end < start {
		start, end = end, start
	}
	//figure out the best res
	resolution := cass.getResolution(start, end)

	start = cass.truncateTo(start, int(resolution))
	end = cass.truncateTo(end, int(resolution))

	b_len := uint32(end-start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, fmt.Errorf("Cassandra: RawRenderOne: time too narrow")
	}

	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nano_end := end * nano
	nano_start := start * nano

	first_t := uint32(start)
	last_t := uint32(end)

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	iter := cass.writer.conn.Query(
		cass.writer._select_time_query,
		metric.Id, resolution, nano_end, nano_start,
	).Iter()

	var t, count int64
	var min, max, sum, first, last float64

	// use mins or maxes for the "upper_xxx, lower_xxx"
	m_key := metric.Id

	var d_points []RawDataPoint

	ct := 0
	// sorting order for the table is time ASC (i.e. first_t == first entry)

	for iter.Scan(&max, &min, &sum, &first, &last, &count, &t) {
		on_t := uint32(t / nano) // back convert to seconds

		d_points = append(d_points, RawDataPoint{
			Count: count,
			Sum:   sum,
			Mean:  sum / float64(count),
			Max:   max,
			Min:   min,
			Last:  last,
			First: first,
			Time:  on_t,
		})
		//cass.log.Critical("POINT %s time:%d data:%f", metric.Id, on_t, sum)
		ct++
		last_t = on_t
	}

	if err := iter.Close(); err != nil {
		cass.writer.log.Error("RawRender: Failure closing iterator: %v", err)
	}

	if ct > 0 && d_points[0].Time > 0 {
		first_t = d_points[0].Time
	}

	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, first_t, last_t, len(d_points), ct)

	rawd.RealEnd = uint32(last_t)
	rawd.RealStart = uint32(first_t)
	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	rawd.Step = resolution
	rawd.Metric = m_key
	rawd.Data = d_points

	return rawd, nil
}

func (cass *CassandraMetric) RenderOne(metric indexer.MetricFindItem, from string, to string) (WhisperRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.renderone.get-time-ns", time.Now())

	var whis WhisperRenderItem

	rawd, err := cass.RawRenderOne(metric, from, to)

	if err != nil {
		return whis, err
	}
	whis.RealEnd = rawd.RealEnd
	whis.RealStart = rawd.RealStart
	whis.Start = rawd.Start
	whis.End = rawd.End
	whis.Step = rawd.Step
	whis.Series = make(map[string][]DataPoint)

	// which value to actually return
	use_metric := metric.SelectValue()

	m_key := metric.Id
	stat_name := metric.StatName()
	stat_name.Resolution = rawd.Step
	b_len := rawd.End - rawd.Start/rawd.Step //"proper" length of the metric

	// Since graphite does not care about the actual time stamp, but assumes
	// a "constant step" in time. Since data may not necessarily "be there" for a given
	// interval we need to "insert nils" for steps that don't really exist
	// as basically (start - end) / resolution needs to match
	// the vector length, in effect we need to "interpolate" the vector to match sizes

	// now for the interpolation bit .. basically leaving "times that have no data as nulls"
	// XXX HOPEFULLY there are usually FEWER or as much "real" data then "wanted" by the resolution
	// if there's not :boom: and you should really keep tabs on who is messing with your data in the DB
	interp_vec := make([]DataPoint, b_len)
	cur_step_time := rawd.Start
	d_points := rawd.Data
	ct := uint32(len(d_points))
	var last_got_t uint32
	var last_got_index uint32

	// grab from cache too if not yet written
	_, inflight, err := cass.writer.cacher.GetById(stat_name.UniqueId())
	// debuggers
	//cass.writer.log.Critical("%s", s_key)
	//cass.writer.cacher.DumpPoints(inflight)
	inflight_len := uint32(len(inflight))

	if ct > 0 { // got something from cassandra, make sure to fill any "missing times" w/ nils
		j := uint32(0)
		for i := uint32(0); i < b_len; i++ {

			interp_vec[i] = DataPoint{Time: cur_step_time, Value: nil}

			for ; j < ct; j++ {
				d := d_points[j]
				if d.Time <= cur_step_time {

					// cass.writer.log.Critical("ONPS %v : %v Len %d :I %d, j %d, iLen: %v SUM: %v", d_points[j], interp_vec[i], ct, i, j, b_len, d_points[j].Sum,)

					// the weird setters here are to get the pointers properly (a weird golang thing)
					switch use_metric {
					case "mean":
						m := d.Sum / float64(d.Count)
						interp_vec[i].Value = &m
					case "min":
						m := d.Min
						interp_vec[i].Value = &m
					case "max":
						m := d.Max
						interp_vec[i].Value = &m
					case "last":
						m := d.Last
						interp_vec[i].Value = &m
					case "first":
						m := d.First
						interp_vec[i].Value = &m
					default:
						s := d.Sum
						interp_vec[i].Value = &s
					}
					interp_vec[i].Time = d.Time //this is the "real" time, graphite does not care, but something might
					last_got_t = d.Time
					last_got_index = j
					j++
				}
				break
			}
			cur_step_time += rawd.Step
		}

		// now attempt to merge the in-flight data
		if len(inflight) > 0 && err == nil && last_got_t <= cur_step_time {
			j := uint32(0)
			for i := last_got_index; i < b_len; i++ {
				for ; j < inflight_len; j++ {
					d := inflight[j]
					if uint32(d.Time.Unix()) <= interp_vec[i].Time {
						// the weird setters here are to get the pointers properly (a weird golang thing)
						switch use_metric {
						case "mean":
							m := float64(d.Sum) / float64(d.Count)
							interp_vec[i].Value = &m
						case "min":
							m := float64(d.Min)
							interp_vec[i].Value = &m
						case "max":
							m := float64(d.Max)
							interp_vec[i].Value = &m
						case "last":
							m := float64(d.Last)
							interp_vec[i].Value = &m
						case "first":
							m := float64(d.First)
							interp_vec[i].Value = &m
						default:
							s := float64(d.Sum)
							interp_vec[i].Value = &s
						}
						j++
					}
					break
				}
			}
		}
	} else if len(inflight) > 0 && err == nil { // no data in cassandra yet, use inflight
		//fill it up
		j := uint32(0)
		for i := uint32(0); i < b_len; i++ {
			interp_vec[i] = DataPoint{Time: cur_step_time, Value: nil}
			for ; j < inflight_len; j++ {
				d := inflight[j]
				if uint32(d.Time.Unix()) <= cur_step_time {

					// cass.writer.log.Critical("ONPS %v : %v Len %d :I %d, j %d, iLen: %v SUM: %v", d_points[j], interp_vec[i], ct, i, j, b_len, d_points[j].Sum,)

					// the weird setters here are to get the pointers properly (a weird golang thing)
					switch use_metric {
					case "mean":
						m := float64(d.Sum) / float64(d.Count)
						interp_vec[i].Value = &m
					case "min":
						m := float64(d.Min)
						interp_vec[i].Value = &m
					case "max":
						m := float64(d.Max)
						interp_vec[i].Value = &m
					case "last":
						m := float64(d.Last)
						interp_vec[i].Value = &m
					case "first":
						m := float64(d.First)
						interp_vec[i].Value = &m
					default:
						s := float64(d.Sum)
						interp_vec[i].Value = &s
					}
					interp_vec[i].Time = uint32(d.Time.Unix()) //this is the "real" time, graphite does not care, but something might
					j++
				}
				break
			}
			cur_step_time += rawd.Step
		}
	}
	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, first_t, last_t, len(d_points), ct)

	whis.Series[m_key] = interp_vec

	return whis, nil
}

func (cass *CassandraMetric) Render(path string, from string, to string) (WhisperRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.render.get-time-ns", time.Now())

	var whis WhisperRenderItem
	whis.Series = make(map[string][]DataPoint)
	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	for _, pth := range paths {
		mets, err := cass.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	// ye old fan out technique
	render_one := func(metric indexer.MetricFindItem) {
		_ri, err := cass.RenderOne(metric, from, to)
		if err != nil {
			cass.render_wg.Done()
			return
		}
		cass.render_mu.Lock()
		for k, rr := range _ri.Series {
			whis.Series[k] = rr
		}
		whis.Start = _ri.Start
		whis.End = _ri.End
		whis.RealStart = _ri.RealStart
		whis.RealEnd = _ri.RealEnd
		whis.Step = _ri.Step

		cass.render_mu.Unlock()
		cass.render_wg.Done()
		return
	}

	for _, metric := range metrics {
		cass.render_wg.Add(1)
		go render_one(metric)
	}
	cass.render_wg.Wait()
	return whis, nil
}

func (cass *CassandraMetric) RawRender(path string, from string, to string) ([]*RawRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.rawrender.get-time-ns", time.Now())

	var rawd []*RawRenderItem

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	for _, pth := range paths {
		mets, err := cass.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	// ye old fan out technique
	render_one := func(metric indexer.MetricFindItem) {
		_ri, err := cass.RawRenderOne(metric, from, to)
		if err != nil {
			cass.render_wg.Done()
			return
		}
		rawd = append(rawd, _ri)
		cass.render_wg.Done()
		return
	}

	for _, metric := range metrics {
		cass.render_wg.Add(1)
		go render_one(metric)
	}
	cass.render_wg.Wait()
	return rawd, nil
}
