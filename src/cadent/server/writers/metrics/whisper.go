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
	The whisper file writer

	NOTE:: the whisper file ITSELF takes care of rollups, NOT, so unlike the DB/CSV ones, we DO NOT
	need to bother with the flushes for anything BUT the lowest time

	[acc.agg.writer.metrics]
	driver="whisper"
	dsn="/root/metrics/path"
	[acc.agg.writer.metrics.options]
		xFilesFactor=0.3
		write_workers=8
		write_queue_length=102400
		cache_series_type="protobuf"
		cache_metric_size=102400  # the "internal carbon-like-cache" size (ram is your friend)
		cache_byte_size=8192 # number of bytes per metric to cache above to keep before we drop (ram requirements cache_metric_size*cache_byte_size)
		cache_low_fruit_rate=0.25 # every 1/4 of the time write "low count" metrics to at least persist them
		writes_per_second=1000 # allowed physical writes per second

*/

package metrics

import (
	"cadent/server/dispatch"
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"fmt"
	"os/signal"
	"syscall"

	"cadent/server/broadcast"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"errors"
	whisper "github.com/robyoung/go-whisper"
	logging "gopkg.in/op/go-logging.v1"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	WHISPER_METRIC_WORKERS    = 8
	WHISPER_METRIC_QUEUE_LEN  = 1024 * 100
	WHISPER_WRITES_PER_SECOND = 1024
)

var errWhisperNotaDataNode = errors.New("Whisper: RawRenderOne: Not a data node")
var errWhisperNoDataInTimes = errors.New("Whisper: No data between these times")
var errWhisperNotImplemented = errors.New("Whisper: Not yet implemented")

// the singleton (per DSN) as we really only want one "controller" of writes not many lest we explode the
// IOwait to death

var _WHISP_WRITER_SINGLETON map[string]*WhisperWriter
var _whisp_set_mutex sync.Mutex

// special initer
func init() {
	_WHISP_WRITER_SINGLETON = make(map[string]*WhisperWriter)
}

func _get_whisp_signelton(conf map[string]interface{}) (*WhisperWriter, error) {
	_whisp_set_mutex.Lock()
	defer _whisp_set_mutex.Unlock()
	gots := conf["dsn"]
	if gots == nil {
		return nil, fmt.Errorf("Metrics: `dsn` /root/path/to/files is needed for whisper config")
	}

	dsn := gots.(string)
	if val, ok := _WHISP_WRITER_SINGLETON[dsn]; ok {
		return val, nil
	}

	writer, err := NewWhisperWriter(conf)
	if err != nil {
		return nil, err
	}
	_WHISP_WRITER_SINGLETON[dsn] = writer
	return writer, nil
}

type WhisperWriter struct {
	base_path    string
	xFilesFactor float32
	resolutions  [][]int

	retentions whisper.Retention
	indexer    indexer.Indexer

	cacher        *Cacher
	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

	shutdown   chan bool // when triggered, we skip the rate limiter and go full out till the queue is done
	shutitdown bool      // just a flag
	startstop  utils.StartStop

	write_queue       chan dispatch.IJob
	dispatch_queue    chan chan dispatch.IJob
	write_dispatcher  *dispatch.Dispatch
	num_workers       int
	queue_len         int
	writes_per_second int // number of actuall writes we will do per second assuming "INF" writes allowed

	write_lock sync.Mutex
	log        *logging.Logger
}

