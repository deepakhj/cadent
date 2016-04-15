/*
   Here we accumulate graphite metrics and then push to a output format of whatever
   basically an internal graphite accumulator server
*/

package accumulator

import (
	"cadent/server/repr"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

/****************** RUNNERS *********************/
const GRAPHITE_ACC_NAME = "graphite_accumlator"
const GRAHPITE_ACC_MIN_LEN = 3
const GRAPHITE_ACC_MIN_FLAG = math.MinInt64

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

	Min   float64
	Max   float64
	Mean  float64
	Sum   float64
	First float64
	Last  float64
	Count int64

	mu sync.Mutex
}

func (s *GraphiteBaseStatItem) Repr() repr.StatRepr {
	return repr.StatRepr{
		Key:   s.InKey,
		Min:   repr.CheckFloat(repr.JsonFloat64(s.Min)),
		Max:   repr.CheckFloat(repr.JsonFloat64(s.Max)),
		Count: s.Count,
		Mean:  repr.CheckFloat(repr.JsonFloat64(s.Mean)),
		Sum:   repr.CheckFloat(repr.JsonFloat64(s.Sum)),
		First: repr.CheckFloat(repr.JsonFloat64(s.First)),
		Last:  repr.CheckFloat(repr.JsonFloat64(s.Last)),
	}
}

func (s *GraphiteBaseStatItem) Type() string { return s.InType }
func (s *GraphiteBaseStatItem) Key() string  { return s.InKey }

func (s *GraphiteBaseStatItem) ZeroOut() error {
	// reset the values
	s.Values = graphiteFloat64{}
	s.Min = GRAPHITE_ACC_MIN_FLAG
	s.Mean = 0.0
	s.Max = GRAPHITE_ACC_MIN_FLAG
	s.Sum = 0.0
	s.Count = 0
	s.First = GRAPHITE_ACC_MIN_FLAG
	s.Last = GRAPHITE_ACC_MIN_FLAG
	return nil
}

func (s *GraphiteBaseStatItem) Out(fmatter FormatterItem, acc AccumulatorItem) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	val := GRAPHITE_ACC_FUN[s.ReduceFunc](s.Values)
	return []string{
		fmatter.ToString(
			s.InKey,
			val,
			0, // let formatter handle the time,
			"c",
			acc.Tags(),
		),
	}
}

func (s *GraphiteBaseStatItem) Accumulate(val float64, sample float64) error {
	if math.IsInf(val, 0) || math.IsNaN(val) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Values = append(s.Values, val)
	if s.Min == GRAPHITE_ACC_MIN_FLAG || s.Min > val {
		s.Min = val
	}
	if s.Max == GRAPHITE_ACC_MIN_FLAG || s.Max < val {
		s.Max = val
	}

	s.Count += 1
	s.Sum += val
	s.Mean = s.Sum / float64(s.Count)
	s.Last = val
	if s.First == GRAPHITE_ACC_MIN_FLAG {
		s.First = val
	}
	return nil
}

/******************************/
/** statsd accumulator **/
/******************************/

type GraphiteAccumulate struct {
	GraphiteStats map[string]StatItem
	OutFormat     FormatterItem
	InTags        []AccumulatorTags
	InKeepKeys    bool

	mu sync.Mutex
}

func NewGraphiteAccumulate() (*GraphiteAccumulate, error) {
	return new(GraphiteAccumulate), nil
}

func (s *GraphiteAccumulate) SetOptions(ops [][]string) error {
	return nil
}
func (s *GraphiteAccumulate) GetOption(opt string, defaults interface{}) interface{} {
	return defaults
}

func (s *GraphiteAccumulate) Tags() []AccumulatorTags {
	return s.InTags
}

func (s *GraphiteAccumulate) SetTags(tags []AccumulatorTags) {
	s.InTags = tags
}

func (s *GraphiteAccumulate) SetKeepKeys(k bool) error {
	s.InKeepKeys = k
	return nil
}

func (s *GraphiteAccumulate) Init(fmatter FormatterItem) error {
	s.OutFormat = fmatter
	fmatter.SetAccumulator(s)
	s.GraphiteStats = make(map[string]StatItem)
	s.SetOptions([][]string{})
	return nil
}

func (s *GraphiteAccumulate) Stats() map[string]StatItem {
	return s.GraphiteStats
}

func (a *GraphiteAccumulate) Name() (name string) { return GRAPHITE_ACC_NAME }

func (a *GraphiteAccumulate) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// keep or reset
	if a.InKeepKeys {
		for idx := range a.GraphiteStats {
			a.GraphiteStats[idx].ZeroOut()
		}
	} else {
		a.GraphiteStats = nil
		a.GraphiteStats = make(map[string]StatItem)
	}

	return nil
}

func (a *GraphiteAccumulate) Flush() *flushedList {
	fl := new(flushedList)

	a.mu.Lock()
	for _, stats := range a.GraphiteStats {
		fl.Add(stats.Out(a.OutFormat, a), stats.Repr())
	}
	a.mu.Unlock()
	a.Reset()
	return fl
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
			Min:        GRAPHITE_ACC_MIN_FLAG,
			Max:        GRAPHITE_ACC_MIN_FLAG,
			First:      GRAPHITE_ACC_MIN_FLAG,
			Last:       GRAPHITE_ACC_MIN_FLAG,
			ReduceFunc: def_agg,
		}
	}

	// needs to lock internally if needed
	gots.Accumulate(f_val, 1.0)

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.GraphiteStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}
