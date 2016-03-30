/*
	The Cassandra Metric Reader/Writer

	The table should have this schema to match the repr item
	The same as the writer items


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
		cache_points_size=1024 # number of points per metric to cache above to keep before we drop (this * cache_metric_size * 32 * 128 bytes == your better have that ram)
		writes_per_second=5000 # allowed insert queries per second

		numcons=5  # cassandra connection pool size
		timeout="30s" # query timeout
		user: ""
		pass: ""
		write_workers=32  # dispatch workers to write
		write_queue_length=102400  # buffered queue size before we start blocking

		# NOPE: batch_count: batch this many inserts for much faster insert performance (default 1000)
		# NOPE: periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/dispatch"
	"cadent/server/writers/indexer"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	CASSANDRA_RESULT_CACHE_SIZE = 1024 * 1024 * 100
	CASSANDRA_RESULT_CACHE_TTL  = 10 * time.Second
	CASSANDRA_METRIC_WORKERS    = 32
	CASSANDRA_METRIC_QUEUE_LEN  = 1024 * 100
	CASSANDRA_WRITES_PER_SECOND = 5000
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

func _get_signelton(conf map[string]interface{}) (*CassandraWriter, error) {
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
	_cass_set_mutex.Lock()
	defer _cass_set_mutex.Unlock()

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
type CassandraMetricJob struct {
	Cass  *CassandraWriter
	Stats []*repr.StatRepr // where the point list live
	retry int
}

func (j CassandraMetricJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j CassandraMetricJob) OnRetry() int {
	return j.retry
}

func (j CassandraMetricJob) DoWork() error {
	_, err := j.Cass.InsertMulti(j.Stats)
	return err
}

type CassandraWriter struct {
	// juse the writer connections for this
	db   *dbs.CassandraDB
	conn *gocql.Session

	write_list       []*repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	write_queue      chan dispatch.IJob
	dispatch_queue   chan chan dispatch.IJob
	write_dispatcher *dispatch.Dispatch
	cacher           *Cacher

	writes_per_second int          // allowed writes per second
	rate_limiter      *RateLimiter // rate limit the QPS
	num_workers       int
	queue_len         int
	max_write_size    int           // size of that buffer before a flush
	max_idle          time.Duration // either max_write_size will trigger a write or this time passing will
	write_lock        sync.Mutex
	log               *logging.Logger
}

func NewCassandraWriter(conf map[string]interface{}) (*CassandraWriter, error) {
	cass := new(CassandraWriter)
	cass.log = logging.MustGetLogger("metrics.cassandra")

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

	// if 5k/s allow 500 bursts in 100 millis
	if cass.writes_per_second > 1000 {
		cass.rate_limiter = NewRateLimiter(cass.writes_per_second, 100*time.Millisecond)
	} else {
		cass.rate_limiter = NewRateLimiter(cass.writes_per_second, time.Second)
	}

	// nope not doing this not efficient enough
	//go cass.PeriodFlush()

	return cass, nil
}

func (cass *CassandraWriter) PeriodFlush() {
	for {
		time.Sleep(cass.max_idle)
		cass.Flush()
	}
	return
}

// note this is not really used.  Batching in cassandra is not always a good idea
// since the token-aware insert will choose the proper server set, where as in batch mode
// that is not the case, this is here in case it turns out to be more performant
func (cass *CassandraWriter) Flush() (int, error) {
	cass.write_lock.Lock()
	defer cass.write_lock.Unlock()
	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.flush.metric-time-ns"), time.Now())
	l, err := cass.InsertMulti(cass.write_list)
	if err == nil {
		cass.write_list = make([]*repr.StatRepr, 0)
	}
	return l, err
}

// we can use the batcher effectively for single metric multi point writes as they share the
// the same token
func (cass *CassandraWriter) InsertMulti(points []*repr.StatRepr) (int, error) {
	cass.write_lock.Lock()
	defer cass.write_lock.Unlock()
	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.batch.metric-time-ns"), time.Now())

	l := len(points)
	if l == 0 {
		return 0, nil
	}
	/*if l == 1 {
		return cass.InsertOne(points[0]) // should be faster then the batcher logic
	}*/

	batch := cass.conn.NewBatch(gocql.LoggedBatch)
	Q := fmt.Sprintf(
		"INSERT INTO %s (id, time, point) VALUES  ({path: ?, resolution: ?}, ?, {sum: ?, mean: ?, min: ?, max: ?, count: ?})",
		cass.db.MetricTable(),
	)

	for _, stat := range points {
		DO_Q := Q
		if stat.TTL > 0 {
			DO_Q += fmt.Sprintf(" USING TTL %d", stat.TTL)
		}
		batch.Query(
			DO_Q,
			stat.Key,
			int64(stat.Resolution),
			stat.Time.UnixNano(),
			float64(stat.Sum),
			float64(stat.Mean),
			float64(stat.Min),
			float64(stat.Max),
			stat.Count,
		)
	}
	err := cass.conn.ExecuteBatch(batch)
	if err != nil {
		cass.log.Error("Cassandra Driver:Batch Metric insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandra.batch.metric-failures", 1)
		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandra.batch.writes", 1)
	stats.StatsdClientSlow.GaugeAvg("writer.cassandra.batch.metrics-per-writes", int64(l))

	return l, nil
}

