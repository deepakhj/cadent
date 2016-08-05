/*
	ReadCacher

	For every "metric" read in the API, we add an entry here that's basically a bound list (read_cache_metrics_points)
	of time -> data elements.

	A Writer has the option to, upon receiving of new point, to add it to this read cache
	such that the read cache is always "hot" even if things are stuck in the writing buffer caches

	[acc.agg.writer.metrics]
	driver="blaa"
	dsn="blaa"
	[acc.agg.writer.metrics.options]
		...
		# ram requirements are  (read_cache_metrics_points * read_cache_max_bytes_per_metric) + some overhead
		read_cache_total_bytes=10485760  #(bytes: default 10Mb) max amount of ram to cache .. the number stored metrics is ~read_cache_max_bytes_per_metric/read_cache_total_ram
		read_cache_max_bytes_per_metric=8192 # number of bytes per metric to keep around
		...

	We use the "protobuf" series as
		a) it's pretty small, and
		b) we need something we can easily "slice" to drop out the old points
		c) we can do array slicing to keep a walking cache in order
		gorrilla is more compressed BUT w/ gorilla is NOT ok to be out-of-time and it's not "sliceable"

*/

package metrics

import (
	"cadent/server/lrucache"
	"cadent/server/repr"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/series"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	READ_CACHER_MAX_SERIES_BYTES = 8192
	READ_CACHER_TOTAL_RAM        = 10485760
	READ_CACHER_MAX_LAST_ACCESS  = time.Minute * time.Duration(60) // if an item is not accesed in 60 min purge it
)

// basic cached item treat it kinda like a round-robin array
type ReadCacheItem struct {
	MetricId   repr.StatId // repr.StatName.UniqueID()
	StartTime  time.Time
	EndTime    time.Time
	LastAccess int64
	Data       *series.ProtobufTimeSeries

	MaxBytes int

	mu sync.RWMutex
}

func NewReadCacheItem(max_bytes int) *ReadCacheItem {
	ts, _ := series.NewTimeSeries("protobuf", 0, nil)

	return &ReadCacheItem{
		MaxBytes: max_bytes,
		Data:     ts.(*series.ProtobufTimeSeries),
	}
}

func (rc *ReadCacheItem) Add(stat *repr.StatRepr) {

	if rc.Data.Len() > rc.MaxBytes {
		rc.Data.Stats.Stats = rc.Data.Stats.Stats[1:]
	}
	rc.Data.AddStat(stat)
	if stat.Time.After(rc.EndTime) {
		rc.EndTime = stat.Time
	}
	if stat.Time.Before(rc.StartTime) || rc.StartTime.IsZero() {
		rc.StartTime = stat.Time
	}
}

func (rc *ReadCacheItem) AddValues(t time.Time, min float64, max float64, first float64, last float64, sum float64, count int64) {

	if rc.Data.Len() > rc.MaxBytes {
		rc.Data.Stats.Stats = rc.Data.Stats.Stats[1:]
	}

	rc.Data.AddPoint(t.UnixNano(), min, max, first, last, sum, count)

	if t.After(rc.EndTime) {
		rc.EndTime = t
	}
	if t.Before(rc.StartTime) || rc.StartTime.IsZero() {
		rc.StartTime = t
	}
}

// put a chunk o data
func (rc *ReadCacheItem) PutSeries(stats repr.StatReprSlice) {
	sort.Sort(stats)
	for _, s := range stats {
		rc.Add(s)
	}
}

//out a "render" item

// put a chunk o data
func (rc *ReadCacheItem) PutRenderedSeries(data []RawDataPoint) {
	for _, s := range data {
		// since the RawDataPoint contains a "mean" .. and may or may not have the Sum/Count
		// if sum/count are 0, we make the "sum" the mean and a count of 1
		if s.Sum == 0.0 && s.Count <= 1 && s.Mean != 0.0 {
			s.Sum = s.Mean
			s.Count = 1
		}
		rc.AddValues(time.Unix(int64(s.Time), 0), s.Min, s.Max, s.First, s.Last, s.Sum, s.Count)
	}
}

