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


	"time"
)

const (
	READ_CACHER_NUMBER_POINTS = 1024
	READ_CACHER_METRICS_KEYS  = 102400
)

type SortedStats  []*repr.StatRepr

func (v SortedStats) Len() int           { return len(v) }
func (v SortedStats) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v SortedStats) Less(i, j int) bool { return v[i] != nil && v[j] != nil && v[i].Time.UnixNano() < v[j].Time.UnixNano() }

// basic cached item treat it kinda like a round-robin array
type ReadCacheItem struct {
	Metric string
	StartTime time.Time
	EndTime	 time.Time
	LastAccess int64
	Data SortedStats

	MaxPoints int

	curindex int
	count  int // number of stats in it
}

func NewReadCacheItem(max_items int) (*ReadCacheItem){
	return &ReadCacheItem{
		MaxPoints: max_items,
		Data: make(SortedStats, max_items, max_items), //preallocate
		count: 0,
		curindex: 0,
	}
}

func (rc *ReadCacheItem) Add(stat *repr.StatRepr){
	if rc.curindex >= rc.MaxPoints{
		rc.curindex = 0
		rc.count = rc.MaxPoints
	}
	rc.Data[rc.curindex] = stat
	if stat.Time.After(rc.EndTime){
		rc.EndTime = stat.Time
	}
	if stat.Time.Before(rc.StartTime) || rc.StartTime.IsZero(){
		rc.StartTime = stat.Time
	}
	rc.curindex++
	rc.count++
}

func (rc *ReadCacheItem) Get(start time.Time, end time.Time) ([]*repr.StatRepr){

	// sort the land
	sort.Sort(rc.Data)
	set_idx := false
	out_arr := make([]*repr.StatRepr, 0)
	for idx, stat := range rc.Data{

		if stat == nil || stat.Time.IsZero(){
			continue
		}
		// set the current index to the "start" of real data
		if !set_idx{
			rc.curindex = idx
		}
		if stat.Time.After(start) && stat.Time.Before(end) {
			out_arr = append(out_arr, stat)
		}
	}

	rc.LastAccess = time.Now().UnixNano()
	return out_arr
}

