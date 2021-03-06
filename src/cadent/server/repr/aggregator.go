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
   maintain a single list of a given resolution of stats
*/

package repr

import (
	"fmt"
	"sync"
	"time"
)

// this will allow us to aggregate the initial accumulated stats
// obviously this is ram intensive for many many unique keys
// So if spreading out the accumulation across multiple nodes, make sure each stat key
// is consistently hashed to a single instance to make the aggregator work

type Aggregator struct {
	mu sync.RWMutex

	Items      map[string]*StatRepr
	Resolution time.Duration
}

func NewAggregator(res time.Duration) *Aggregator {
	return &Aggregator{
		Resolution: res,
		Items:      make(map[string]*StatRepr),
	}
}

// based on the resolution we need to aggregate around a
// "key+time bucket" mix.  to figure out the time bucket
// we simply use the resolution -- time % resolution
func (sa *Aggregator) ResolutionTime(t time.Time) time.Time {
	return t.Truncate(sa.Resolution)
}

func (sa *Aggregator) MapKey(id string, t time.Time) string {
	return id + fmt.Sprintf("-%d", sa.ResolutionTime(t).UnixNano())
}

func (sa *Aggregator) Len() int {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return len(sa.Items)
}

// get the data and clear out the current cache
func (sa *Aggregator) GetAndClear() map[string]*StatRepr {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	t_items := make(map[string]*StatRepr)
	for k, v := range sa.Items {
		t_items[k] = v
		//delete(sa.Items, k)
	}
	sa.Items = make(map[string]*StatRepr)
	return t_items
}

func (sa *Aggregator) Add(stat *StatRepr) error {

	t_t := stat.ToTime()
	res_time := sa.ResolutionTime(t_t)
	m_k := sa.MapKey(stat.Name.UniqueIdString(), t_t)
	sa.mu.RLock()
	element, ok := sa.Items[m_k]
	sa.mu.RUnlock()

	if !ok {
		sa.mu.Lock()
		sa.Items[m_k] = stat.Copy()
		sa.mu.Unlock()

		return nil
	}
	//sa.mu.Lock()
	//defer sa.mu.Unlock()

	element.Last = stat.Last

	element.Count += stat.Count
	if element.Max < stat.Max {
		element.Max = stat.Max
	}
	if element.Min > stat.Min {
		element.Min = stat.Min
	}
	element.Sum += stat.Sum
	element.Time = res_time.UnixNano()
	element.Name.Resolution = uint32(sa.Resolution.Seconds())
	//sa.Items[m_k] = element

	return nil
}

func (sa *Aggregator) Clear() {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.Items = make(map[string]*StatRepr)
	/*
		for k := range sa.Items {
			delete(sa.Items, k)
		}*/
}
