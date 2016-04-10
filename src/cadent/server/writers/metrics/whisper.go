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
		cache_metric_size=102400  # the "internal carbon-like-cache" size (ram is your friend)
		cache_points_size=1024 # number of points per metric to cache above to keep before we drop (this * cache_metric_size * 32 * 128 bytes == your better have that ram)
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
	WHISPER_WRITES_PER_SECOND = 1000
)

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
	retentions   whisper.Retention

	cache_queue *Cacher

	shutdown   chan bool // when triggered, we skip the rate limiter and go full out till the queue is done
	shutitdown bool      // just a flag

	write_queue       chan dispatch.IJob
	dispatch_queue    chan chan dispatch.IJob
	write_dispatcher  *dispatch.Dispatch
	num_workers       int
	queue_len         int
	writes_per_second int // number of acctuall writes we will do per second assuming "INF" writes allowed

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
		return nil, fmt.Errorf("Whisper Metrics: base path %s is not a directory", err)
	}

	// set up the cacher
	ws.cache_queue = NewCacher()

	_ms := conf["cache_metric_size"]
	if _ms != nil {
		ws.cache_queue.maxKeys = int(_ms.(int64))
	}

	_ps := conf["cache_points_size"]
	if _ps != nil {
		ws.cache_queue.maxPoints = int(_ps.(int64))
	}

	_lf := conf["cache_low_fruit_rate"]
	if _lf != nil {
		ws.cache_queue.lowFruitRate = _lf.(float64)
	}

	_rs := conf["writes_per_second"]
	ws.writes_per_second = WHISPER_WRITES_PER_SECOND
	if _rs != nil {
		ws.writes_per_second = int(_rs.(int64))
	}
	ws.shutdown = make(chan bool, 1)
	ws.shutitdown = false
	go ws.TrapExit()

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
	agg_type := indexer.GuessAggregateType(key)
	switch agg_type {
	case "sum":
		return whisper.Sum
	case "max":
		return whisper.Max
	case "min":
		return whisper.Min
	default:
		return whisper.Average
	}
	return whisper.Average
}

func (ws *WhisperWriter) getxFileFactor(key string) float32 {
	agg_type := indexer.GuessAggregateType(key)
	switch agg_type {
	case "sum":
		return 0.0
	case "max":
		return 0.1
	case "min":
		return 0.1
	default:
		return ws.xFilesFactor
	}
	return ws.xFilesFactor
}

func (ws *WhisperWriter) getFile(metric string) (*whisper.Whisper, error) {
	path := filepath.Join(ws.base_path, strings.Replace(metric, ".", "/", -1)) + ".wsp"
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

	return whisper.Open(path)
}

func (ws *WhisperWriter) guessValue(metric string, stat *repr.StatRepr) float64 {

	use_metric := indexer.GuessAggregateType(metric)

	switch use_metric {
	case "sum":
		return float64(stat.Sum)
	case "max":
		return float64(stat.Max)
	case "min":
		return float64(stat.Min)
	default:
		return float64(stat.Mean)
	}
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

	if ws.shutitdown {
		return // already did
	}
	ws.shutitdown = true
	ws.shutdown <- true
	ws.cache_queue.Stop()
	if ws.write_dispatcher != nil {
		ws.write_dispatcher.Shutdown()
	}
	if ws.cache_queue == nil {
		ws.log.Warning("Whisper: Shutdown finished, nothing in queue to write")
		return
	}
	mets := ws.cache_queue.Queue
	mets_l := len(mets)
	ws.log.Warning("Whisper: Shutting down, exhausting the queue (%d items) and quiting", mets_l)
	// full tilt write out
	did := 0
	for {
		met, pts := ws.cache_queue.Pop()
		if met == "" {
			break
		}
		if did%100 == 0 {
			ws.log.Warning("Whisper: shutdown purge: written %d/%d...", did, mets_l)
		}
		if pts != nil {
			stats.StatsdClient.Incr(fmt.Sprintf("writer.whisper.write.send-to-writers"), 1)
			ws.InsertMulti(met, pts)
		}
		did++
	}

	ws.log.Warning("Whisper: shutdown purge: written %d/%d...", did, mets_l)
	ws.log.Warning("Whisper: Shutdown finished ... quiting whisper writer")
	return
}