func NewWhisperWriter(conf map[string]interface{}) (*WhisperWriter, error) {
	ws := new(WhisperWriter)
	ws.log = logging.MustGetLogger("metrics.whisper")
	gots := conf["dsn"]
	if gots == nil {
		return nil, fmt.Errorf("`dsn` /root/path/of/data is needed for whisper config")
	}
	dsn := gots.(string)
	ws.base_path = dsn

	// remove trialing "/"
	if strings.HasSuffix(dsn, "/") {
		ws.base_path = dsn[0 : len(dsn)-1]
	}
	ws.xFilesFactor = 0.3
	_xf := conf["xFilesFactor"]
	if _xf != nil {
		ws.xFilesFactor = float32(_xf.(float64))
	}

	ws.num_workers = WHISPER_METRIC_WORKERS
	_workers := conf["write_workers"]
	if _workers != nil {
		ws.num_workers = int(_workers.(int64))
	}

	_qs := conf["write_queue_length"]
	ws.queue_len = WHISPER_METRIC_QUEUE_LEN
	if _qs != nil {
		ws.queue_len = int(_qs.(int64))
	}

	info, err := os.Stat(ws.base_path)
	if err != nil {
		return nil, fmt.Errorf("Whisper Metrics: counld not find base path %s", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("Whisper Metrics: base path %s is not a directory", ws.base_path)
	}

	_cache := conf["cache"]
	if _cache == nil {
		return nil, errMetricsCacheRequired
	}
	ws.cacher = _cache.(*Cacher)

	_rs := conf["writes_per_second"]
	ws.writes_per_second = WHISPER_WRITES_PER_SECOND
	if _rs != nil {
		ws.writes_per_second = int(_rs.(int64))
	}
	ws.shutdown = make(chan bool, 5)
	ws.shutitdown = false
	//go ws.TrapExit()
	//go ws.Start()
	return ws, nil
}

func (ws *WhisperWriter) getRetentions() (whisper.Retentions, error) {

	retentions := make(whisper.Retentions, 0)
	for _, item := range ws.resolutions {
		timebin := item[0]
		points := item[1]
		retent, err := whisper.ParseRetentionDef(fmt.Sprintf("%ds:%ds", timebin, points))
		if err != nil {
			return retentions, err
		}
		retentions = append(retentions, retent)
	}
	return retentions, nil
}

func (ws *WhisperWriter) getAggregateType(key string) whisper.AggregationMethod {
	agg_type := repr.GuessReprValueFromKey(key)
	switch agg_type {
	case repr.SUM:
		return whisper.Sum
	case repr.MAX:
		return whisper.Max
	case repr.MIN:
		return whisper.Min
	case repr.LAST:
		return whisper.Last
	default:
		return whisper.Average
	}
}

func (ws *WhisperWriter) getxFileFactor(key string) float32 {
	agg_type := repr.GuessReprValueFromKey(key)
	switch agg_type {
	case repr.SUM:
		return 0.0
	case repr.MAX:
		return 0.1
	case repr.MIN:
		return 0.1
	case repr.LAST:
		return 0.1
	default:
		return ws.xFilesFactor
	}
}

func (ws *WhisperWriter) getFilePath(metric string) string {
	if strings.Contains(metric, "/") {
		return metric
	}
	return filepath.Join(ws.base_path, strings.Replace(metric, ".", "/", -1)) + ".wsp"
}

func (ws *WhisperWriter) getFile(metric string) (*whisper.Whisper, error) {
	path := ws.getFilePath(metric)
	_, err := os.Stat(path)
	if err != nil {
		retentions, err := ws.getRetentions()
		if err != nil {
			return nil, err
		}
		agg := ws.getAggregateType(metric)

		// need to make the dirs
		if err = os.MkdirAll(filepath.Dir(path), os.ModeDir|os.ModePerm); err != nil {
			ws.log.Error("Could not make directory: %s", err)
			return nil, err
		}
		stats.StatsdClientSlow.Incr("writer.whisper.metric-creates", 1)
		return whisper.Create(path, retentions, agg, ws.xFilesFactor)
	}

	// need a recover in case files is corrupt and thus start over
	defer func() {
		if r := recover(); r != nil {
			ws.log.Critical("Whisper Failure (panic) %v :: need to remove file", r)
			ws.DeleteByName(metric)
		}
	}()

	return whisper.Open(path)
}

func (ws *WhisperWriter) TrapExit() {
	//trap kills to flush queues and close connections
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func(ins *WhisperWriter) {
		s := <-sc
		ins.log.Warning("Caught %s: Flushing remaining points out before quit ", s)

		ins.Stop()
		signal.Stop(sc)
		close(sc)

		// re-raise it
		process, _ := os.FindProcess(os.Getpid())
		process.Signal(s)
		return
	}(ws)
	return
}

// shutdown the writer, purging all cache to files as fast as we can
func (ws *WhisperWriter) Stop() {
	ws.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		ws.log.Warning("Whisper: Shutting down")

		if ws.shutitdown {
			return // already did
		}

		ws.shutitdown = true
		if ws.write_dispatcher != nil {
			ws.write_dispatcher.Shutdown()
		}
		ws.shutdown <- true

		ws.cacher.Stop()

		if ws.cacher == nil {
			ws.log.Warning("Whisper: Shutdown finished, nothing in queue to write")
			return
		}
		mets := ws.cacher.Queue
		mets_l := len(mets)
		ws.log.Warning("Whisper: Shutting down, exhausting the queue (%d items) and quiting", mets_l)
		// full tilt write out
		did := 0
		for _, queueitem := range mets {
			if did%100 == 0 {
				ws.log.Warning("Whisper: shutdown purge: written %d/%d...", did, mets_l)
			}
			name, points, _ := ws.cacher.GetById(queueitem.metric)
			if points != nil {
				stats.StatsdClient.Incr(fmt.Sprintf("writer.whisper.write.send-to-writers"), 1)
				ws.InsertMulti(name, points)
			}
			did++
		}

		r_cache := GetReadCache()
		if r_cache != nil {
			ws.log.Warning("Whisper: shutdown read cache")
			r_cache.Stop()
		}

		ws.log.Warning("Whisper: shutdown purge: written %d/%d...", did, mets_l)
		ws.log.Warning("Whisper: Shutdown finished ... quiting whisper writer")
	})
}

