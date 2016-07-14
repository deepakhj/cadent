/*
	ReadCacher

	For every "metric" read , we add an entry here that's basically a bound list (read_cache_metrics_points)
	of time -> data elements.

	The "writer" will then add points to these items so that (if the time span is in range) the reader
	will simply use this internall cache buffer

	[acc.agg.writer.metrics]
	driver="blaa"
	dsn="blaa"
	[acc.agg.writer.metrics.options]
		...
		read_cache_metrics_points=102400  # number of metric strings to keep
		read_cache_metrics_series_bytes=1024 # number of bytes per metric to keep around
		...

	We use the "protobuf" series as
		a) it's pretty small, and
		b) we can do array slicing to keep a walking cache in order
		(gorrilla is more compressed, but much harder to deal w/ the byte size issue as it's highly variable)
		and w/ gorilla is NOT ok to be out-of-time

*/

package metrics

import (
	"cadent/server/lrucache"
	"cadent/server/repr"
	"cadent/server/writers/series"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	READ_CACHER_MAX_SERIES_BYTES = 1024
	READ_CACHER_METRICS_KEYS     = 102400
	READ_CACHER_MAX_LAST_ACCESS  = time.Minute * time.Duration(60) // if an item is not accesed in 60 min purge it
)

// basic cached item treat it kinda like a round-robin array
type ReadCacheItem struct {
	Metric     string
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

// put a chunk o data
func (rc *ReadCacheItem) PutSeries(stats repr.StatReprSlice) {
	sort.Sort(stats)
	for _, s := range stats {
		rc.Add(s)
	}
}

// Get out a sorted list of stats by time
//
func (rc *ReadCacheItem) Get(start time.Time, end time.Time) (stats repr.StatReprSlice, firsttime time.Time, lasttime time.Time) {

	stats = make(repr.StatReprSlice, 0)
	var last_added *repr.StatRepr

	it, err := rc.Data.Iter()

	if err != nil {
		log.Error("Cache Get Itterator Error: %v", err)
		return stats, time.Time{}, time.Time{}
	}
	s_int := start.UnixNano()
	e_int := end.UnixNano()

	for it.Next() {
		stat := it.ReprValue()
		if stat == nil || stat.Time.IsZero() {
			continue
		}

		// some de-dupeing
		if last_added != nil && last_added.IsSameStat(stat) {
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

			last_added = stat
			stats = append(stats, stat)
		}
	}
	sort.Sort(stats)
	rc.LastAccess = time.Now().UnixNano()
	return stats, firsttime, lasttime
}

func (rc *ReadCacheItem) GetAll() repr.StatReprSlice {
	out_arr := make([]*repr.StatRepr, 0)
	var last_added *repr.StatRepr
	it, err := rc.Data.Iter()
	if err != nil {
		return out_arr
	}
	for it.Next() {
		stat := it.ReprValue()
		if stat == nil || stat.Time.IsZero() {
			continue
		}
		// some dedupeing
		if last_added != nil && last_added.IsSameStat(stat) {
			continue
		}
		last_added = stat
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

	MaxItems          int
	MaxBytesPerMetric int
	MaxLastAccess     time.Duration // we periodically prune the cache or things that have not been accessed in ths duration

	shutdown    chan bool
	InsertQueue chan *repr.StatRepr
}

// this will estimate the bytes needed for the cache, since the key names
// are dynamic, there is no way to "really know" much one "stat" ram will take up
// so we use a 100 char string as a rough guess
func NewReadCache(max_items int, max_bytes_per_metric int, maxback time.Duration) *ReadCache {
	rc := &ReadCache{
		MaxItems:          max_items,
		MaxBytesPerMetric: max_bytes_per_metric,
		shutdown:          make(chan bool),
		InsertQueue:       make(chan *repr.StatRepr, 512), // a little buffer, just more to make "adding async"
	}

	// lru capacity is the size of a stat object * MaxItemsPerMetric * MaxItems
	rc.lru = lrucache.NewLRUCache(uint64(max_bytes_per_metric * max_items))

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
			break
		}
	}
	return
}

func (rc *ReadCache) Stop() {
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

// if we DON'T have the metric yet, DO NOT put it, the reader api
// will put a "block" of points and basically tag the metric as "active"
// so the writers will add metrics to the cache as it's assumed to be used in the
// future
func (rc *ReadCache) Put(metric string, stat *repr.StatRepr) bool {
	if len(metric) == 0 {
		metric = stat.Name.StatKey
	}
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
		return stats, time.Time{}, time.Time{}
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

func InitReadCache(max_items int, max_items_per_metric int, maxback time.Duration) *ReadCache {
	if _READ_CACHE_SINGLETON != nil {
		return _READ_CACHE_SINGLETON
	}
	_READ_CACHE_SINGLETON = NewReadCache(max_items, max_items_per_metric, maxback)
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

func ActivateMetric(metric string, stats []*repr.StatRepr) bool {
	if _READ_CACHE_SINGLETON != nil {
		return _READ_CACHE_SINGLETON.ActivateMetric(metric, stats)
	}
	return false
}