func (ws *WhisperWriter) InsertMulti(metric string, points []*repr.StatRepr) (int, error) {
	whis, err := ws.getFile(metric)
	if err != nil {
		ws.log.Error("Whisper:InsertMetric write error: %s", err)
		stats.StatsdClientSlow.Incr("writer.whisper.update-metric-errors", 1)
		return 0, err
	}

	/* ye old debuggin
	if strings.Contains(metric, "flushesposts") {
		ws.cache_queue.DumpPoints(points)
	} */

	// convert "poits" to a whisper object
	whisp_points := make([]*whisper.TimeSeriesPoint, 0)
	for _, pt := range points {

		whisp_points = append(whisp_points,
			&whisper.TimeSeriesPoint{
				Time: int(pt.Time.Unix()), Value: ws.guessValue(metric, pt),
			},
		)
	}

	if points != nil {
		stats.StatsdClientSlow.GaugeAvg("writer.whisper.points-per-update", int64(len(points)))
		whis.UpdateMany(whisp_points)
	}
	stats.StatsdClientSlow.Incr("writer.whisper.update-many-writes", 1)

	whis.Close()
	return len(points), nil
}

// insert the metrics from the cache
func (ws *WhisperWriter) InsertNext() (int, error) {
	defer stats.StatsdSlowNanoTimeFunc("writer.whisper.update-time-ns", time.Now())

	metric, points := ws.cache_queue.Pop()
	if points == nil {
		return 0, nil
	}
	return ws.InsertMulti(metric, points)
}

// the "simple" insert one metric at a time model
func (ws *WhisperWriter) InsertOne(stat repr.StatRepr) (int, error) {

	whis, err := ws.getFile(stat.Key)
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

	/**** start the acctuall disk writer dispatcher queue ***/
	if ws.write_queue == nil {
		workers := ws.num_workers
		ws.write_queue = make(chan dispatch.IJob, ws.queue_len)
		ws.dispatch_queue = make(chan chan dispatch.IJob, workers)
		ws.write_dispatcher = dispatch.NewDispatch(workers, ws.dispatch_queue, ws.write_queue)
		ws.write_dispatcher.SetRetries(2)
		ws.write_dispatcher.Run()
		go ws.sendToWriters() // fireup queue puller
	}

	// add to the cache only
	ws.cache_queue.Add(stat.Key, &stat)
	//ws.write_queue <- WhisperMetricsJob{Stat: stat, Whis: ws}
	return nil
}

// pop from the cache and send to actual writers
func (ws *WhisperWriter) sendToWriters() error {
	// NOTE: if the writes per second is really high, this is NOT a good idea (using tickers)
	// that said, if you actually have the disk capacity to do 10k/writes per second ..  well damn
	// the extra CPU burn here may not kill you
	sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(ws.writes_per_second))
	ticker := time.NewTicker(time.Duration(int(sleep_t)))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if ws.write_queue != nil {
				ws.write_queue <- WhisperMetricsJob{Whis: ws}
			}
		case <-ws.shutdown:
			ws.log.Warning("Whisper shutdown received, stopping write loop")
			break
		}
	}
	return nil
}

/****************** Main Writer Interfaces *********************/
type WhisperMetrics struct {
	writer      *WhisperWriter
	resolutions [][]int
	indexer     indexer.Indexer
	render_wg   sync.WaitGroup
	render_mu   sync.Mutex

	shutonce sync.Once // just shutdown "once and only once ever"

	log *logging.Logger
}

func NewWhisperMetrics() *WhisperMetrics {
	ws := new(WhisperMetrics)
	ws.log = logging.MustGetLogger("metrics.whisper")
	return ws
}

