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
		cache_byte_size=8192 # number of bytes to keep in caches per metric
		cache_series_type="protobuf" # gob, protobuf, json, gorilla
		...

	We use the default "gob" series as
		a) it's pretty small, and
		b) we can "sort it" by time (i.e. the format is NOT time-ordered)
		(note: gorilla would be ideal for space, but REQUIRES time ordered inputs)

	Overflow cache_overflow_method ::
		2 overflow modes allowed, an overflow is when there are too make bytes in a series
		based on the cache_byte_size option.

		DROP ::
		`cache_overflow_method="drop"` : just drop the points as we can do no more until the current blob is written

		if the writers are known not to be able to keep up fast enough, this is really the only thing we can do
		otherwise we will run out of ram, and lock on the "chan" method stopping the world basically.

		CHAN ::
		`cache_overflow_method="chan"` : send the current about to expire timeseries + name pair to a channel

		 for the `chan` method to function correctly, it must be set "external" to this object
		 (i.e. the writers/whatever need to give this object the channel to send the overflows)

		 this is useful for writers that use want to write entire "blobs" of timeseries rather then
		 write single points of data.  Basically we want to write the entire cache_byte_size blob in one
		 write action and not the single points.



*/

package metrics

import (
	"cadent/server/repr"
	"cadent/server/series"
	"cadent/server/stats"
	"fmt"
	"math/rand"
	"sort"

	"cadent/server/broadcast"
	"cadent/server/utils/shutdown"
	"errors"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	CACHER_NUMBER_BYTES     = 8192
	CACHER_SERIES_TYPE      = "protobuf"
	CACHER_METRICS_KEYS     = 102400
	CACHER_DEFAULT_OVERFLOW = "drop"
)

/*
To save disk writes, the way carbon-cache does things is that it maintains an internal timseries
map[metric]timesereis
once a "worker" gets to the writing of that queue it dumps all the points at once not
one write per point.  The trick is that since graphite "queries" the cache it can backfill all the
not-written points into the return that are still waiting to be written, just from some initial testing
at a rate of 100k points in a 10s window it can take up to 5-15 min for the metrics to actually be written

welcome to diskIO pain, we will try to do the same thing here

a "write" operation will simply add things to this map of points and let a single writer write
the "render" step will then attempt to backfill those points

*/

// Cache singletons as the readers may need to use this too

// the singleton
var _CACHER_SINGLETON map[string]*Cacher
var _cacher_mutex sync.RWMutex

func getCacherSingleton(nm string) (*Cacher, error) {
	_cacher_mutex.Lock()
	defer _cacher_mutex.Unlock()

	if val, ok := _CACHER_SINGLETON[nm]; ok {
		return val, nil
	}

	cacher := NewCacher()
	_CACHER_SINGLETON[nm] = cacher
	cacher.Name = nm
	return cacher, nil
}

// just GET by name if it exists
func getCacherByName(nm string) *Cacher {
	_cacher_mutex.RLock()
	defer _cacher_mutex.RUnlock()

	if val, ok := _CACHER_SINGLETON[nm]; ok {
		return val
	}
	return nil
}

// special onload init
func init() {
	_CACHER_SINGLETON = make(map[string]*Cacher)
}

// struct to pull the next thing to "write"
type CacheQueueItem struct {
	metric repr.StatId
	count  int // number of stats in it
	bytes  int // byte number
}

func (wqi *CacheQueueItem) ToString() string {
	return fmt.Sprintf("Metric Id: %v Count: %d", wqi.metric, wqi.count)
}

// we pop the "most" populated item
type CacheQueue []*CacheQueueItem

func (v CacheQueue) Len() int           { return len(v) }
func (v CacheQueue) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v CacheQueue) Less(i, j int) bool { return v[i].count < v[j].count }

// we pop the "most" biggest item
type CacheQueueBytes []*CacheQueueItem

func (v CacheQueueBytes) Len() int           { return len(v) }
func (v CacheQueueBytes) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v CacheQueueBytes) Less(i, j int) bool { return v[i].bytes < v[j].bytes }

//errors
var errWriteCacheTooManyMetrics = errors.New("Cacher: too many keys, dropping metric")
var errWriteCacheTooManyPoints = errors.New("Cacher: too many points in cache, dropping metric")

// The "cache" item for points
type Cacher struct {
	mu           *sync.RWMutex
	qmu          *sync.Mutex
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
	NameCache    map[repr.StatId]*repr.StatName
	Cache        map[repr.StatId]series.TimeSeries
	Name         string // just a human name for things

	//overflow pieces
	overFlowMethod string

	// allow for multiple registering entities
	overFlowBroadcast *broadcast.Broadcaster // should pass in *TotalTimeSeries

	started bool
	inited  bool
}

