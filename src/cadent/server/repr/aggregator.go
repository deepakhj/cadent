/*
   maintain a single list of a given resolution of stats
*/

package repr

import (
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

func (sa *Aggregator) Len() int {
	return len(sa.Items)
}

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

	element, ok := sa.Items[stat.Key]
	if !ok {
		sa.Items[stat.Key] = stat
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
	element.Mean = JsonFloat64(float64(element.Sum) / float64(element.Count))
	element.Time = stat.Time
	element.Resolution = sa.Resolution.Seconds()
	sa.Items[stat.Key] = element
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