func (ws *WhisperWriter) Start() {

	/**** start the acctuall disk writer dispatcher queue ***/
	ws.startstop.Start(func() {
		workers := ws.num_workers
		ws.write_queue = make(chan dispatch.IJob, ws.queue_len)
		ws.dispatch_queue = make(chan chan dispatch.IJob, workers)
		ws.write_dispatcher = dispatch.NewDispatch(workers, ws.dispatch_queue, ws.write_queue)
		ws.write_dispatcher.SetRetries(2)
		ws.write_dispatcher.Run()

		ws.cacher.Start()
		go ws.sendToWriters() // fire up queue puller

		// set th overflow chan, and start the listener for that channel
		if ws.cacher.overFlowMethod == "chan" {
			ws.cacheOverFlow = ws.cacher.GetOverFlowChan()
			go ws.overFlowWrite()
		}
	})

}

// listen to the overflow chan from the cache and attempt to write "now"
func (ws *WhisperWriter) overFlowWrite() {
	for {
		select {
		case item, more := <-ws.cacheOverFlow.Ch:

			// bail
			if ws.shutitdown || !more {
				return
			}
			statitem := item.(*TotalTimeSeries)
			// need to make a list of points from the series
			iter, err := statitem.Series.Iter()
			if err != nil {
				ws.log.Error("error in overflow writer %v", err)
				continue
			}
			pts := make(repr.StatReprSlice, 0)
			for iter.Next() {
				pts = append(pts, iter.ReprValue())
			}
			if iter.Error() != nil {
				ws.log.Error("error in overflow iterator %v", iter.Error())
			}
			ws.log.Debug("Cache overflow force write for %s you may want to do something about that", statitem.Name.Key)
			ws.InsertMulti(statitem.Name, pts)
		}
	}
}

func (ws *WhisperWriter) InsertMulti(metric *repr.StatName, points repr.StatReprSlice) (npts int, err error) {

	whis, err := ws.getFile(metric.Key)
	if err != nil {
		ws.log.Error("Whisper:InsertMetric write error: %s", err)
		stats.StatsdClientSlow.Incr("writer.whisper.update-metric-errors", 1)
		return 0, err
	}

	// convert "points" to a whisper object
	whisp_points := make([]*whisper.TimeSeriesPoint, 0)
	agg_func := repr.GuessReprValueFromKey(metric.Key)
	for _, pt := range points {
		whisp_points = append(whisp_points,
			&whisper.TimeSeriesPoint{
				Time: int(pt.Time.Unix()), Value: pt.AggValue(agg_func),
			},
		)
	}

	// need a recover in case files is corrupt and thus start over
	defer func() {
		if r := recover(); r != nil {
			ws.log.Critical("Whisper Failure (panic) '%v' :: Corrupt :: need to remove file: %s", r, metric.Key)
			ws.DeleteByName(metric.Key)
			err = fmt.Errorf("%v", r)
		}
	}()

	if points != nil && len(whisp_points) > 0 {
		stats.StatsdClientSlow.GaugeAvg("writer.whisper.points-per-update", int64(len(points)))
		whis.UpdateMany(whisp_points)
	}
	stats.StatsdClientSlow.Incr("writer.whisper.update-many-writes", 1)

	whis.Close()
	return len(points), err
}

