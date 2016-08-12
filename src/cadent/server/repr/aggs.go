/*
   An aggregator functions stat object
   and "guessing" agg function from the string or stat type
*/

package repr

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

var _upperReg *regexp.Regexp
var _upperMaxReg *regexp.Regexp
var _lowerReg *regexp.Regexp
var _lowerMinReg *regexp.Regexp
var _medianReg *regexp.Regexp

func init() {
	_upperReg, _ = regexp.Compile(".*upper_[0-9]+$")
	_upperMaxReg, _ = regexp.Compile(".*max_[0-9]+$")
	_lowerReg, _ = regexp.Compile(".*lower_[0-9]+$")
	_lowerMinReg, _ = regexp.Compile(".*min_[0-9]+$")
	_medianReg, _ = regexp.Compile(".*median_[0-9]+$")
}

type AggType uint8

const (
	MEAN AggType = iota
	SUM
	FIRST
	LAST
	MIN
	MAX
	STD
	MEDIAN
)

// if there is a tag that has the agg func in it
func AggTypeFromTag(stat string) AggType {
	switch {
	case stat == "min" || stat == "lower" || _lowerMinReg.MatchString(stat) || _lowerReg.MatchString(stat):
		return MIN
	case stat == "max" || stat == "upper" || _upperReg.MatchString(stat) || _upperMaxReg.MatchString(stat):
		return MAX
	case stat == "sum" || stat == "count" || stat == "requests":
		return SUM
	case stat == "gauge" || stat == "abs" || stat == "absolute":
		return LAST
	case stat == "std":
		return STD
	case stat == "median" || stat == "middle":
		return MEDIAN
	default:
		return MEAN
	}
}

func AggFuncFromTag(stat string) AGG_FUNC {
	return ACCUMULATE_FUNC[AggTypeFromTag(stat)]
}

// guess the agg func from the my.metric.is.good string
func GuessReprValueFromKey(metric string) AggType {
	spl := strings.Split(metric, ".")
	last_path := spl[len(spl)-1]

	// statsd like things are "mean_XX", "upper_XX", "lower_XX", "count_XX"
	switch {
	case last_path == "count" || strings.HasPrefix(metric, "stats.count"):
		return SUM
	case last_path == "last" || last_path == "gauge" || strings.HasPrefix(metric, "stats.gauge"):
		return LAST
	case last_path == "requests" || last_path == "sum" || last_path == "errors" || last_path == "error":
		return SUM
	case last_path == "max" || _upperMaxReg.MatchString(last_path) || _upperReg.MatchString(last_path):
		return MAX
	case last_path == "min" || _lowerMinReg.MatchString(last_path) || _lowerReg.MatchString(last_path):
		return MIN
	case last_path == "median" || strings.HasPrefix(metric, "stats.median") || _medianReg.MatchString(last_path):
		return MEDIAN
	default:
		return MEAN
	}
}

// for sorting
type AggFloat64 []float64

func (a AggFloat64) Len() int           { return len(a) }
func (a AggFloat64) Swap(i int, j int)  { a[i], a[j] = a[j], a[i] }
func (a AggFloat64) Less(i, j int) bool { return (a[i] - a[j]) < 0 } //this is the sorting statsd uses for its timings

type AGG_FUNC func(AggFloat64) float64

var ACCUMULATE_FUNC = map[AggType]AGG_FUNC{
	SUM: func(vals AggFloat64) float64 {
		val := 0.0
		for _, item := range vals {
			val += item
		}
		return val
	},
	MEAN: func(vals AggFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		val := 0.0
		for _, item := range vals {
			val += item
		}

		return val / float64(len(vals))
	},
	MEDIAN: func(vals AggFloat64) float64 {
		l_val := len(vals)
		if l_val == 0 {
			return 0
		}
		sort.Sort(vals)
		if l_val%2 == 0 {
			return vals[l_val-1] + vals[l_val+1]/2.0
		}

		return vals[l_val]
	},
	MAX: func(vals AggFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		sort.Sort(vals)

		return vals[len(vals)-1]
	},
	MIN: func(vals AggFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		sort.Sort(vals)

		return vals[0]
	},
	FIRST: func(vals AggFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		return vals[0]
	},
	LAST: func(vals AggFloat64) float64 {
		if len(vals) == 0 {
			return 0
		}
		return vals[len(vals)-1]
	},
	STD: func(vals AggFloat64) float64 {
		l := len(vals)
		if l == 0 {
			return 0
		}
		val := 0.0
		for _, item := range vals {
			val += item
		}

		mean := val / float64(l)
		std := float64(0)
		for _, item := range vals {
			std += math.Pow(item-mean, 2.0)
		}
		return math.Sqrt(std / float64(l))
	},
}