func (cass *CassandraWriter) InsertOne(stat *repr.StatRepr) (int, error) {

	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.write.metric-time-ns"), time.Now())

	ttl := int64(0)
	if stat.TTL > 0 {
		ttl = stat.TTL
	}

	Q := fmt.Sprintf(
		"INSERT INTO %s (id, time, point) VALUES  ({path: ?, resolution: ?}, ?, {sum: ?, mean: ?, min: ?, max: ?, count: ?})",
		cass.db.MetricTable(),
	)
	if ttl > 0 {
		Q += " USING TTL ?"
	}

	err := cass.conn.Query(Q,
		stat.Key,
		int64(stat.Resolution),
		stat.Time.UnixNano(),
		float64(stat.Sum),
		float64(stat.Mean),
		float64(stat.Min),
		float64(stat.Max),
		stat.Count,
		ttl,
	).Exec()
	//cass.log.Critical("METRICS WRITE %d: %v", ttl, stat)
	if err != nil {
		cass.log.Error("Cassandra Driver: insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandra.metric-failures", 1)

		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandra.metric-writes", 1)

	return 1, nil
}

// pop from the cache and send to actual writers
func (cass *CassandraWriter) sendToWriters() error {
	// this may not be the "greatest" ratelimiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backedup
	// it will block on the write_queue stage
	sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(cass.writes_per_second))
	ticker := time.NewTicker(time.Duration(int(sleep_t)))
	cass.log.Notice("Starting Write limiter every %f nanoseconds (%d writes per second)", sleep_t, cass.writes_per_second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, points := cass.cacher.Pop()
			if points != nil {
				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandra.write.send-to-writers"), 1)
				cass.write_queue <- CassandraMetricJob{Cass: cass, Stats: points}
			}
		}
	}
}