func NewCacher() *Cacher {
	wc := new(Cacher)
	wc.mu = new(sync.RWMutex)
	wc.qmu = new(sync.Mutex)
	wc.maxKeys = CACHER_METRICS_KEYS
	wc.maxBytes = CACHER_NUMBER_BYTES
	wc.seriesType = CACHER_SERIES_TYPE
	wc.overFlowMethod = CACHER_DEFAULT_OVERFLOW
	wc.overFlowBroadcast = nil

	wc.curSize = 0
	wc.log = logging.MustGetLogger("cacher.metrics")
	wc.Cache = make(map[repr.StatId]series.TimeSeries)
	wc.NameCache = make(map[repr.StatId]*repr.StatName)
	wc.shutdown = make(chan bool)
	wc._accept = true
	wc.lowFruitRate = 0.25
	wc.started = false
	wc.inited = false

	wc.overFlowBroadcast = broadcast.New(128)
	return wc
}

func (wc *Cacher) Start() {
	if !wc.started {
		wc.started = true
		wc.log.Notice("Starting Metric Cache sorter tick (%d max metrics, %d max bytes per metric) [%s]", wc.maxKeys, wc.maxBytes, wc.Name)
		go wc.startUpdateTick()
	}
}

func (wc *Cacher) Stop() {
	shutdown.AddToShutdown()
	if wc.started {
		wc.shutdown <- true
		wc.overFlowBroadcast.Close()
	}
}

func (wc *Cacher) GetOverFlowChan() *broadcast.Listener {
	return wc.overFlowBroadcast.Listen()
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
			shutdown.ReleaseFromShutdown()

			return
		}
	}
}

