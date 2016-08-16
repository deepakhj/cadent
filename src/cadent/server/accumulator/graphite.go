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
   Here we accumulate graphite metrics and then push to a output format of whatever
   basically an internal graphite accumulator server
*/

package accumulator

import (
	"cadent/server/repr"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

/****************** RUNNERS *********************/
const GRAPHITE_ACC_NAME = "graphite_accumlator"
const GRAHPITE_ACC_MIN_LEN = 3
const GRAPHITE_ACC_MIN_FLAG = math.MinInt64

/** counter/gauge type **/

type GraphiteBaseStatItem struct {
	InKey      repr.StatName
	Values     repr.AggFloat64
	InType     string
	ReduceFunc repr.AGG_FUNC
	Time       time.Time
	Resolution time.Duration

	Min   float64
	Max   float64
	Sum   float64
	First float64
	Last  float64
	Count int64

	mu sync.Mutex
}

func (s *GraphiteBaseStatItem) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  s.Time,
		Name:  s.InKey,
		Min:   repr.CheckFloat(repr.JsonFloat64(s.Min)),
		Max:   repr.CheckFloat(repr.JsonFloat64(s.Max)),
		Count: s.Count,
		Sum:   repr.CheckFloat(repr.JsonFloat64(s.Sum)),
		First: repr.CheckFloat(repr.JsonFloat64(s.First)),
		Last:  repr.CheckFloat(repr.JsonFloat64(s.Last)),
	}
}
func (s *GraphiteBaseStatItem) StatTime() time.Time { return s.Time }
func (s *GraphiteBaseStatItem) Type() string        { return s.InType }
func (s *GraphiteBaseStatItem) Key() repr.StatName  { return s.InKey }

func (s *GraphiteBaseStatItem) ZeroOut() error {
	// reset the values
	s.Time = time.Time{}
	s.Values = repr.AggFloat64{}
	s.Min = GRAPHITE_ACC_MIN_FLAG
	s.Max = GRAPHITE_ACC_MIN_FLAG
	s.Sum = 0.0
	s.Count = 0
	s.First = GRAPHITE_ACC_MIN_FLAG
	s.Last = GRAPHITE_ACC_MIN_FLAG
	return nil
}

func (s *GraphiteBaseStatItem) Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val := s.ReduceFunc(s.Values)
	fmatter.Write(
		buffer,
		&s.InKey,
		val,
		int32(s.StatTime().Unix()),
		"c",
		acc.Tags(),
	)

}

func (s *GraphiteBaseStatItem) Accumulate(val float64, sample float64, stattime time.Time) error {
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
	InTags        repr.SortingTags
	InKeepKeys    bool
	Resolution    time.Duration

	mu sync.Mutex
}

func NewGraphiteAccumulate() (*GraphiteAccumulate, error) {
	return new(GraphiteAccumulate), nil
}

// based on the resolution we need to aggregate around a
// "key+time bucket" mix.  to figure out the time bucket
// we simply use the resolution -- time % resolution
func (s *GraphiteAccumulate) ResolutionTime(t time.Time) time.Time {
	return t.Truncate(s.Resolution)
}

func (s *GraphiteAccumulate) MapKey(name string, t time.Time) string {
	return fmt.Sprintf("%s-%d", name, s.ResolutionTime(t).UnixNano())
}

func (s *GraphiteAccumulate) SetResolution(dur time.Duration) error {
	s.Resolution = dur
	return nil
}

func (s *GraphiteAccumulate) GetResolution() time.Duration {
	return s.Resolution
}

func (s *GraphiteAccumulate) SetOptions(ops [][]string) error {
	return nil
}

func (s *GraphiteAccumulate) GetOption(opt string, defaults interface{}) interface{} {
	return defaults
}

func (s *GraphiteAccumulate) Tags() repr.SortingTags {
	return s.InTags
}

func (s *GraphiteAccumulate) SetTags(tags repr.SortingTags) {
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

func (a *GraphiteAccumulate) Flush(buf io.Writer) *flushedList {
	fl := new(flushedList)

	a.mu.Lock()
	for _, stats := range a.GraphiteStats {
		stats.Write(buf, a.OutFormat, a)
		fl.AddStat(stats.Repr())
	}
	a.mu.Unlock()
	a.Reset()
	return fl
}

/*
	<key> <value> <time> <tag> <tag> ...
	<tag> are not required, but will get saved into the MetaTags of the internal rep
	as it's assumed that the "key" is the unique item we want to lookup
	tags are of the form name=val
*/

func (a *GraphiteAccumulate) ProcessLine(line string) (err error) {

	stats_arr := strings.Fields(line)
	l := len(stats_arr)

	if l < GRAHPITE_ACC_MIN_LEN {
		return fmt.Errorf("Accumulate: Invalid Graphite line `%s`", line)
	}

	//
	key := stats_arr[0]
	val := stats_arr[1]
	_intime := stats_arr[2] // should be unix timestamp
	t := time.Now()
	i, err := strconv.ParseInt(_intime, 10, 64)
	if err == nil {
		t = time.Unix(i, 0)
	}

	f_val, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fmt.Errorf("Accumulate: Bad Value | Invalid Graphite line `%s`", line)
	}

	var tags repr.SortingTags
	if l > GRAHPITE_ACC_MIN_LEN {
		// gots some potential tags
		tags = repr.SortingTagsFromArray(stats_arr[3:])
	}

	stat_key := a.MapKey(key, t)
	// now the accumlator
	a.mu.Lock()

	gots, ok := a.GraphiteStats[stat_key]
	a.mu.Unlock()

	if !ok {
		gots = &GraphiteBaseStatItem{
			InType:     "graphite",
			Time:       a.ResolutionTime(t),
			InKey:      repr.StatName{Key: key, MetaTags: tags},
			Min:        GRAPHITE_ACC_MIN_FLAG,
			Max:        GRAPHITE_ACC_MIN_FLAG,
			First:      GRAPHITE_ACC_MIN_FLAG,
			Last:       GRAPHITE_ACC_MIN_FLAG,
			ReduceFunc: repr.GuessAggFuncFromKey(key),
		}
	}

	// needs to lock internally if needed
	gots.Accumulate(f_val, 1.0, t)
	// log.Critical("key: %s Dr: %s, InTime: %s (%s), ResTime: %s", stat_key, a.Resolution.String(), t.String(), _intime, a.ResolutionTime(t).String())

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.GraphiteStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}
