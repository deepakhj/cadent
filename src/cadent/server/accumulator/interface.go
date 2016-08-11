/*
   Here we accumulate statsd metrics and then push to a output format of whatever
*/

package accumulator

import (
	"cadent/server/repr"
	"io"
	"time"
)

/****************** Interfaces *********************/
type StatItem interface {
	Key() repr.StatName
	StatTime() time.Time
	Type() string
	Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem)
	Accumulate(val float64, sample float64, stattime time.Time) error
	ZeroOut() error
	Repr() *repr.StatRepr
}

type AccumulatorItem interface {
	Init(FormatterItem) error
	Stats() map[string]StatItem
	Flush(buf io.Writer) *flushedList
	Name() string
	ProcessLine(string) error
	Reset() error
	Tags() []AccumulatorTags
	SetKeepKeys(bool) error
	SetTags([]AccumulatorTags)
	SetResolution(time.Duration) error
	GetResolution() time.Duration
	SetOptions([][]string) error
	GetOption(name string, defaults interface{}) interface{}
}

type FormatterItem interface {
	ToString(name *repr.StatName, val float64, tstamp int32, stats_type string, tags []AccumulatorTags) string
	Write(buf io.Writer, name *repr.StatName, val float64, tstamp int32, stats_type string, tags []AccumulatorTags)
	Type() string
	Init(...string) error
	SetAccumulator(AccumulatorItem)
	GetAccumulator() AccumulatorItem
}

// This is an internal struct used for the Accumulator to get both lines and StatReprs on a Flush
type flushedList struct {
	Lines []string
	Stats []*repr.StatRepr
}

func (fl *flushedList) Add(lines []string, stat *repr.StatRepr) {
	fl.Lines = append(fl.Lines, lines...)
	fl.Stats = append(fl.Stats, stat)
}

func (fl *flushedList) AddStat(stat *repr.StatRepr) {
	fl.Stats = append(fl.Stats, stat)
}
