/*
	The "cacher"

	Designed to behave like "carbon cache" which allows a few things to happen

	1) ability to "buffer/ratelimit" writes so that we don't over whelm and writing backend
	2) Query things that are not yet written
	3) allow to reject incoming should things get too far behind (the only real recourse for stats influx)

	NOTE: keep in mind if you're backend uses multiple aggregation writers there will be a Cache for each
	writer (i.e. 10s, 1m, 10m bins) so plan your memory consumption accordingly

	settings

	[acc.agg.writer.metrics]
	driver="blaa"
	dsn="blaa"
	[acc.agg.writer.metrics.options]
		cache_metric_size=102400  # number of metric strings to keep
		cache_points_size=1024 # number of points per metric to cache above to keep before we drop (this * cache_metric_size * 32 * 128 bytes == your better have that ram)

*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/stats"
	"fmt"
	"sort"

	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	CACHER_NUMBER_POINTS = 1024
	CACHER_METRICS_KEYS  = 102400
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
	mu        sync.Mutex
	qmu       sync.Mutex
	maxKeys   int // max num of keys to keep before we have to drop
	maxPoints int // max num of points per key to keep before we have to drop
	log       *logging.Logger
	Queue     CacheQueue
	Cache     map[string][]*repr.StatRepr
}

func NewCacher() *Cacher {
	wc := new(Cacher)
	wc.maxKeys = CACHER_METRICS_KEYS
	wc.maxPoints = CACHER_NUMBER_POINTS
	wc.log = logging.MustGetLogger("cacher")
	wc.Cache = make(map[string][]*repr.StatRepr)
	go wc.startUpdateTick()
	return wc
}

func (wc *Cacher) DumpPoints(pts []*repr.StatRepr) {
	for idx, pt := range pts {
		wc.log.Notice("TimerSeries: %d Time: %d Mean: %f", idx, pt.Time.Unix(), pt.Mean)
	}
}

// do this only once a second as it can be expensive
func (wc *Cacher) startUpdateTick() {

	tick := time.NewTicker(time.Second)
	for {
		select {
		case <-tick.C:
			wc.updateQueue()
		}
	}
}

func (wc *Cacher) updateQueue() {
	newQueue := make(CacheQueue, 0)
	wc.qmu.Lock()
	defer wc.qmu.Unlock()

	for key, values := range wc.Cache {
		newQueue = append(newQueue, &CacheQueueItem{key, len(values)})
	}

	sort.Sort(newQueue)
	//wc.log.Critical("DATA %v", wc.Cache)
	stats.StatsdClientSlow.Gauge("cacher.metrics", int64(len(newQueue)))
	wc.Queue = newQueue
}

func (wc *Cacher) Add(metric string, stat *repr.StatRepr) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	//wc.log.Critical("STAT: %s, %d, %f", metric, time, value)

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
		if len(gots) > wc.maxPoints {
			wc.log.Error("Too Many points for %s (max points: %d)... have to drop this one", metric, wc.maxPoints)
			return fmt.Errorf("Cacher: too many points in cache, dropping metric")
			stats.StatsdClientSlow.Incr("cacher.points.overflow", 1)
		}
		wc.Cache[metric] = append(gots, stat)
		return nil
	}
	tp := make([]*repr.StatRepr, 0)
	tp = append(tp, stat)
	wc.Cache[metric] = tp

	if len(wc.Queue) == 0 {
		wc.updateQueue()
	}

	return nil
}

func (wc *Cacher) Get(metric string) ([]*repr.StatRepr, error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets", 1)

	if gots, ok := wc.Cache[metric]; ok {
		return gots, nil
	}
	return nil, nil
}

func (wc *Cacher) GetNextMetric() string {
	wc.mu.Lock()
	defer wc.mu.Unlock()
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

func (wc *Cacher) Pop() (string, []*repr.StatRepr) {
	metric := wc.GetNextMetric()
	if len(metric) != 0 {
		wc.mu.Lock()
		defer wc.mu.Unlock()

		if value, exists := wc.Cache[metric]; exists {
			delete(wc.Cache, metric)
			return metric, value
		}
	}
	return "", nil
}