func (ws *WhisperMetrics) Stop() {
	ws.shutonce.Do(ws.writer.Stop)
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

func (ws *WhisperMetrics) Write(stat repr.StatRepr) error {
	err := ws.writer.Write(stat)
	return err
}

/**** READER ***/

func (ws *WhisperMetrics) RawRenderOne(metric indexer.MetricFindItem, from string, to string) (*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.whisper.renderraw.get-time-ns", time.Now())

	rawd := new(RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("Whisper: RawRenderOne: Not a data node")
	}

	start, err := ParseTime(from)
	if err != nil {
		ws.writer.log.Error("Invalid from time `%s` :: %v", from, err)
		return rawd, err
	}

	end, err := ParseTime(to)
	if err != nil {
		ws.writer.log.Error("Invalid from time `%s` :: %v", to, err)
		return rawd, err
	}
	if end < start {
		start, end = end, start
	}

	whis, err := whisper.Open(metric.Id)
	if err != nil {
		// try the cache
		var d_points []RawDataPoint
		inflight, err := ws.writer.cache_queue.Get(metric.Path)
		if inflight != nil && err == nil && len(inflight) > 0 {
			f_t := 0
			step_t := 0

			for _, pt := range inflight {
				t := int(pt.Time.Unix())
				d_points = append(d_points, RawDataPoint{
					Mean: ws.writer.guessValue(metric.Path, pt), //just a stub for the only value we know
					Time: t,
				})
				if f_t <= 0 {
					f_t = t
				}
				if step_t <= 0 && f_t >= 0 {
					step_t = t - f_t
				}
			}

			rawd.RealEnd = int(end)
			rawd.RealStart = int(start)
			rawd.Start = int(start)
			rawd.End = int(end)
			rawd.Step = int(step_t)
			rawd.Metric = metric.Path
			rawd.Data = d_points
			return rawd, nil
		} else {
			ws.writer.log.Error("Could not open file %s", metric.Path)
			return rawd, err
		}
	}

	series, err := whis.Fetch(int(start), int(end))
	if err != nil {
		ws.writer.log.Error("Could not get series from %s", metric.Path)
		return rawd, err
	}

	var d_points []RawDataPoint
	f_t := 0
	step_t := 0

	for _, point := range series.Points() {
		d_points = append(d_points, RawDataPoint{
			Mean: point.Value, //just a stub for the only value we know
			Time: point.Time,
		})
		if f_t <= 0 {
			f_t = point.Time
		}
		if step_t <= 0 && f_t >= 0 {
			step_t = point.Time - f_t
		}
	}

	// grab the "current inflight" from the cache
	inflight, err := ws.writer.cache_queue.Get(metric.Path)
	if inflight != nil && err == nil && len(inflight) > 0 && len(d_points) > 0 {
		ws.writer.cache_queue.DumpPoints(inflight)
		inflight_l := len(inflight)

		// since inflight is to have later data then the file itself, we go "backwards" down the
		// inflight list

		first_inflight := inflight[0]
		first_dpoint := d_points[0]
		start_d_point := 0
		start_inf_point := 0

		// find the start index in the data points if we've more then inflight
		if int(first_inflight.Time.Unix()) > first_dpoint.Time {
			for idx, pt := range d_points {
				if pt.Time >= int(first_inflight.Time.Unix()) {
					start_d_point = idx
					break
				}
			}
		} else {
			// otherwise the inflight has more data then the file read
			for idx, pt := range inflight {
				if int(pt.Time.Unix()) >= first_dpoint.Time {
					start_inf_point = idx
					break
				}
			}
		}

		for i := start_d_point; i < len(d_points); i++ {
			for j := start_inf_point; j < inflight_l; j++ {
				//ws.writer.log.Critical("MM: onT: %d, inflight T: %d", d_points[i].Time, inflight[j].Time)

				if int(inflight[j].Time.Unix()) == d_points[i].Time {
					d_points[i].Mean = ws.writer.guessValue(metric.Path, inflight[j])
					start_inf_point++
					break
				}
			}
		}

	}
	// if the first/last fit w/i the timewindow we can backfill

	rawd.RealEnd = int(end)
	rawd.RealStart = int(start)
	rawd.Start = int(start)
	rawd.End = int(end)
	rawd.Step = int(step_t)
	rawd.Metric = metric.Path
	rawd.Data = d_points

	return rawd, nil
}

func (ws *WhisperMetrics) Render(path string, from string, to string) (WhisperRenderItem, error) {

	raw_data, err := ws.RawRender(path, from, to)

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
			v := d.Mean
			d_points = append(d_points, DataPoint{Time: d.Time, Value: &v})
		}
		whis.Series[data.Metric] = d_points
	}
	return whis, err
}

func (ws *WhisperMetrics) RawRender(path string, from string, to string) ([]*RawRenderItem, error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.whisper.rawrender.get-time-ns", time.Now())

	var rawd []*RawRenderItem

	paths := strings.Split(path, ",")
	var metrics []indexer.MetricFindItem

	for _, pth := range paths {
		mets, err := ws.indexer.Find(pth)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	// ye old fan out technique
	render_one := func(metric indexer.MetricFindItem) {
		_ri, err := ws.RawRenderOne(metric, from, to)
		if err != nil {
			ws.render_wg.Done()
			return
		}
		rawd = append(rawd, _ri)
		ws.render_wg.Done()
		return
	}

	for _, metric := range metrics {
		ws.render_wg.Add(1)
		go render_one(metric)
	}
	ws.render_wg.Wait()
	return rawd, nil
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type WhisperMetricsJob struct {
	Whis  *WhisperWriter
	retry int
}

func (j WhisperMetricsJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j WhisperMetricsJob) OnRetry() int {
	return j.retry
}

func (j WhisperMetricsJob) DoWork() error {
	_, err := j.Whis.InsertNext()
	if err != nil {
		j.Whis.log.Error("Insert failed for Metric: %v retrying ...", j)
	}
	return err
}
