/*
   Here we accumulate statsd metrics and then push to a output format of whatever
*/

package accumulator

/****************** Interfaces *********************/

type StatItem interface {
	Key() string
	Type() string
	Out(fmatter FormaterItem) []string
	Accumulate(val float64) error
}

type AccumulatorItem interface {
	Stats() map[string]StatItem
	Flush() []string
	Name() string
	ProcessLine(string) error
	Reset() error
}

type FormaterItem interface {
	ToString(key string, val float64, tstamp int32, stats_type string, tags map[string]string) string
	Type() string
}