// Get out a sorted list of stats by time
//
func (rc *ReadCacheItem) Get(start time.Time, end time.Time) (stats repr.StatReprSlice, firsttime time.Time, lasttime time.Time) {

	stats = make(repr.StatReprSlice, 0)

	it, err := rc.Data.Iter()

	if err != nil {
		log.Error("Cache Get Iterator Error: %v", err)
		return stats, time.Time{}, time.Time{}
	}
	s_int := start.UnixNano()
	e_int := end.UnixNano()

	for it.Next() {
		stat := it.ReprValue()

		if stat == nil || stat.Time.IsZero() {
			continue
		}
		t_int := stat.Time.UnixNano()
		if s_int <= t_int && e_int >= t_int {
			if firsttime.IsZero() {
				firsttime = stat.Time
			} else if stat.Time.After(firsttime) {
				firsttime = stat.Time
			}
			if lasttime.IsZero() {
				lasttime = stat.Time
			} else if stat.Time.Before(lasttime) {
				lasttime = stat.Time
			}

			stats = append(stats, stat)
		}
	}
	sort.Sort(stats)
	rc.LastAccess = time.Now().UnixNano()
	return stats, firsttime, lasttime
}

func (rc *ReadCacheItem) GetAll() repr.StatReprSlice {
	out_arr := make([]*repr.StatRepr, 0)
	it, err := rc.Data.Iter()
	if err != nil {
		return out_arr
	}
	for it.Next() {
		stat := it.ReprValue()
		if stat == nil || stat.Time.IsZero() {
			continue
		}

		out_arr = append(out_arr, stat)
	}
	rc.LastAccess = time.Now().UnixNano()
	return out_arr
}

// match the "value" interface for LRUcache
func (rc *ReadCacheItem) Size() int {
	return rc.Data.Len()
}

func (rc *ReadCacheItem) ToString() string {
	return fmt.Sprintf("ReadCacheItem: max: %d", rc.MaxBytes)
}

// LRU read cache

type ReadCache struct {
	lru *lrucache.LRUCache

	MaxBytes          int
	MaxBytesPerMetric int
	MaxLastAccess     time.Duration // we periodically prune the cache or things that have not been accessed in ths duration

	shutdown    chan bool
	InsertQueue chan *repr.StatRepr
}

// this will estimate the bytes needed for the cache, since the key names
// are dynamic, there is no way to "really know" much one "stat" ram will take up
// so we use a 100 char string as a rough guess
func NewReadCache(max_bytes int, max_bytes_per_metric int, maxback time.Duration) *ReadCache {
	rc := &ReadCache{
		MaxBytes:          max_bytes,
		MaxBytesPerMetric: max_bytes_per_metric,
		shutdown:          make(chan bool),
		InsertQueue:       make(chan *repr.StatRepr, 512), // a little buffer, just more to make "adding async"
	}

	// lru capacity is the size of a stat object * MaxItemsPerMetric * MaxItems
	rc.lru = lrucache.NewLRUCache(uint64(max_bytes))

	go rc.Start()

	return rc
}

// start up the insert queue
func (rc *ReadCache) Start() {
	for {
		select {
		case stat := <-rc.InsertQueue:
			rc.Put(stat.Name.Key, stat)
		case <-rc.shutdown:
			shutdown.ReleaseFromShutdown()
			return
		}
	}
}

func (rc *ReadCache) Stop() {
	shutdown.AddToShutdown()
	rc.shutdown <- true
}

// add a series to the cache .. this should only be called by a reader api
// or some 'pre-seed' mechanism
func (rc *ReadCache) ActivateMetric(metric string, stats []*repr.StatRepr) bool {

	_, ok := rc.lru.Get(metric)
	if !ok {
		rc_item := NewReadCacheItem(rc.MaxBytesPerMetric)
		// blank ones are ok, just to activate it
		if stats != nil {
			rc_item.PutSeries(stats)
		}
		rc.lru.Set(metric, rc_item)
		return true
	}
	return false // already activated
}