// insert the metrics from the cache
func (ws *WhisperWriter) InsertNext() (int, error) {
	defer stats.StatsdSlowNanoTimeFunc("writer.whisper.update-time-ns", time.Now())

	metric, points := ws.cacher.Pop()
	if metric == nil || points == nil || len(points) == 0 {
		return 0, nil
	}
	return ws.InsertMulti(metric, points)
}

// the "simple" insert one metric at a time model
func (ws *WhisperWriter) InsertOne(stat repr.StatRepr) (int, error) {

	whis, err := ws.getFile(stat.Name.Key)
	if err != nil {
		ws.log.Error("Whisper write error: %s", err)
		stats.StatsdClientSlow.Incr("writer.whisper.metric-errors", 1)
		return 0, err
	}

	err = whis.Update(float64(stat.Sum), int(stat.Time.Unix()))
	if err != nil {
		stats.StatsdClientSlow.Incr("writer.whisper.metric-writes", 1)
	}
	whis.Close()
	return 1, err
}

// "write" just adds to the whispercache
func (ws *WhisperWriter) Write(stat repr.StatRepr) error {

	if ws.shutitdown {
		return nil
	}

	// add to the writeback cache only
	ws.cacher.Add(&stat.Name, &stat)
	agg_func := repr.GuessReprValueFromKey(stat.Name.Key)
	// and now add it to the readcache iff it's been activated
	r_cache := GetReadCache()
	if r_cache != nil {
		// add only the "value" we need count==1 makes some timeseries much smaller in cache
		p_val := stat.AggValue(agg_func)
		t_stat := &repr.StatRepr{
			Time:  stat.Time,
			Sum:   repr.JsonFloat64(p_val),
			Count: 1,
			Name:  stat.Name,
		}
		r_cache.InsertQueue <- t_stat
	}
	//ws.write_queue <- WhisperMetricsJob{Stat: stat, Whis: ws}
	return nil
}

// pop from the cache and send to actual writers
func (ws *WhisperWriter) sendToWriters() error {
	// NOTE: if the writes per second is really high, this is NOT a good idea (using tickers)
	// that said, if you actually have the disk capacity to do 10k/writes per second ..  well damn
	// the extra CPU burn here may not kill you
	sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(ws.writes_per_second))
	ws.log.Notice("Starting Write limiter every %f nanoseconds (%d writes per second)", sleep_t, ws.writes_per_second)
	ticker := time.NewTicker(time.Duration(int(sleep_t)))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if ws.write_queue != nil {
				stats.StatsdClient.Incr("writer.whisper.write.send-to-writers", 1)
				ws.write_queue <- &WhisperMetricsJob{Whis: ws}
			}
		case <-ws.shutdown:
			ws.log.Warning("Whisper shutdown received, stopping write loop")
			return nil
		}
	}
}

// the only recourse to handle "corrupt" files is to nuke it and start again
// otherwise panics all around for no good reason
func (ws *WhisperWriter) DeleteByName(name string) (err error) {
	path := ws.getFilePath(name)
	err = os.Remove(path)
	if err != nil {
		return err
	}
	if ws.indexer == nil {
		return nil
	}
	return ws.indexer.Delete(&repr.StatName{Key: name})
}

/****************** Main Writer Interfaces *********************/
type WhisperMetrics struct {
	writer            *WhisperWriter
	resolutions       [][]int
	currentResolution int
	indexer           indexer.Indexer

	shutonce sync.Once // just shutdown "once and only once ever"

	log *logging.Logger
}

func NewWhisperMetrics() *WhisperMetrics {
	ws := new(WhisperMetrics)
	ws.log = logging.MustGetLogger("metrics.whisper")
	return ws
}

func (ws *WhisperMetrics) Driver() string {
	return "whisper"
}

func (ws *WhisperMetrics) Start() {
	ws.writer.Start()
}
func (ws *WhisperMetrics) Stop() {
	ws.writer.Stop()
}

func (ws *WhisperMetrics) Config(conf map[string]interface{}) error {
	gots, err := _get_whisp_signelton(conf)
	if err != nil {
		return err
	}
	ws.writer = gots
	return nil
}

