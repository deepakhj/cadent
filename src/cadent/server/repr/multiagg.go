/*
   maintain multiple Aggregator Lists each with there own Resolution
   just that an Add moves things into their proper various aggrigation slots
*/

package repr

import (
	"sync"
	"time"
)

/** Multi Aggregator **/

type MultiAggregator struct {
	mu sync.Mutex

	Aggs        map[float64]*Aggregator
	Resolutions []time.Duration
}

func NewMulti(res []time.Duration) *MultiAggregator {
	ma := &MultiAggregator{
		Resolutions: res,
		Aggs:        make(map[float64]*Aggregator),
	}
	for _, dur := range res {
		t := dur.Seconds()
		ma.Aggs[t] = NewAggregator(dur)
	}
	return ma
}

func (ma *MultiAggregator) Get(dur time.Duration) *Aggregator {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	return ma.Aggs[dur.Seconds()]
}

func (ma *MultiAggregator) Len() int {
	return len(ma.Resolutions)
}

// Add one stat to all the queues
func (ma *MultiAggregator) Add(stat StatRepr) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	for _, dur := range ma.Resolutions {
		t := dur.Seconds()
		ma.Aggs[t].Add(stat)
	}
	return nil
}

func (ma *MultiAggregator) Clear(dur time.Duration) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.Aggs[dur.Seconds()].Clear()
}
