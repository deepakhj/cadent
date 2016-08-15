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
	CASSANDRA_DEFAULT_SERIES_TYPE    = "gorilla"
	CASSANDRA_DEFAULT_LONGEST_TIME   = "3600s"
	CASSANDRA_DEFAULT_SERIES_CHUNK   = 16 * 1024 * 1024 // 16kb
	CASSANDRA_DEFAULT_RENDER_TIMEOUT = "5s"
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

/****************** Metrics Writer *********************/
type CassandraMetric struct {
	resolutions [][]int
	static_tags repr.SortingTags
	indexer     indexer.Indexer
	writer      *CassandraWriter

	series_encoding string
	blobMaxBytes    int
	blobOldestTime  time.Duration

	cacher        *Cacher
	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

	render_wg     sync.WaitGroup
	render_mu     sync.Mutex
	renderTimeout time.Duration

	shutitdown bool
	shutdown   chan bool
}

func NewCassandraMetrics() *CassandraMetric {
	cass := new(CassandraMetric)
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

	dsn := _dsn.(string)

	resolution := conf["resolution"]
	if resolution == nil {
		return fmt.Errorf("resolution needed for cassandra writer")
	}

	cache_key := fmt.Sprintf("cassandrablob:cache:%s:%v", dsn, resolution)
	cass.cacher, err = getCacherSingleton(cache_key)
	if err != nil {
		return err
	}

	g_tag, ok := conf["tags"]
	if ok {
		cass.static_tags = repr.SortingTagsFromString(g_tag.(string))
	}

	cass.blobMaxBytes = CASSANDRA_DEFAULT_SERIES_CHUNK
	_bz := conf["cache_byte_size"]
	if _bz != nil {
		cass.blobMaxBytes = int(_bz.(int64))
	}

	pd := CASSANDRA_DEFAULT_LONGEST_TIME
	_pd := conf["cache_longest_time"]
	if _pd != nil {
		pd = _pd.(string)
	}
	dur, err := time.ParseDuration(pd)
	if err != nil {
		return fmt.Errorf("Cassandra cache_longest_time is not a valid duration: %v", err)
	}
	cass.blobOldestTime = dur

	cass.series_encoding = CASSANDRA_DEFAULT_SERIES_TYPE
	_se := conf["series_encoding"]
	if _se != nil {
		cass.series_encoding = _se.(string)
	}

	cass.blobMaxBytes = CASSANDRA_DEFAULT_SERIES_CHUNK
	_ps := conf["cache_byte_size"]
	if _ps != nil {
		cass.blobMaxBytes = int(_ps.(int64))
	}

	rdur, err := time.ParseDuration(CASSANDRA_DEFAULT_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	cass.renderTimeout = rdur

	if !cass.cacher.started && !cass.cacher.inited {
		// set the cacher bits
		_ms := conf["cache_metric_size"]
		if _ms != nil {
			cass.cacher.maxKeys = int(_ms.(int64))
		} else {
			cass.cacher.maxKeys = CACHER_METRICS_KEYS
		}

		// match blob types
		cass.cacher.seriesType = cass.series_encoding
		cass.cacher.maxBytes = cass.blobMaxBytes

		// unlike the other writers, overflows of cache size are
		// exactly what we want to write
		cass.cacher.overFlowMethod = "chan"
	}

	return nil
}

func (cass *CassandraMetric) Start() {
	/**** dispatcher queue ***/
	cass.writer.log.Notice("Starting cassandra series writer for %s at %d bytes per series", cass.writer.db.MetricTable(), cass.blobMaxBytes)
	cass.cacher.maxBytes = cass.blobMaxBytes
	cass.cacher.Start()

	// register the overflower
	cass.cacheOverFlow = cass.cacher.GetOverFlowChan()

	cass.shutitdown = false
	go cass.overFlowWrite()
}

func (cass *CassandraMetric) Stop() {
	shutdown.AddToShutdown()
	defer shutdown.ReleaseFromShutdown()
	cass.writer.log.Warning("Starting Shutdown of cassandra series writer")

	if cass.shutitdown {
		return // already did
	}
	cass.shutitdown = true

	cass.cacher.Stop()

	mets := cass.cacher.Queue
	mets_l := len(mets)
	cass.writer.log.Warning("Shutting down, exhausting the queue (%d items) and quiting", mets_l)
	// full tilt write out
	did := 0
	for _, queueitem := range mets {
		if did%100 == 0 {
			cass.writer.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
		}
		name, series, _ := cass.cacher.GetSeriesById(queueitem.metric)
		if series != nil {
			stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandra.write.send-to-writers"), 1)
			cass.writer.InsertSeries(name, series)
		}
		did++
	}
	cass.writer.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
	cass.writer.log.Warning("Shutdown finished ... quiting cassandra series writer")
	return
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

// listen to the overflow chan from the cache and attempt to write "now"
func (cass *CassandraMetric) overFlowWrite() {
	for {
		if cass.shutitdown {
			return
		}
		statitem, more := <-cass.cacheOverFlow.Ch
		if !more {
			return
		}
		cass.writer.InsertSeries(statitem.(*TotalTimeSeries).Name, statitem.(*TotalTimeSeries).Series)
	}
}

// simple proxy to the cacher
func (cass *CassandraMetric) Write(stat repr.StatRepr) error {
	if cass.shutitdown {
		return nil
	}
	stat.Name.MergeMetric2Tags(cass.static_tags)
	cass.indexer.Write(stat.Name)
	return cass.cacher.Add(&stat.Name, &stat)
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

	cass.writer.log.Debug("Select Q for %s: %s (%v, %v, %v, %v)", metric.Id, Q, metric.UniqueId, resolution, nano_start, nano_end)

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
			to, mi, mx, fi, ls, su, ct := s_iter.Values()

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
				First: fi,
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

func (cass *CassandraMetric) RawDataRenderOne(metric *indexer.MetricFindItem, from string, to string) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.renderraw.get-time-ns", time.Now())
	rawd := new(RawRenderItem)

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
	rawd.AggFunc = repr.GuessReprValueFromKey(metric.Id)

	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return rawd, errNotADataNode
	}

	b_len := (u_end - u_start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, errTimeTooSmall
	}

	//cache writer  check
	stat_name := metric.StatName()

	// grab data from the write inflight cache
	inflight, err := cass.cacher.GetAsRawRenderItem(stat_name)

	// need at LEAST 2 points to get the proper step size
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// all the data we need is in the inflight
		in_range := inflight.DataInRange(u_start, u_end)
		inflight.Metric = metric.Path
		inflight.Id = metric.UniqueId
		inflight.Step = resolution
		inflight.Start = u_start
		inflight.End = u_end

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
func (cass *CassandraMetric) RawRenderOne(metric indexer.MetricFindItem, from string, to string) (*RawRenderItem, error) {
	return cass.RawDataRenderOne(&metric, from, to)
}

func (cass *CassandraMetric) Render(path string, from string, to string) (WhisperRenderItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.render.get-time-ns", time.Now())

	raw_data, err := cass.RawRender(path, from, to)

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

func (cass *CassandraMetric) RawRender(path string, from string, to string) ([]*RawRenderItem, error) {

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
	render_one := func(metric indexer.MetricFindItem, idx int) {
		defer render_wg.Done()
		timeout := time.NewTimer(cass.renderTimeout)
		for {
			select {
			case <-timeout.C:
				cass.writer.log.Error("Render Timeout for %s (%s->%s)", path, from, to)
				timeout.Stop()
				return
			default:
				_ri, err := cass.RawRenderOne(metric, from, to)

				if err != nil {
					cass.writer.log.Error("Read Error for %s (%s->%s) : %v", path, from, to, err)
					return
				}
				rawd[idx] = _ri
				return
			}
		}

	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(metric, idx)
	}
	render_wg.Wait()
	return rawd, nil
}