func (ws *WhisperMetrics) SetIndexer(idx indexer.Indexer) error {
	ws.indexer = idx
	ws.writer.indexer = idx // needed for "delete" actions
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (ws *WhisperMetrics) SetResolutions(res [][]int) int {
	ws.resolutions = res
	ws.writer.resolutions = res
	return 1 // ONLY ONE writer needed whisper self-rolls up
}

func (ws *WhisperMetrics) SetCurrentResolution(res int) {
	ws.currentResolution = res
}

func (ws *WhisperMetrics) Write(stat repr.StatRepr) error {
	ws.indexer.Write(stat.Name) // write an index for the key
	err := ws.writer.Write(stat)

	return err
}

/**** READER ***/

func (ws *WhisperMetrics) GetFromReadCache(metric string, start int64, end int64) (rawd *RawRenderItem, got bool) {
	rawd = new(RawRenderItem)

	// check read cache
	r_cache := GetReadCache()
	if r_cache == nil {
		stats.StatsdClient.Incr("reader.whisper.render.cache.miss", 1)
		return rawd, false
	}

	t_start := time.Unix(int64(start), 0)
	t_end := time.Unix(int64(end), 0)
	cached_stats, _, _ := r_cache.Get(metric, t_start, t_end)
	var d_points []RawDataPoint
	step := uint32(0)
	if cached_stats != nil && len(cached_stats) > 0 {
		stats.StatsdClient.Incr("reader.whisper.render.cache.hits", 1)
		rawd.AggFunc = repr.GuessReprValueFromKey(metric)

		f_t := uint32(0)
		for _, stat := range cached_stats {
			t := uint32(stat.Time.Unix())
			d_points = append(d_points, RawDataPoint{
				Sum:   stat.AggValue(rawd.AggFunc),
				Count: 1,
				Time:  t,
			})
			if f_t <= 0 {
				f_t = t
			}
			if step <= 0 && f_t >= 0 {
				step = t - f_t
			}
		}

		rawd.RealEnd = d_points[len(d_points)-1].Time
		rawd.RealStart = d_points[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.RealEnd + step
		rawd.Metric = metric
		rawd.Step = step
		rawd.Data = d_points
		return rawd, len(d_points) > 0
	} else {
		stats.StatsdClient.Incr("reader.whisper.render.cache.miss", 1)
	}

	return rawd, false
}

func (ws *WhisperMetrics) RawDataRenderOne(metric indexer.MetricFindItem, start int64, end int64) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.whisper.renderraw.get-time-ns", time.Now())
	rawd := new(RawRenderItem)

	if metric.Leaf == 0 {
		//data only
		return rawd, errWhisperNotaDataNode
	}

	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	rawd.AggFunc = repr.GuessReprValueFromKey(metric.Id)
	rawd.Step = 1 // just for something in case of errors
	rawd.Metric = metric.Id
	rawd.Id = metric.UniqueId
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags

	//cache check
	// the read cache should have "all" the points from a "start" to "end" if the read cache has been activated for
	// a while.  If not, then it's a partial list (basically the read cache just started)
	stat_name := metric.StatName()
	cached, got_cache := ws.GetFromReadCache(stat_name.Key, start, end)
	// we assume the "cache" is hot data (by design) so if the num points is "lacking"
	// we know we need to get to the data store (or at least the inflight) for the rest
	// add the "step" here as typically we have the "forward" step of data
	if got_cache && cached.StartInRange(uint32(start)+cached.Step) {
		stats.StatsdClient.Incr("reader.whisper.render.read.cache.hit", 1)
		ws.log.Debug("Read Cache Hit: %s [%d, %d)", metric.Id, start, end)
		// need to set the start end to the ones requested before quantization to create the proper
		// spanning range
		cached.Start = uint32(start)
		cached.End = uint32(end)
		cached.AggFunc = rawd.AggFunc
		return cached, nil
	}
	path := ws.writer.getFilePath(stat_name.Key)
	whis, err := whisper.Open(path)

	if err != nil {
		// try the write inflight cache as nothing is written yet
		inflight_renderitem, err := ws.writer.cacher.GetAsRawRenderItem(stat_name)
		// need at LEAST 2 points to get the proper step size
		if inflight_renderitem != nil && err == nil && len(inflight_renderitem.Data) > 1 {
			// move the times to the "requested" ones and quantize the list
			inflight_renderitem.RealEnd = uint32(end)
			inflight_renderitem.RealStart = uint32(start)
			inflight_renderitem.Start = inflight_renderitem.RealStart
			inflight_renderitem.End = inflight_renderitem.RealEnd
			inflight_renderitem.Metric = metric.Id
			inflight_renderitem.Id = metric.UniqueId
			inflight_renderitem.AggFunc = rawd.AggFunc
			return inflight_renderitem, err
		} else {
			ws.writer.log.Error("Could not open file %s", path)
			return rawd, err
		}
	}
	defer whis.Close()

	// need a recover in case files is corrupt and thus start over
	var series *whisper.TimeSeries
	/*defer func() {
		if r := recover(); r != nil {
			ws.log.Critical("Read Whisper Failure (panic) %v :: Corrupt :: need to remove file: %s", r, path)
			ws.writer.DeleteByName(path)
			err = r.(error)
		}
	}()*/

	series, err = whis.Fetch(int(start), int(end))

	if err != nil || series == nil {
		if err == nil {
			err = errWhisperNoDataInTimes
		}
		ws.writer.log.Error("Could not get series from %s : error: %v", path, err)
		return rawd, err
	}

	points := series.Points()
	rawd.Data = make([]RawDataPoint, len(points), len(points))

	f_t := uint32(0)
	step_t := uint32(0)

	for idx, point := range points {
		rawd.Data[idx] = RawDataPoint{
			Count: 1,
			Sum:   point.Value, //just a stub for the only value we know
			Time:  uint32(point.Time),
		}
		if f_t <= 0 {
			f_t = uint32(point.Time)
		}
		if step_t <= 0 && f_t >= 0 {
			step_t = uint32(point.Time) - f_t
		}
	}

	rawd.RealEnd = uint32(end)
	rawd.RealStart = uint32(start)
	rawd.Start = rawd.RealStart
	rawd.End = rawd.RealEnd
	rawd.Step = step_t

	// grab the "current inflight" from the cache and merge into the main array
	inflight_data, err := ws.writer.cacher.GetAsRawRenderItem(stat_name)
	if err == nil && inflight_data != nil && len(inflight_data.Data) > 1 {
		inflight_data.Step = step_t // need to force this step size
		//merge with any inflight bits (inflight has higher precedence over the file)
		inflight_data.MergeWithResample(rawd, step_t)
		return inflight_data, nil
	}

	return rawd, nil
}

// after the "raw" render we need to yank just the "point" we need from the data which
// will make the read-cache much smaller (will compress just the Sum value as the count is 1)
// we set the "mean" as thats the value the graphite out render uses
func (ws *WhisperMetrics) RawRenderOne(metric indexer.MetricFindItem, from int64, to int64) (*RawRenderItem, error) {

	return ws.RawDataRenderOne(metric, from, to)

}

func (ws *WhisperMetrics) RawRender(path string, from int64, to int64) ([]*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.whisper.rawrender.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	for _, pth := range paths {
		mets, err := ws.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*RawRenderItem, len(metrics), len(metrics))

	// ye old fan out technique
	render_one := func(metric indexer.MetricFindItem, idx int) {
		defer render_wg.Done()
		_ri, err := ws.RawRenderOne(metric, from, to)

		if err != nil {
			return
		}
		rawd[idx] = _ri
		return
	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(metric, idx)
	}
	render_wg.Wait()
	return rawd, nil
}

func (ws *WhisperMetrics) CacheRender(path string, from int64, to int64, tags repr.SortingTags) ([]*RawRenderItem, error) {
	return nil, errWhisperNotImplemented
}
func (ws *WhisperMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*TotalTimeSeries, error) {
	return nil, fmt.Errorf("WhisperMetrics: CachedSeries: NOT YET IMPLIMNETED")
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type WhisperMetricsJob struct {
	Whis  *WhisperWriter
	Retry int
}

func (j *WhisperMetricsJob) IncRetry() int {
	j.Retry++
	return j.Retry
}
func (j *WhisperMetricsJob) OnRetry() int {
	return j.Retry
}

func (j *WhisperMetricsJob) DoWork() error {
	_, err := j.Whis.InsertNext()
	if err != nil {
		j.Whis.log.Error("Insert failed for Metric: %v retrying ...", err)
	}
	return err
}
