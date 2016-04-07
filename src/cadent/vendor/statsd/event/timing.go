package event

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// Timing keeps min/max/avg information about a timer over a certain interval
type Timing struct {
	Name string

	mu               sync.Mutex
	Min              int64
	Max              int64
	Value            int64
	Values           []int64
	Count            int64
	PercentThreshold []float64
}

func (e *Timing) StatClass() string {
	return "timer"
}

// for sorting
type statdInt64arr []int64

func (a statdInt64arr) Len() int           { return len(a) }
func (a statdInt64arr) Swap(i int, j int)  { a[i], a[j] = a[j], a[i] }
func (a statdInt64arr) Less(i, j int) bool { return (a[i] - a[j]) < 0 } //this is the sorting statsd uses for its timings

func round(a float64) float64 {
	if a < 0 {
		return math.Ceil(a - 0.5)
	}
	return math.Floor(a + 0.5)
}

// NewTiming is a factory for a Timing event, setting the Count to 1 to prevent div_by_0 errors
func NewTiming(k string, delta int64) *Timing {
	return &Timing{Name: k, Min: delta, Max: delta, Value: delta, Count: 1, PercentThreshold: []float64{0.9, 0.99}}
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *Timing) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e2.Payload().(map[string]interface{})
	e.Count += p["cnt"].(int64)
	e.Value += p["val"].(int64)
	e.Min = minInt64(e.Min, p["min"].(int64))
	e.Max = maxInt64(e.Max, p["max"].(int64))
	e.Values = append(e.Values, p["val"].(int64))
	return nil
}

//Reset the value
func (e *Timing) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Value = 0
	e.Count = 1
	e.Min = 0
	e.Max = 0
	e.Values = []int64{}
}

// Payload returns the aggregated value for this event
func (e Timing) Payload() interface{} {
	return map[string]interface{}{
		"min":  e.Min,
		"max":  e.Max,
		"val":  e.Value,
		"cnt":  e.Count,
		"vals": e.Values,
	}
}

// Stats returns an array of StatsD events as they travel over UDP
func (e Timing) Stats(tick time.Duration) []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	numInThreshold := int64(len(e.Values))
	val_len := numInThreshold

	std := float64(0)
	avg := float64(e.Value / e.Count)
	cumulativeValues := []int64{e.Min}

	sort.Sort(statdInt64arr(e.Values))

	for idx, v := range e.Values {
		std += math.Pow((float64(v) - avg), 2.0)
		if idx > 0 {
			cumulativeValues = append(cumulativeValues, v+cumulativeValues[idx-1])
		}

	}
	std = math.Sqrt(std / float64(e.Count))

	base := []string{
		fmt.Sprintf("%s.count:%d|c", e.Name, int64(val_len)),
		fmt.Sprintf("%s.count_ps:%d|c", e.Name, int64(float64(val_len)/float64(tick)/float64(time.Second))),
		fmt.Sprintf("%s.mean:%d|c", e.Name, int64(avg)), // make sure e.Count != 0
		fmt.Sprintf("%s.lower:%d|c", e.Name, e.Min),
		fmt.Sprintf("%s.upper:%d|c", e.Name, e.Max),
		fmt.Sprintf("%s.std:%d|c", e.Name, int64(std)),
		fmt.Sprintf("%s.sum:%d|c", e.Name, int64(e.Value)),
	}
	if e.Count > 0 {
		sum := e.Min
		mean := e.Min
		thresholdBoundary := e.Max

		mid := int(math.Floor(float64(e.Count) / 2.0))
		median := int64(0)
		if mid < len(e.Values) {
			if math.Mod(float64(mid), 2.0) == 0 {
				median = e.Values[mid]
			} else if len(e.Values) > 1 {
				median = (e.Values[mid-1] + e.Values[mid]) / 2.0
			}
		}
		base = append(base,
			fmt.Sprintf("%s.median:%d|c", e.Name, int64(median)),
		)
		for _, pct := range e.PercentThreshold {
			// handle 0.90 or 90%
			multi := 1.0 / 100.0
			per_mul := 1.0
			if math.Abs(pct) < 1 {
				multi = 1.0
				per_mul = 100.0
			}

			p_name := strings.Replace(fmt.Sprintf("%d", int(math.Abs(pct)*per_mul)), ".", "", -1)
			if val_len > 1 {
				numInThreshold := int64(round(float64(math.Abs(pct) * multi * float64(val_len))))
				if numInThreshold == 0 {
					continue
				}
				if pct > 0 {
					//fmt.Printf("%s thr:%d ct: %d vL: %d Cl: %d\n", e.Name, numInThreshold, e.Count, len(e.Values), len(cumulativeValues))
					thresholdBoundary = e.Values[numInThreshold-1]
					sum = cumulativeValues[numInThreshold-1]
				} else {
					thresholdBoundary = e.Values[val_len-numInThreshold]
					sum = cumulativeValues[val_len-1] - cumulativeValues[e.Count-numInThreshold-1]
				}

				mean = sum / numInThreshold
			}
			base = append(base,
				[]string{
					fmt.Sprintf("%s.count_%s:%d|c", e.Name, p_name, int64(numInThreshold)),
					fmt.Sprintf("%s.mean_%s:%d|c", e.Name, p_name, int64(mean)),
				}...,
			)
			if pct > 0 {
				base = append(base, fmt.Sprintf("%s.upper_%s:%d|c", e.Name, p_name, int64(thresholdBoundary)))
			} else {
				base = append(base, fmt.Sprintf("%s.lower_%s:%d|c", e.Name, p_name, int64(thresholdBoundary)))
			}
		}
	}
	return base
}

// Key returns the name of this metric
func (e Timing) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *Timing) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e Timing) Type() int {
	return EventTiming
}

// TypeString returns a name for this type of metric
func (e Timing) TypeString() string {
	return "Timing"
}

// String returns a debug-friendly representation of this metric
func (e Timing) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %+v}", e.TypeString(), e.Name, e.Payload())
}

func minInt64(v1, v2 int64) int64 {
	if v1 <= v2 {
		return v1
	}
	return v2
}
func maxInt64(v1, v2 int64) int64 {
	if v1 >= v2 {
		return v1
	}
	return v2
}
