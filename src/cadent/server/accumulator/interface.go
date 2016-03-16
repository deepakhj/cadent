/*
   Here we accumulate statsd metrics and then push to a output format of whatever
*/

package accumulator

import (
	"cadent/server/repr"
)

/****************** Interfaces *********************/
type StatItem interface {
	Key() string
	Type() string
	Out(fmatter FormatterItem, acc AccumulatorItem) []string
	Accumulate(val float64) error
	ZeroOut() error
	Repr() repr.StatRepr
}

type AccumulatorItem interface {
	Init(FormatterItem) error
	Stats() map[string]StatItem
	Flush() *flushedList
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

// This is an internal struct used for the Accumulator to get both lines and StatReprs on a Flush
type flushedList struct {
	Lines []string
	Stats []repr.StatRepr
}

func (fl *flushedList) Add(lines []string, stat repr.StatRepr) {
	fl.Lines = append(fl.Lines, lines...)
	fl.Stats = append(fl.Stats, stat)
}
