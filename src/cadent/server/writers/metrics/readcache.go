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
		read_cache_metrics_points=1024 # number of points per metric to cache above to keep before we drop (this * cache_metric_size * 32 * 128 bytes == your better have that ram)
		...
*/

package metrics

import (
	"cadent/server/repr"

	"sort"

	"cadent/server/lrucache"
	"fmt"
	"sync"
	"time"
)

const (
	READ_CACHER_NUMBER_POINTS   = 1024
	READ_CACHER_METRICS_KEYS    = 102400
	READ_CACHER_MAX_LAST_ACCESS = time.Minute * time.Duration(60) // if an item is not accesed in 60 min purge it
)

type SortedStats []*repr.StatRepr

func (v SortedStats) Len() int      { return len(v) }
func (v SortedStats) Swap(i, j int) { v[i], v[j] = v[j], v[i] }
func (v SortedStats) Less(i, j int) bool {
	return v[i] != nil && v[j] != nil && v[i].Time.UnixNano() < v[j].Time.UnixNano()
}

// basic cached item treat it kinda like a round-robin array
type ReadCacheItem struct {
	Metric     string
	StartTime  time.Time
	EndTime    time.Time
	LastAccess int64
	Data       SortedStats

	MaxPoints int

	curindex int

	mu sync.RWMutex
}

func NewReadCacheItem(max_items int) *ReadCacheItem {
	return &ReadCacheItem{
		MaxPoints: max_items,
		Data:      make(SortedStats, max_items, max_items), //preallocate
		curindex:  0,
	}
}

func (rc *ReadCacheItem) Add(stat *repr.StatRepr) {
	if rc.curindex+1 >= rc.MaxPoints {
		rc.curindex = 0
	} else {
		rc.curindex++
	}
	rc.Data[rc.curindex] = stat
	if stat.Time.After(rc.EndTime) {
		rc.EndTime = stat.Time
	}
	if stat.Time.Before(rc.StartTime) || rc.StartTime.IsZero() {
		rc.StartTime = stat.Time
	}

}

// put a chunk o data
func (rc *ReadCacheItem) PutSeries(stats []*repr.StatRepr) {
	for _, s := range stats {
		rc.Add(s)
	}
}

// Get out a sorted list of stats by time
// note we only sort the list on a "get" so the current Data list can easily be
// out of order
func (rc *ReadCacheItem) Get(start time.Time, end time.Time) []*repr.StatRepr {

	// sort the land
	sort.Sort(rc.Data)
	set_idx := false
	out_arr := make([]*repr.StatRepr, 0)
	var last_added *repr.StatRepr
	// our dedupe map
	rc.mu.RLock()
	for idx, stat := range rc.Data {

		if stat == nil || stat.Time.IsZero() {
			continue
		}
		// set the current index to the "start" of real data
		if !set_idx {
			rc.curindex = idx
		}
		// some dedupeing as we are sorted ...
		if last_added != nil && last_added.IsSameStat(stat) {
			continue
		}
		if stat.Time.After(start) && stat.Time.Before(end) {
			last_added = stat
			out_arr = append(out_arr, stat)
		}
	}

	rc.mu.RUnlock()

	rc.LastAccess = time.Now().UnixNano()
	return out_arr
}

func (rc *ReadCacheItem) GetAll() []*repr.StatRepr {
	sort.Sort(rc.Data)
	set_idx := false
	out_arr := make([]*repr.StatRepr, 0)
	var last_added *repr.StatRepr
	rc.mu.RLock()
	for idx, stat := range rc.Data {

		if stat == nil || stat.Time.IsZero() {
			continue
		}
		// set the current index to the "start" of real data
		if !set_idx {
			rc.curindex = idx
		}
		//dedupe
		if last_added != nil && last_added.IsSameStat(stat) {
			continue
		}
		last_added = stat
		out_arr = append(out_arr, stat)
	}
	rc.mu.RUnlock()

	rc.LastAccess = time.Now().UnixNano()
	return out_arr
}

// match the "value" interface for LRUcache
func (rc *ReadCacheItem) Size() int {
	sz := 0
	for _, s := range rc.Data {
		if s == nil {
			continue
		}
		if s.Time.IsZero() {
			continue
		}
		sz += int(s.ByteSize())
	}
	return sz
}

func (rc *ReadCacheItem) ToString() string {
	return fmt.Sprintf("ReadCacheItem: max: %d", rc.MaxPoints)
}

// LRU read cache

type ReadCache struct {
	lru *lrucache.LRUCache

	MaxItems          int
	MaxItemsPerMetric int
	MaxLastAccess     time.Duration // we periodically prune the cache or things that have not been accessed in ths duration

	shutdown    chan bool
	InsertQueue chan *repr.StatRepr
}

// this will estimate the bytes needed for the cache, since the key names
// are dynamic, there is no way to "really know" much one "stat" ram will take up
// so we use a 100 char string as a rough guess
func NewReadCache(max_items int, max_items_per_metric int, maxback time.Duration) *ReadCache {
	rc := &ReadCache{
		MaxItems:          max_items,
		MaxItemsPerMetric: max_items_per_metric,
		shutdown:          make(chan bool),
		InsertQueue:       make(chan *repr.StatRepr, 512), // a little buffer, just more to make "adding async"
	}

	_long_tmp_key := "iiiiiiiiii.aaaaaaaaaaa.mmmmmmmmm.aaaaaaaaa.looooog.sssssttttaaattttt.tooooobeee.esssttiiiimmmaatted"
	dummy_stat := repr.StatRepr{
		Key:     _long_tmp_key,
		StatKey: _long_tmp_key,
	}
	// lru capacity is the size of a stat object * MaxItemsPerMetric * MaxItems
	rc.lru = lrucache.NewLRUCache(uint64(int(dummy_stat.ByteSize()) * max_items * max_items_per_metric))

	go rc.Start()

	return rc
}

// start up the insert queue
func (rc *ReadCache) Start() {
	for {
		select {
		case stat := <-rc.InsertQueue:
			rc.Put(stat.Key, stat)
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
		rc_item := NewReadCacheItem(rc.MaxItemsPerMetric)
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
		metric = stat.StatKey
	}
	if len(metric) == 0 {
		metric = stat.Key
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

func (rc *ReadCache) Get(metric string, start time.Time, end time.Time) (stats []*repr.StatRepr) {
	gots, ok := rc.lru.Get(metric)
	if !ok {
		return stats
	}
	return gots.(*ReadCacheItem).Get(start, end)
}

func (rc *ReadCache) GetAll(metric string) (stats []*repr.StatRepr) {
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

func Get(metric string, start time.Time, end time.Time) (stats []*repr.StatRepr) {
	if _READ_CACHE_SINGLETON != nil {
		return _READ_CACHE_SINGLETON.Get(metric, start, end)
	}
	return nil
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