func (wc *Cacher) updateQueue() {
	if !wc._accept {
		return
	}
	f_len := 0
	idx := 0
	wc.mu.RLock() // need both locks
	newQueue := make(CacheQueue, len(wc.Cache))
	for key, values := range wc.Cache {
		num_points := values.Count()
		newQueue[idx] = &CacheQueueItem{key, num_points, values.Len()}
		idx++
		f_len += num_points
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
func (wc *Cacher) AddAndUpdate(metric *repr.StatName, stat *repr.StatRepr) (err error) {
	err = wc.Add(metric, stat)
	wc.updateQueue()
	return err
}

func (wc *Cacher) Add(name *repr.StatName, stat *repr.StatRepr) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	//wc.log.Critical("STAT: %s, %v", metric, stat)

	if !wc._accept {
		//wc.log.Error("Shutting down, will not add any more items to the queue")
		return nil
	}

	if len(wc.Cache) > wc.maxKeys {
		wc.log.Error("Key Cache is too large .. over %d metrics keys, have to drop this one", wc.maxKeys)
		stats.StatsdClientSlow.Incr("cacher.metrics.overflow", 1)
		return errWriteCacheTooManyMetrics
	}

	/** ye old debuggin'
	if strings.Contains(metric, "flushesposts") {
		wc.log.Critical("ADDING: %s Time: %d, Val: %f", metric, time, value)
	}
	*/
	unique_id := name.UniqueId()
	if gots, ok := wc.Cache[unique_id]; ok {
		cur_size := gots.Len()
		if gots.Len() > wc.maxBytes {

			// if the overflow method is chan, and there is valid overFLowChan, we "pop" the item from
			// the cache and send it to the chan (note we're already "locked" here)
			if wc.overFlowBroadcast != nil && wc.overFlowMethod == "chan" {

				nm := wc.NameCache[unique_id]
				wc.curSize -= int64(gots.Len()) // shrink the bytes
				delete(wc.Cache, unique_id)
				delete(wc.NameCache, unique_id)

				wc.overFlowBroadcast.Send(&TotalTimeSeries{nm, gots})
				stats.StatsdClientSlow.Incr("cacher.metrics.write.overflow", 1)
				//must recompute the ordering XXX LOCKING ISSUE
				//wc.updateQueue()

				// break out of this loop and add a new guy
				goto NEWSTAT
			}

			wc.log.Error("Too Many Bytes for %v (max bytes: %d current metrics: %v)... have to drop this one", unique_id, wc.maxBytes, gots.Count())
			stats.StatsdClientSlow.Incr("cacher.metics.points.overflow", 1)
			return errWriteCacheTooManyPoints
		}
		wc.Cache[unique_id].AddStat(stat)
		wc.NameCache[unique_id] = name
		now_len := wc.Cache[unique_id].Len()
		wc.curSize += int64(now_len - cur_size)
		stats.StatsdClient.GaugeAvg("cacher.add.ave-points-per-metric", int64(gots.Count()))
		return nil
	}

NEWSTAT:
	tp, err := series.NewTimeSeries(wc.seriesType, stat.Time.UnixNano(), nil)
	if err != nil {
		return err
	}

	tp.AddStat(stat)

	wc.Cache[unique_id] = tp
	wc.NameCache[unique_id] = name

	wc.curSize += int64(tp.Len())
	stats.StatsdClient.GaugeAvg("cacher.add.ave-points-per-metric", 1)

	return nil
}

func (wc *Cacher) getStatsStream(name *repr.StatName, ts series.TimeSeries) (repr.StatReprSlice, error) {
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
		st.Name = *name
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

func (wc *Cacher) Get(name *repr.StatName) (repr.StatReprSlice, error) {
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets", 1)

	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if gots, ok := wc.Cache[name.UniqueId()]; ok {
		stats.StatsdClientSlow.Incr("cacher.read.cache-gets.values", 1)
		return wc.getStatsStream(name, gots)
	}
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets.empty", 1)
	return nil, nil
}

func (wc *Cacher) GetAsRawRenderItem(name *repr.StatName) (*RawRenderItem, error) {

	name, data, err := wc.GetById(name.UniqueId())
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	rawd := new(RawRenderItem)
	rawd.Start = uint32(data[0].Time.Unix())
	rawd.End = uint32(data[len(data)-1].Time.Unix())
	rawd.RealEnd = rawd.End
	rawd.RealStart = rawd.Start
	rawd.AggFunc = name.AggType()

	f_t := uint32(0)
	step_t := uint32(0)

	rawd.Data = make([]RawDataPoint, len(data), len(data))
	for idx, pt := range data {
		on_t := uint32(pt.Time.Unix())
		rawd.Data[idx] = RawDataPoint{
			Time:  on_t,
			Count: pt.Count,
			Min:   float64(pt.Min),
			Max:   float64(pt.Max),
			Last:  float64(pt.Last),
			Sum:   float64(pt.Sum),
		}
		if f_t <= 0 {
			f_t = on_t
		}
		if step_t <= 0 && f_t >= 0 {
			step_t = on_t - f_t
		}
	}
	rawd.Step = step_t
	return rawd, nil
}

func (wc *Cacher) GetById(metric_id repr.StatId) (*repr.StatName, repr.StatReprSlice, error) {
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets", 1)

	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if gots, ok := wc.Cache[metric_id]; ok {
		stats.StatsdClientSlow.Incr("cacher.read.cache-gets.values", 1)
		nm := wc.NameCache[metric_id]
		tseries, err := wc.getStatsStream(nm, gots)
		return nm, tseries, err
	}
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets.empty", 1)
	return nil, nil, nil
}

func (wc *Cacher) GetSeries(name *repr.StatName) (*repr.StatName, series.TimeSeries, error) {
	stats.StatsdClientSlow.Incr("cacher.read.cache-series-gets", 1)

	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if gots, ok := wc.Cache[name.UniqueId()]; ok {
		stats.StatsdClientSlow.Incr("cacher.read.cache-series-gets.values", 1)
		return wc.NameCache[name.UniqueId()], gots, nil
	}
	stats.StatsdClientSlow.Incr("cacher.read.cache-series-gets.empty", 1)
	return nil, nil, nil
}

func (wc *Cacher) GetSeriesById(metric_id repr.StatId) (*repr.StatName, series.TimeSeries, error) {
	stats.StatsdClientSlow.Incr("cacher.read.cache-series-by-idgets", 1)

	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if gots, ok := wc.Cache[metric_id]; ok {
		stats.StatsdClientSlow.Incr("cacher.read.cache-series-by-idgets.values", 1)
		return wc.NameCache[metric_id], gots, nil
	}
	stats.StatsdClientSlow.Incr("cacher.read.cache-series-by-idgets.empty", 1)
	return nil, nil, nil
}

func (wc *Cacher) GetNextMetric() *repr.StatName {

	for {
		wc.qmu.Lock()
		size := len(wc.Queue)
		if size == 0 {
			wc.qmu.Unlock()
			break
		}
		item := wc.Queue[size-1]
		wc.Queue = wc.Queue[:size-1]
		wc.qmu.Unlock()

		wc.mu.RLock()
		v := wc.NameCache[item.metric]
		wc.mu.RUnlock()
		return v
	}
	return nil
}

// just grab something from the list
func (wc *Cacher) GetAnyStat() (name *repr.StatName, stats repr.StatReprSlice) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	for uid, stats := range wc.Cache {
		name = wc.NameCache[uid]
		out, _ := wc.getStatsStream(name, stats)
		wc.curSize -= int64(stats.Len()) // shrink the bytes
		delete(wc.Cache, uid)            // need to purge if error as things are corrupted somehow
		delete(wc.NameCache, uid)
		return name, out
	}
	return nil, nil
}

func (wc *Cacher) Pop() (*repr.StatName, repr.StatReprSlice) {
	metric, ts := wc.PopSeries()
	if ts == nil {
		return nil, nil
	}
	out, _ := wc.getStatsStream(metric, ts)
	return metric, out
}

func (wc *Cacher) PopSeries() (*repr.StatName, series.TimeSeries) {
	metric := wc.GetNextMetric()
	if metric != nil {
		wc.mu.Lock()
		defer wc.mu.Unlock()
		unique_id := metric.UniqueId()
		if stats, exists := wc.Cache[unique_id]; exists {
			wc.curSize -= int64(stats.Len()) // shrink the bytes
			delete(wc.Cache, unique_id)      // need to delete regardless as we have byte errors and things are corrupted
			delete(wc.NameCache, unique_id)
			return metric, stats
		}
	}
	return nil, nil
}

// add a metrics/point list back on the queue as it either "failed" or was ratelimited
func (wc *Cacher) AddBack(name *repr.StatName, points repr.StatReprSlice) {
	for _, pt := range points {
		wc.Add(name, pt)
	}
}