func (cass *CassandraWriter) Write(stat repr.StatRepr) error {

	/**** dispatcher queue ***/
	if cass.write_queue == nil {
		workers := cass.num_workers
		cass.write_queue = make(chan dispatch.IJob, cass.queue_len)
		cass.dispatch_queue = make(chan chan dispatch.IJob, workers)
		cass.write_dispatcher = dispatch.NewDispatch(workers, cass.dispatch_queue, cass.write_queue)
		cass.write_dispatcher.SetRetries(2)
		cass.write_dispatcher.Run()
		go cass.sendToWriters() // the dispatcher
	}
	s_key := fmt.Sprintf("%s:%d", stat.Key, int(stat.Resolution))
	cass.cacher.Add(s_key, &stat)

	//cass.write_queue <- CassandraMetricJob{Stat: &stat, Cass: cass, Cacher: cacher}
	return nil

	/**** single write queue to keep a connection depletion in cassandra
	if cass.write_queue == nil {
		cass.write_queue = make(chan repr.StatRepr, cass.db.Cluster().NumConns*100)
		for i := 0; i < cass.db.Cluster().NumConns; i++ {
			go cass.consumeWriter()
		}
	}

	cass.write_queue <- stat
	return nil
	***/

	/** Direct insert_one tech
	go cass.InsertOne(stat)
	return nil
	**/

	/* If cassandra batching is faster ... use this .. (cassandra batching is not like mysql batching)
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

/****************** Metrics Writer *********************/
type CassandraMetric struct {
	resolutions [][]int
	indexer     indexer.Indexer
	writer      *CassandraWriter
	render_wg   sync.WaitGroup
	render_mu   sync.Mutex
}

func NewCassandraMetrics() *CassandraMetric {
	cass := new(CassandraMetric)
	return cass
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
	gots, err := _get_signelton(conf)
	if err != nil {
		return err
	}
	cass.writer = gots

	resolution := conf["resolution"]
	if resolution == nil {
		return fmt.Errorf("Resulotuion needed for cassandra writer")
	}
	gots.cacher, err = _get_cacher_signelton(conf["dsn"].(string))
	if err != nil {
		return err
	}

	// set the cacher bits
	_ms := conf["cache_metric_size"]
	if _ms != nil {
		gots.cacher.maxKeys = int(_ms.(int64))
	}

	_ps := conf["cache_points_size"]
	if _ps != nil {
		gots.cacher.maxPoints = int(_ps.(int64))
	}

	return nil
}

// simple proxy to the cacher
func (cass *CassandraMetric) Write(stat repr.StatRepr) error {
	return cass.writer.Write(stat)
}

/************************ READERS ****************/

// based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
// from and to needs to be > then the TTL as well
func (cass *CassandraMetric) getResolution(from int64, to int64) int {
	diff := int(math.Abs(float64(to - from)))
	n := int(time.Now().Unix())
	back_f := n - int(from)
	back_t := n - int(to)
	for _, res := range cass.resolutions {
		if diff < res[1] && back_f < res[1] && back_t < res[1] {
			return res[0]
		}
	}
	return cass.resolutions[len(cass.resolutions)-1][0]
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

	start = cass.truncateTo(start, resolution)
	end = cass.truncateTo(end, resolution)

	b_len := int(end-start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, fmt.Errorf("Cassandra: RawRenderOne: time too narrow")
	}

	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nano_end := end * nano
	nano_start := start * nano

	first_t := int(start)
	last_t := int(end)

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	cass_Q := fmt.Sprintf(
		"SELECT point.mean, point.max, point.min, point.sum, point.count, time FROM %s WHERE id={path: ?, resolution: ?} AND time <= ? and time >= ?",
		cass.writer.db.MetricTable(),
	)
	iter := cass.writer.conn.Query(cass_Q,
		metric.Id, resolution, nano_end, nano_start,
	).Iter()

	var t, count int64
	var mean, min, max, sum float64

	// use mins or maxes for the "upper_xxx, lower_xxx"
	m_key := metric.Id

	var d_points []RawDataPoint

	ct := 0
	// sorting order for the table is time ASC (i.e. first_t == first entry)

	for iter.Scan(&mean, &max, &min, &sum, &count, &t) {
		on_t := int(t / nano) // back convert to seconds

		d_points = append(d_points, RawDataPoint{
			Count: count,
			Sum:   sum,
			Mean:  mean,
			Max:   max,
			Min:   min,
			Time:  on_t,
		})
		//cass.log.Critical("POINT %s time:%d data:%f", metric.Id, on_t, mean)
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

	rawd.RealEnd = int(last_t)
	rawd.RealStart = int(first_t)
	rawd.Start = int(start)
	rawd.End = int(end)
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
	b_len := (rawd.End - rawd.Start) / rawd.Step //"proper" length of the metric

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
	ct := len(d_points)
	var last_got_t int
	var last_got_index int

	// grab from cache too if not yet written
	s_key := fmt.Sprintf("%s:%d", m_key, rawd.Step)
	inflight, err := cass.writer.cacher.Get(s_key)
	cass.writer.log.Critical("%s", s_key)
	cass.writer.cacher.DumpPoints(inflight)
	inflight_len := len(inflight)

	if ct > 0 { // got something from cassandra, make sure to fill any "missing times" w/ nils
		j := 0
		for i := 0; i < b_len; i++ {

			interp_vec[i] = DataPoint{Time: cur_step_time, Value: nil}

			for ; j < ct; j++ {
				d := d_points[j]
				if d.Time <= cur_step_time {

					// cass.writer.log.Critical("ONPS %v : %v Len %d :I %d, j %d, iLen: %v SUM: %v", d_points[j], interp_vec[i], ct, i, j, b_len, d_points[j].Sum,)

					// the weird setters here are to get the pointers properly (a weird golang thing)
					switch use_metric {
					case "mean":
						m := d.Mean
						interp_vec[i].Value = &m
					case "min":
						m := d.Min
						interp_vec[i].Value = &m
					case "max":
						m := d.Max
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

		// now attempt to merge the inflight data
		if len(inflight) > 0 && err == nil && last_got_t <= cur_step_time {
			j := 0
			for i := last_got_index; i < b_len; i++ {
				for ; j < inflight_len; j++ {
					d := inflight[j]
					if int(d.Time.Unix()) <= interp_vec[i].Time {
						// the weird setters here are to get the pointers properly (a weird golang thing)
						switch use_metric {
						case "mean":
							m := float64(d.Mean)
							interp_vec[i].Value = &m
						case "min":
							m := float64(d.Min)
							interp_vec[i].Value = &m
						case "max":
							m := float64(d.Max)
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
		j := 0
		for i := 0; i < b_len; i++ {
			interp_vec[i] = DataPoint{Time: cur_step_time, Value: nil}
			for ; j < inflight_len; j++ {
				d := inflight[j]
				if int(d.Time.Unix()) <= cur_step_time {

					// cass.writer.log.Critical("ONPS %v : %v Len %d :I %d, j %d, iLen: %v SUM: %v", d_points[j], interp_vec[i], ct, i, j, b_len, d_points[j].Sum,)

					// the weird setters here are to get the pointers properly (a weird golang thing)
					switch use_metric {
					case "mean":
						m := float64(d.Mean)
						interp_vec[i].Value = &m
					case "min":
						m := float64(d.Min)
						interp_vec[i].Value = &m
					case "max":
						m := float64(d.Max)
						interp_vec[i].Value = &m
					default:
						s := float64(d.Sum)
						interp_vec[i].Value = &s
					}
					interp_vec[i].Time = int(d.Time.Unix()) //this is the "real" time, graphite does not care, but something might
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
