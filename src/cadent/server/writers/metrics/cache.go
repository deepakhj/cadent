/*
	The "cacher"

	Designed to behave like "carbon cache" which allows a few things to happen

	1) ability to "buffer/ratelimit" writes so that we don't overwhelm and writing backend
	2) Query things that are not yet written
	3) allow to reject incoming should things get too far behind (the only real recourse for stats influx overload)

	settings

	[acc.agg.writer.metrics]
	driver="blaa"
	dsn="blaa"
	[acc.agg.writer.metrics.options]
		...
		cache_metric_size=102400  # number of metric strings to keep
		cache_byte_size=1024 # number of bytes to keep in caches per metric
		cache_series_type="protobuf" # gob, protobuf, json, gorilla
		...

	We use the default "protobuf" series as
		a) it's pretty small, and
		b) we can "sort it" by time (i.e. the format is NOT timeordered)
		(note: gorilla would be ideal for space, but REQUIRES time ordered inputs)
*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"cadent/server/writers/series"
	"fmt"
	"math/rand"
	"sort"

	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	CACHER_NUMBER_BYTES = 1024
	CACHER_SERIES_TYPE  = "protobuf"
	CACHER_METRICS_KEYS = 102400
)

/*
To save disk writes, the way carbon-cache does things is that it maintains an internal Queue of points
map[metric][]points
once a "worker" gets to the writing of that queue it dumps all the points at once not
one write per point.  The trick is that since graphite "queries" the cache it can backfill all the
not-written points into the return that are still waiting to be written, just from some initial testing
at a rate of 100k points in a 10s window it can take up to 5-15 min for the metrics to actually be written

welcome to diskIO pain, we will try to do the same thing here

a "write" operation will simply add things to this map of points and let a single writer write
the "render" step will then attempt to backfill those points

*/

// struct to pull the next thing to "write"
type CacheQueueItem struct {
	metric string
	count  int // number of stats in it
}

func (wqi *CacheQueueItem) ToString() string {
	return fmt.Sprintf("Metric: %s Count: %d", wqi.metric, wqi.count)
}

// we pop the "most" populated item
type CacheQueue []*CacheQueueItem

func (v CacheQueue) Len() int           { return len(v) }
func (v CacheQueue) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v CacheQueue) Less(i, j int) bool { return v[i].count < v[j].count }

// The "cache" item for points
type Cacher struct {
	mu           sync.RWMutex
	qmu          sync.Mutex
	maxKeys      int // max num of keys to keep before we have to drop
	maxBytes     int // max num of points per key to keep before we have to drop
	seriesType   string
	curSize      int64
	numCurPoint  int
	numCurKeys   int
	lowFruitRate float64   // % of the time we reverse the max sortings to persist low volume stats
	shutdown     chan bool // when recieved stop allowing adds and updates
	_accept      bool      // flag to stop
	log          *logging.Logger
	Queue        CacheQueue
	Cache        map[string]series.TimeSeries
}

func NewCacher() *Cacher {
	wc := new(Cacher)
	wc.maxKeys = CACHER_METRICS_KEYS
	wc.maxBytes = CACHER_NUMBER_BYTES
	wc.seriesType = CACHER_SERIES_TYPE
	wc.curSize = 0
	wc.log = logging.MustGetLogger("cacher.metrics")
	wc.Cache = make(map[string]series.TimeSeries)
	wc.shutdown = make(chan bool)
	wc._accept = true
	wc.lowFruitRate = 0.25
	go wc.startUpdateTick()
	return wc
}

func (wc *Cacher) Stop() {
	wc.shutdown <- true
}

func (wc *Cacher) DumpPoints(pts []*repr.StatRepr) {
	for idx, pt := range pts {
		wc.log.Notice("TimerSeries: %d Time: %d Sum: %f", idx, pt.Time.Unix(), pt.Sum)
	}
}

// do this only once a second as it can be expensive
func (wc *Cacher) startUpdateTick() {

	tick := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-tick.C:
			wc.updateQueue()
		case <-wc.shutdown:
			tick.Stop()
			wc._accept = false
			wc.log.Warning("Cache shutdown .. stopping accepts")
			return
		}
	}
	return
}

func (wc *Cacher) updateQueue() {
	newQueue := make(CacheQueue, 0)

	wc.mu.RLock() // need both locks
	f_len := 0
	for key, values := range wc.Cache {
		p_len := values.Len()
		newQueue = append(newQueue, &CacheQueueItem{key, p_len})
		f_len += values.Count()
	}
	wc.mu.RUnlock()
	wc.numCurPoint = f_len
	m_len := len(newQueue)
	wc.numCurKeys = m_len
	sort.Sort(newQueue)
	//wc.log.Critical("DATA %v", wc.Cache)
	wc.log.Debug("Cacher Sort: Metrics: %v :: Points: %v :: Bytes:: %d", m_len, f_len, wc.curSize)

	stats.StatsdClientSlow.Gauge("cacher.metrics", int64(m_len))
	stats.StatsdClientSlow.Gauge("cacher.points", int64(f_len))
	stats.StatsdClientSlow.Gauge("cacher.bytes", wc.curSize)

	// now for a bit of randomness, where we "reverse" the order on occasion to get the
	// not-updated often and thus hardly written to try to persist some slow stats
	// do this 1/4 of the time, so that we don't end up with having to shutdown in order to all things written
	r := rand.Float64()
	if r < wc.lowFruitRate {
		for i, j := 0, len(newQueue)-1; i < j; i, j = i+1, j-1 {
			newQueue[i], newQueue[j] = newQueue[j], newQueue[i]
		}
	}

	wc.qmu.Lock()
	defer wc.qmu.Unlock()
	wc.Queue = nil
	wc.Queue = newQueue
}

