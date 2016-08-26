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
func (ma *MultiAggregator) Add(stat *StatRepr) error {
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
