/*
   Here we accumulate statsd metrics and then push to a output format of whatever
*/

package accumulator

/****************** Interfaces *********************/

type StatItem interface {
	Key() string
	Type() string
	Out(fmatter FormatterItem, tags []AccumulatorTags) []string
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
}

type FormatterItem interface {
	ToString(key string, val float64, tstamp int32, stats_type string, tags []AccumulatorTags) string
	Type() string

	Init(...string) error
}
