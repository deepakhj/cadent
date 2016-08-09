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
	mu sync.Mutex

	Items      map[string]StatRepr
	Resolution time.Duration
}

func NewAggregator(res time.Duration) *Aggregator {
	return &Aggregator{
		Resolution: res,
		Items:      make(map[string]StatRepr),
	}
}

// based on the resolution we need to aggregate around a
// "key+time bucket" mix.  to figure out the time bucket
// we simply use the resolution -- time % resolution
func (sa *Aggregator) ResolutionTime(t time.Time) time.Time {
	return t.Truncate(sa.Resolution)
}

func (sa *Aggregator) MapKey(name string, t time.Time) string {
	return fmt.Sprintf("%s-%d", name, sa.ResolutionTime(t).UnixNano())
}

func (sa *Aggregator) Len() int {
	return len(sa.Items)
}

// get the data and clear out the current cache
func (sa *Aggregator) GetAndClear() map[string]StatRepr {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	t_items := make(map[string]StatRepr)
	for k, v := range sa.Items {
		t_items[k] = v
		//delete(sa.Items, k)
	}
	sa.Items = make(map[string]StatRepr)
	return t_items
}

func (sa *Aggregator) Add(stat StatRepr) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	res_time := sa.ResolutionTime(stat.Time)
	m_k := sa.MapKey(stat.Name.UniqueIdString(), stat.Time)
	element, ok := sa.Items[m_k]
	if !ok {
		sa.Items[m_k] = stat
		return nil
	}

	element.Last = stat.Last
	element.First = stat.First

	element.Count += stat.Count
	if element.Max < stat.Max {
		element.Max = stat.Max
	}
	if element.Min > stat.Min {
		element.Min = stat.Min
	}
	element.Sum += stat.Sum
	element.Time = res_time
	element.Name.Resolution = uint32(sa.Resolution.Seconds())
	sa.Items[m_k] = element
	return nil
}

func (sa *Aggregator) Clear() {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.Items = make(map[string]StatRepr)
	/*
		for k := range sa.Items {
			delete(sa.Items, k)
		}*/
}
