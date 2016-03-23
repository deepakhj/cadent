/*
	The whisper file writer

	[acc.agg.writer.metrics]
	driver="whisper"
	dsn="/root/metrics/path"
	xFilesFactor=0.3
	write_workers=32
	write_queue_length=102400

*/

package metrics

import (
	"cadent/server/dispatch"
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"fmt"
	whisper "github.com/robyoung/go-whisper"
	logging "gopkg.in/op/go-logging.v1"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	WHISPER_METRIC_WORKERS   = 32
	WHISPER_METRIC_QUEUE_LEN = 1024 * 100
)

// the singleton (per DSN) as we really only want one "controller" of writes not many

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
	base_path        string
	xFilesFactor     float32
	resolutions      [][]int
	retentions       whisper.Retention
	write_queue      chan dispatch.IJob
	dispatch_queue   chan chan dispatch.IJob
	write_dispatcher *dispatch.Dispatch
	num_workers      int
	queue_len        int
	max_write_size   int // size of that buffer before a flush

	write_lock sync.Mutex
	log        *logging.Logger
}

func NewWhisperWriter(conf map[string]interface{}) (*WhisperWriter, error) {
	ws := new(WhisperWriter)
	ws.log = logging.MustGetLogger("metrics.cassandra")
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
		ws.xFilesFactor = float32(_xf.(float32))
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

func (ws *WhisperWriter) getFile(stat repr.StatRepr) (*whisper.Whisper, error) {
	path := filepath.Join(ws.base_path, strings.Replace(stat.Key, ".", "/", -1)) + ".wsp"
	_, err := os.Stat(path)
	if err != nil {
		retentions, err := ws.getRetentions()
		if err != nil {
			return nil, err
		}
		agg := ws.getAggregateType(stat.Key)

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

func (ws *WhisperWriter) InsertOne(stat repr.StatRepr) (int, error) {

	whis, err := ws.getFile(stat)
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

func (ws *WhisperWriter) Write(stat repr.StatRepr) error {

	/**** dispatcher queue ***/
	if ws.write_queue == nil {
		workers := ws.num_workers
		ws.write_queue = make(chan dispatch.IJob, ws.queue_len)
		ws.dispatch_queue = make(chan chan dispatch.IJob, workers)
		ws.write_dispatcher = dispatch.NewDispatch(workers, ws.dispatch_queue, ws.write_queue)
		ws.write_dispatcher.SetRetries(2)
		ws.write_dispatcher.Run()
	}

	ws.write_queue <- WhisperMetricsJob{Stat: stat, Whis: ws}
	return nil
}

/****************** Main Writer Interfaces *********************/
type WhisperMetrics struct {
	writer      *WhisperWriter
	resolutions [][]int
	indexer     indexer.Indexer
	render_wg   sync.WaitGroup
	render_mu   sync.Mutex

	log *logging.Logger
}

func NewWhisperMetrics() *WhisperMetrics {
	my := new(WhisperMetrics)
	my.log = logging.MustGetLogger("metrics.whisper")
	return my
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
func (ws *WhisperMetrics) SetResolutions(res [][]int) {
	ws.resolutions = res
	ws.writer.resolutions = res
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
		ws.writer.log.Error("Could not open file %s", metric.Path)
		return rawd, err
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
	Stat  repr.StatRepr
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
	_, err := j.Whis.InsertOne(j.Stat)
	if err != nil {
		j.Whis.log.Error("Insert failed for Metric: %v retrying ...", j.Stat)
	}
	return err
}