func (rc *ReadCache) ActivateMetricFromRenderData(data *RawRenderItem) bool {
	_, ok := rc.lru.Get(data.Metric)
	if !ok {
		rc_item := NewReadCacheItem(rc.MaxBytesPerMetric)
		// blank ones are ok, just to activate it
		if data != nil {
			rc_item.PutRenderedSeries(data.Data)
		}
		rc.lru.Set(data.Metric, rc_item)
		return true
	}

	return false // already activated
}

func (rc *ReadCache) PutSeries(metric string, stats []*repr.StatRepr) bool {
	item, ok := rc.lru.Get(metric)
	if !ok {
		return false
	}
	item.(*ReadCacheItem).PutSeries(stats)
	rc.lru.Set(metric, item)
	return true
}

func (rc *ReadCache) PutRenderedSeries(metric string, data []RawDataPoint) bool {
	item, ok := rc.lru.Get(metric)
	if !ok {
		return false
	}
	item.(*ReadCacheItem).PutRenderedSeries(data)
	rc.lru.Set(metric, item)
	return true
}

// if we DON'T have the metric yet, DO NOT put it, the reader api
// will put a "block" of points and basically tag the metric as "active"
// so the writers will add metrics to the cache as it's assumed to be used in the
// future
func (rc *ReadCache) Put(metric string, stat *repr.StatRepr) bool {
	if len(metric) == 0 {
		metric = stat.Name.Key
	}

	gots, ok := rc.lru.Get(metric)

	// only add it if it's been included
	if !ok {
		return false
	}

	gots.(*ReadCacheItem).Add(stat)
	rc.lru.Set(metric, gots)
	return true
}

func (rc *ReadCache) Get(metric string, start time.Time, end time.Time) (stats repr.StatReprSlice, first time.Time, last time.Time) {
	gots, ok := rc.lru.Get(metric)
	if !ok {
		return stats, first, last
	}
	return gots.(*ReadCacheItem).Get(start, end)
}

func (rc *ReadCache) GetAll(metric string) (stats repr.StatReprSlice) {
	gots, ok := rc.lru.Get(metric)
	if !ok {
		return stats
	}
	return gots.(*ReadCacheItem).GetAll()
}

func (rc *ReadCache) NumKeys() uint64 {
	length, _, _, _ := rc.lru.Stats()
	return length
}

func (rc *ReadCache) Size() uint64 {
	_, size, _, _ := rc.lru.Stats()
	return size
}

// since this is used for both the Writer and Readers, we need a proper singleton
// when the "reader api" initialized we grab one and only one of these caches
// a writer will then use this singleton to "push" things onto the metrics

var _READ_CACHE_SINGLETON *ReadCache

func InitReadCache(max_bytes int, max_bytes_per_metric int, maxback time.Duration) *ReadCache {
	if _READ_CACHE_SINGLETON != nil {
		return _READ_CACHE_SINGLETON
	}
	_READ_CACHE_SINGLETON = NewReadCache(max_bytes, max_bytes_per_metric, maxback)
	return _READ_CACHE_SINGLETON
}

func GetReadCache() *ReadCache {
	return _READ_CACHE_SINGLETON
}

func Get(metric string, start time.Time, end time.Time) (stats []*repr.StatRepr, first time.Time, last time.Time) {
	if _READ_CACHE_SINGLETON != nil {
		return _READ_CACHE_SINGLETON.Get(metric, start, end)
	}
	return nil, time.Time{}, time.Time{}
}

func Put(metric string, stat *repr.StatRepr) bool {
	if _READ_CACHE_SINGLETON != nil {
		return _READ_CACHE_SINGLETON.Put(metric, stat)
	}
	return false
}

func PutRenderedSeries(metric string, data []RawDataPoint) bool {
	if _READ_CACHE_SINGLETON != nil {
		return _READ_CACHE_SINGLETON.PutRenderedSeries(metric, data)
	}
	return false
}

func ActivateMetric(metric string, stats []*repr.StatRepr) bool {
	if _READ_CACHE_SINGLETON != nil {
		return _READ_CACHE_SINGLETON.ActivateMetric(metric, stats)
	}
	return false
}
