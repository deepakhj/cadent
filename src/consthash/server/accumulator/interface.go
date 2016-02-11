/*
   Here we accumulate statsd metrics and then push to a output format of whatever
*/

package accumulator

import (
	"fmt"
	"math"
)

/****************** Interfaces *********************/
type jsonFloat64 float64

// needed to handle "Inf" values
func (s jsonFloat64) MarshalJSON() ([]byte, error) {
	if math.IsInf(float64(s), 0) || float64(s) == math.MinInt64 {
		return []byte("0.0"), nil
	}
	return []byte(fmt.Sprintf("%v", float64(s))), nil
}

type StatRepr struct {
	Key     string      `json:"key"`
	StatKey string      `json:"stat_key"`
	Min     jsonFloat64 `json:"min"`
	Max     jsonFloat64 `json:"max"`
	Sum     jsonFloat64 `json:"sum"`
	Mean    jsonFloat64 `json:"mean"`
	Count   int64       `json:"count"`
	Time    int64       `json:"time_ns"`
}

type StatItem interface {
	Key() string
	Type() string
	Out(fmatter FormatterItem, acc AccumulatorItem) []string
	Accumulate(val float64) error
	ZeroOut() error
	Repr() StatRepr
}

type AccumulatorItem interface {
	Init(FormatterItem) error
	Stats() map[string]StatItem
	Flush() []string
	Name() string
	ProcessLine(string) error
	Reset() error
	Tags() []AccumulatorTags
	SetKeepKeys(bool) error
	SetTags([]AccumulatorTags)
	SetOptions([][]string) error
	GetOption(name string, defaults interface{}) interface{}
}

type FormatterItem interface {
	ToString(key string, val float64, tstamp int32, stats_type string, tags []AccumulatorTags) string
	Type() string
	Init(...string) error
	SetAccumulator(AccumulatorItem)
	GetAccumulator() AccumulatorItem
}
