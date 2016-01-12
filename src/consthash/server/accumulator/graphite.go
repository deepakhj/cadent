/*
   Here we accumulate graphite metrics and then push to a output format of whatever
   basically an internal graphite accumulator server
*/

package accumulator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

/****************** RUNNERS *********************/
const GRAPHITE_ACC_NAME = "graphite_accumlator"
const GRAHPITE_ACC_MIN_LEN = 3

/** counter/gauge type **/
// for sorting
type graphiteFloat64 []float64

func (a graphiteFloat64) Len() int           { return len(a) }
func (a graphiteFloat64) Swap(i int, j int)  { a[i], a[j] = a[j], a[i] }
func (a graphiteFloat64) Less(i, j int) bool { return (a[i] - a[j]) < 0 } //this is the sorting statsd uses for its timings

type AGG_TYPE func(graphiteFloat64) float64

var GRAPHITE_ACC_FUN = map[string]AGG_TYPE{
	"sum": func(vals graphiteFloat64) float64 {
		val := 0.0
		for _, item := range vals {
			val += item
		}
		return val
	},
	"avg": func(vals graphiteFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		val := 0.0
		for _, item := range vals {
			val += item
		}

		return val / float64(len(vals))
	},
	"max": func(vals graphiteFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		sort.Sort(vals)

		return vals[len(vals)-1]
	},
	"min": func(vals graphiteFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		sort.Sort(vals)

		return vals[0]
	},
	"first": func(vals graphiteFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		return vals[0]
	},
	"last": func(vals graphiteFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		return vals[len(vals)-1]
	},
}

type GraphiteBaseStatItem struct {
	InKey      string
	Values     graphiteFloat64
	InType     string
	ReduceFunc string

	mu sync.Mutex
}

func (s *GraphiteBaseStatItem) Type() string { return s.InType }
func (s *GraphiteBaseStatItem) Key() string  { return s.InKey }
func (s *GraphiteBaseStatItem) Out(fmatter FormatterItem, tags []AccumulatorTags) []string {
	val := GRAPHITE_ACC_FUN[s.ReduceFunc](s.Values)
	return []string{
		fmatter.ToString(
			s.InKey,
			val,
			0, // let formatter handle the time,
			"c",
			tags,
		),
	}
}

func (s *GraphiteBaseStatItem) Accumulate(val float64) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Values = append(s.Values, val)

	return nil
}

/******************************/
/** statsd accumulator **/
/******************************/

type GraphiteAccumulate struct {
	GraphiteStats map[string]StatItem
	OutFormat     FormatterItem
	InTags        []AccumulatorTags

	mu sync.Mutex
}

func NewGraphiteAccumulate() (*GraphiteAccumulate, error) {
	return new(GraphiteAccumulate), nil
}

func (s *GraphiteAccumulate) Tags() []AccumulatorTags {
	return s.InTags
}

func (s *GraphiteAccumulate) SetTags(tags []AccumulatorTags) {
	s.InTags = tags
}

func (s *GraphiteAccumulate) Init(fmatter FormatterItem) error {
	s.OutFormat = fmatter
	s.GraphiteStats = make(map[string]StatItem)
	return nil
}

func (s *GraphiteAccumulate) Stats() map[string]StatItem {
	return s.GraphiteStats
}

func (a *GraphiteAccumulate) Name() (name string) { return GRAPHITE_ACC_NAME }

func (a *GraphiteAccumulate) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.GraphiteStats = nil
	a.GraphiteStats = make(map[string]StatItem)
	return nil
}

func (a *GraphiteAccumulate) Flush() []string {
	base := []string{}
	a.mu.Lock()
	for _, stats := range a.GraphiteStats {
		base = append(base, stats.Out(a.OutFormat, a.Tags())...)
	}
	a.mu.Unlock()
	a.Reset()
	return base
}

func (a *GraphiteAccumulate) ProcessLine(line string) (err error) {
	//<key> <value> <time>

	stats_arr := strings.Fields(line)

	if len(stats_arr) < GRAHPITE_ACC_MIN_LEN {
		return fmt.Errorf("Accumulate: Invalid Graphite line `%s`", line)
	}

	//
	key := stats_arr[0]
	val := stats_arr[1]

	f_val, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fmt.Errorf("Accumulate: Bad Value | Invalid Graphite line `%s`", line)
	}

	stat_key := key
	// now the accumlator
	a.mu.Lock()
	gots, ok := a.GraphiteStats[stat_key]
	a.mu.Unlock()

	if !ok {
		def_agg := "avg"
		//some "tricks" to get the correct agg fun
		if strings.Contains(key, ".lower") || strings.Contains(key, ".min") {
			def_agg = "min"
		} else if strings.Contains(key, ".upper") || strings.Contains(key, ".max") {
			def_agg = "max"
		} else if strings.Contains(key, ".sum") {
			def_agg = "sum"
		} else if strings.Contains(key, ".gauges") || strings.Contains(key, ".abs") || strings.Contains(key, ".absolute") {
			def_agg = "last"
		} else if strings.Contains(key, ".counters") || strings.Contains(key, ".count") || strings.Contains(key, ".errors") {
			def_agg = "sum"
		}
		gots = &GraphiteBaseStatItem{
			InType:     "graphite",
			InKey:      key,
			ReduceFunc: def_agg,
		}
	}

	// needs to lock internally if needed
	gots.Accumulate(f_val)

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.GraphiteStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}