// add metric then update the sort queue
// use this for more "direct" writing for very small caches
func (wc *Cacher) AddAndUpdate(metric string, stat *repr.StatRepr) (err error) {
	err = wc.Add(metric, stat)
	wc.updateQueue()
	return err
}

func (wc *Cacher) Add(metric string, stat *repr.StatRepr) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	//wc.log.Critical("STAT: %s, %v", metric, stat)

	if !wc._accept {
		//wc.log.Error("Shutting down, will not add any more items to the queue")
		return nil
	}

	if len(wc.Cache) > wc.maxKeys {
		wc.log.Error("Key Cache is too large .. over %d metrics keys, have to drop this one", wc.maxKeys)
		return fmt.Errorf("Cacher: too many keys, dropping metric")
		stats.StatsdClientSlow.Incr("cacher.metrics.overflow", 1)
	}

	/** ye old debuggin'
	if strings.Contains(metric, "flushesposts") {
		wc.log.Critical("ADDING: %s Time: %d, Val: %f", metric, time, value)
	}
	*/

	if gots, ok := wc.Cache[metric]; ok {
		cur_size := gots.Len()
		if gots.Len() > wc.maxBytes {
			wc.log.Error("Too Many Bytes for %s (max points: %d)... have to drop this one", metric, wc.maxBytes)
			return fmt.Errorf("Cacher: too many points in cache, dropping metric")
			stats.StatsdClientSlow.Incr("cacher.points.overflow", 1)
		}
		wc.Cache[metric].AddStat(stat)
		now_len := wc.Cache[metric].Len()
		wc.curSize += int64(now_len - cur_size)
		stats.StatsdClient.GaugeAvg("cacher.add.ave-points-per-metric", int64(gots.Count()))
		return nil
	}
	tp, err := series.NewTimeSeries(wc.seriesType, stat.Time.UnixNano(), nil)
	if err != nil {
		return err
	}
	tp.AddStat(stat)
	wc.Cache[metric] = tp
	wc.curSize += int64(tp.Len())
	stats.StatsdClient.GaugeAvg("cacher.add.ave-points-per-metric", 1)

	return nil
}

func (wc *Cacher) getStatsStream(metric string, ts series.TimeSeries) (repr.StatReprSlice, error) {
	it, err := ts.Iter()
	if err != nil {
		wc.log.Error("Failed to get series itterator: %v", err)
		stats.StatsdClientSlow.Incr("cacher.read.cache-gets.error", 1)
		return nil, err
	}
	stats := make(repr.StatReprSlice, 0)
	for it.Next() {
		st := it.ReprValue()
		if st == nil {
			continue
		}
		st.Name.Key = metric
		stats = append(stats, st)
	}
	if it.Error() != nil {
		wc.log.Error("Iterator error: %v", err)
		return stats, it.Error()
	}
	if stats != nil {
		sort.Sort(stats)
	}
	return stats, nil
}

func (wc *Cacher) Get(metric string) (repr.StatReprSlice, error) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets", 1)

	if gots, ok := wc.Cache[metric]; ok {
		stats.StatsdClientSlow.Incr("cacher.read.cache-gets.values", 1)
		return wc.getStatsStream(metric, gots)
	}
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets.empty", 1)
	return nil, nil
}

func (wc *Cacher) GetNextMetric() string {
	wc.qmu.Lock()
	defer wc.qmu.Unlock()

	for {
		size := len(wc.Queue)
		if size == 0 {
			break
		}
		item := wc.Queue[size-1]
		wc.Queue = wc.Queue[:size-1]
		return item.metric
	}
	return ""
}

// just grab something from the list
func (wc *Cacher) GetAnyStat() (string, repr.StatReprSlice) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	for metric, stats := range wc.Cache {
		out, _ := wc.getStatsStream(metric, stats)
		wc.curSize -= int64(stats.Len()) // shrink the bytes
		delete(wc.Cache, metric)         // need to purge if error as things are corrupted somehow
		return metric, out
	}
	return "", nil
}

func (wc *Cacher) Pop() (string, repr.StatReprSlice) {
	metric, ts := wc.PopSeries()
	if ts == nil {
		return "", nil
	}
	out, _ := wc.getStatsStream(metric, ts)
	return metric, out
}

func (wc *Cacher) PopSeries() (string, series.TimeSeries) {
	metric := wc.GetNextMetric()
	if len(metric) != 0 {
		wc.mu.Lock()
		defer wc.mu.Unlock()

		if stats, exists := wc.Cache[metric]; exists {
			wc.curSize -= int64(stats.Len()) // shrink the bytes
			delete(wc.Cache, metric)         // need to delete regardless as we have byte errors and things are corrupted
			return metric, stats
		}
	}
	return "", nil
}

// add a metrics/point list back on the queue as it either "failed" or was ratelimited
func (wc *Cacher) AddBack(metric string, points repr.StatReprSlice) {
	for _, pt := range points {
		wc.Add(metric, pt)
	}
}
