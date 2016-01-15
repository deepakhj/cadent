/*
   Here we accumulate statsd metrics and then push to a output format of whatever
*/

package accumulator

/****************** Interfaces *********************/

type StatItem interface {
	Key() string
	Type() string
	Out(fmatter FormatterItem, acc AccumulatorItem) []string
	Accumulate(val float64) error
}

type AccumulatorItem interface {
	Init(FormatterItem) error
	Stats() map[string]StatItem
	Flush() []string
	Name() string
	ProcessLine(string) error
	Reset() error
	Tags() []AccumulatorTags
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
