/*
Copyright 2016 Under Armour, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
var _countReg *regexp.Regexp
var _stdReg *regexp.Regexp
var aggMap map[string]AggType

func init() {
	_upperReg = regexp.MustCompile(".*upper_[0-9]+$")
	_upperMaxReg = regexp.MustCompile(".*max_[0-9]+$")
	_lowerReg = regexp.MustCompile(".*lower_[0-9]+$")
	_lowerMinReg = regexp.MustCompile(".*min_[0-9]+$")
	_medianReg = regexp.MustCompile(".*median_[0-9]+$")
	_countReg = regexp.MustCompile(".*count_[0-9]+$")
	_stdReg = regexp.MustCompile(".*std_[0-9]+$")

	aggMap = make(map[string]AggType, 0)

	aggMap["min"] = MIN
	aggMap["lower"] = MIN

	aggMap["max"] = MAX
	aggMap["upper"] = MAX

	aggMap["hit"] = SUM
	aggMap["hits"] = SUM
	aggMap["sum"] = SUM
	aggMap["count"] = SUM
	aggMap["counts"] = SUM
	aggMap["request"] = SUM
	aggMap["requests"] = SUM
	aggMap["ok"] = SUM
	aggMap["error"] = SUM
	aggMap["errors"] = SUM
	aggMap["select"] = SUM
	aggMap["selects"] = SUM
	aggMap["selected"] = SUM
	aggMap["insert"] = SUM
	aggMap["inserts"] = SUM
	aggMap["inserted"] = SUM
	aggMap["update"] = SUM
	aggMap["updates"] = SUM
	aggMap["updated"] = SUM
	aggMap["delete"] = SUM
	aggMap["deletes"] = SUM
	aggMap["deleted"] = SUM
	aggMap["consume"] = SUM
	aggMap["consumed"] = SUM
	aggMap["set"] = SUM
	aggMap["sets"] = SUM

	aggMap["gauge"] = LAST
	aggMap["last"] = LAST
	aggMap["abs"] = LAST
	aggMap["absolute"] = LAST

	aggMap["median"] = MEDIAN
	aggMap["middle"] = MEDIAN

	aggMap["std"] = STD

}

type AggType uint8

const (
	MEAN AggType = iota + 1
	SUM
	FIRST
	LAST
	MIN
	MAX
	STD
	MEDIAN
	COUNT
)

// if there is a tag that has the agg func in it
func AggTypeFromTag(stat string) AggType {
	stat = strings.ToLower(stat)

	got, ok := aggMap[stat]
	if ok {
		return got
	}

	switch {
	case _lowerMinReg.MatchString(stat) || _lowerReg.MatchString(stat):
		return MIN
	case _upperReg.MatchString(stat) || _upperMaxReg.MatchString(stat):
		return MAX
	case _countReg.MatchString(stat):
		return SUM
	case _stdReg.MatchString(stat):
		return STD
	case _medianReg.MatchString(stat):
		return MEDIAN
	default:
		return MEAN
	}
}

func AggFuncFromTag(stat string) AGG_FUNC {
	return ACCUMULATE_FUNC[AggTypeFromTag(stat)]
}

// guess the agg func from the my.metric.is.good string
// (there is certainly a better way to do this)

func GuessReprValueFromKey(metric string) AggType {
	spl := strings.Split(metric, ".")
	last_path := strings.ToLower(spl[len(spl)-1])

	// statsd like things are "mean_XX", "upper_XX", "lower_XX", "count_XX"
	got, ok := aggMap[last_path]
	if ok {
		return got
	}

	switch {
	case strings.HasPrefix(metric, "stats_count") || strings.HasPrefix(metric, "stats.count") || strings.HasPrefix(metric, "stats.set") || strings.HasPrefix(metric, "stats.sets") || _countReg.MatchString(metric):
		return SUM
	case strings.HasPrefix(metric, "stats.gauge"):
		return LAST
	case _upperMaxReg.MatchString(last_path) || _upperReg.MatchString(last_path):
		return MAX
	case _lowerMinReg.MatchString(last_path) || _lowerReg.MatchString(last_path):
		return MIN
	case strings.HasPrefix(metric, "stats.median") || _medianReg.MatchString(last_path):
		return MEDIAN
	default:
		return MEAN
	}
}

func GuessAggFuncFromKey(stat string) AGG_FUNC {
	return ACCUMULATE_FUNC[GuessReprValueFromKey(stat)]
}

func GuessAggFuncFromName(nm *StatName) AGG_FUNC {
	tg := nm.Tags.Stat()
	if len(tg) > 0 {
		return AggFuncFromTag(tg)
	}
	return GuessAggFuncFromKey(nm.Key)
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
		use_v := l_val / 2
		if l_val%2 == 0 && l_val > 3 {
			return (vals[use_v-1] + vals[use_v+1]) / 2.0
		}

		return vals[use_v]
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
	COUNT: func(vals AggFloat64) float64 {
		return float64(len(vals))
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
