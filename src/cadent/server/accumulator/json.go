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
   Incoming Json metric

   {
   	metric: xxx
   	value: xxx
   	timestamp: xxx
   	tags: {
   		name: val,
   		name: val
   	}

   }

*/

package accumulator

import (
	"cadent/server/repr"
	"cadent/server/splitter"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"sync"
	"time"
)

/****************** RUNNERS *********************/
const JSON_ACC_NAME = "json_accumlator"
const JSON_ACC_MIN_FLAG = math.MinInt64

/** counter/gauge type **/

type JsonBaseStatItem struct {
	InKey      repr.StatName
	Values     repr.AggFloat64
	InType     string
	ReduceFunc repr.AGG_FUNC
	Time       time.Time
	Resolution time.Duration

	Min   float64
	Max   float64
	Sum   float64
	Last  float64
	Count int64

	mu sync.Mutex
}

func (s *JsonBaseStatItem) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  s.Time,
		Name:  s.InKey,
		Min:   repr.CheckFloat(repr.JsonFloat64(s.Min)),
		Max:   repr.CheckFloat(repr.JsonFloat64(s.Max)),
		Count: s.Count,
		Sum:   repr.CheckFloat(repr.JsonFloat64(s.Sum)),
		Last:  repr.CheckFloat(repr.JsonFloat64(s.Last)),
	}
}
func (s *JsonBaseStatItem) StatTime() time.Time { return s.Time }
func (s *JsonBaseStatItem) Type() string        { return s.InType }
func (s *JsonBaseStatItem) Key() repr.StatName  { return s.InKey }

func (s *JsonBaseStatItem) ZeroOut() error {
	// reset the values
	s.Time = time.Time{}
	s.Values = repr.AggFloat64{}
	s.Min = JSON_ACC_MIN_FLAG
	s.Max = JSON_ACC_MIN_FLAG
	s.Sum = 0.0
	s.Count = 0
	s.Last = JSON_ACC_MIN_FLAG
	return nil
}

func (s *JsonBaseStatItem) Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem) {
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

// merge this item w/ another stat repr
func (s *JsonBaseStatItem) Merge(stat *repr.StatRepr) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Min == JSON_ACC_MIN_FLAG || s.Min > float64(stat.Min) {
		s.Min = float64(stat.Min)
	}
	if s.Max == JSON_ACC_MIN_FLAG || s.Max < float64(stat.Max) {
		s.Max = float64(stat.Max)
	}

	s.Count += stat.Count
	s.Sum += float64(stat.Sum)
	if s.Time.Before(stat.Time) || s.Last == JSON_ACC_MIN_FLAG {
		s.Last = float64(stat.Last)
	}
	return nil
}

func (s *JsonBaseStatItem) Accumulate(val float64, sample float64, stattime time.Time) error {
	if math.IsInf(val, 0) || math.IsNaN(val) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Values = append(s.Values, val)
	if s.Min == JSON_ACC_MIN_FLAG || s.Min > val {
		s.Min = val
	}
	if s.Max == JSON_ACC_MIN_FLAG || s.Max < val {
		s.Max = val
	}

	s.Count += 1
	s.Sum += val
	s.Last = val
	return nil
}

/******************************/
/** statsd accumulator **/
/******************************/

type JsonAccumulate struct {
	JsonStats  map[string]StatItem
	OutFormat  FormatterItem
	InTags     repr.SortingTags
	InKeepKeys bool
	Resolution time.Duration
	TagMode    uint8

	mu sync.RWMutex
}

func NewJsonAccumulate() (*JsonAccumulate, error) {
	return new(JsonAccumulate), nil
}

// based on the resolution we need to aggregate around a
// "key+time bucket" mix.  to figure out the time bucket
// we simply use the resolution -- time % resolution
func (s *JsonAccumulate) ResolutionTime(t time.Time) time.Time {
	return t.Truncate(s.Resolution)
}
func (s *JsonAccumulate) SetTagMode(mode uint8) error {
	s.TagMode = mode
	return nil
}
func (s *JsonAccumulate) MapKey(name string, t time.Time) string {
	return fmt.Sprintf("%s-%d", name, s.ResolutionTime(t).UnixNano())
}

func (s *JsonAccumulate) SetResolution(dur time.Duration) error {
	s.Resolution = dur
	return nil
}

func (s *JsonAccumulate) GetResolution() time.Duration {
	return s.Resolution
}

func (s *JsonAccumulate) SetOptions(ops [][]string) error {
	return nil
}

func (s *JsonAccumulate) GetOption(opt string, defaults interface{}) interface{} {
	return defaults
}

func (s *JsonAccumulate) Tags() repr.SortingTags {
	return s.InTags
}

func (s *JsonAccumulate) SetTags(tags repr.SortingTags) {
	s.InTags = tags
}

func (s *JsonAccumulate) SetKeepKeys(k bool) error {
	s.InKeepKeys = k
	return nil
}

func (s *JsonAccumulate) Init(fmatter FormatterItem) error {
	s.OutFormat = fmatter
	fmatter.SetAccumulator(s)
	s.JsonStats = make(map[string]StatItem)
	s.SetOptions([][]string{})
	return nil
}

func (s *JsonAccumulate) Stats() map[string]StatItem {
	return s.JsonStats
}

func (a *JsonAccumulate) Name() (name string) { return JSON_ACC_NAME }

func (a *JsonAccumulate) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// keep or reset
	if a.InKeepKeys {
		for idx := range a.JsonStats {
			a.JsonStats[idx].ZeroOut()
		}
	} else {
		a.JsonStats = nil
		a.JsonStats = make(map[string]StatItem)
	}

	return nil
}

func (a *JsonAccumulate) Flush(buf io.Writer) *flushedList {
	fl := new(flushedList)

	a.mu.RLock()
	for _, stats := range a.JsonStats {
		stats.Write(buf, a.OutFormat, a)
		fl.AddStat(stats.Repr())
	}
	a.mu.RUnlock()
	a.Reset()
	return fl
}

func (a *JsonAccumulate) FlushList() *flushedList {
	fl := new(flushedList)

	a.mu.RLock()
	for _, stats := range a.JsonStats {
		fl.AddStat(stats.Repr())
	}
	a.mu.RUnlock()
	a.Reset()
	return fl
}

// process a "repr" metric as if it was injected as a Graphite like item
// basically this is a post parsed string line
func (a *JsonAccumulate) ProcessRepr(stat *repr.StatRepr) error {

	tgs := stat.Name.SortedTags()

	// obey tag mode
	var meta repr.SortingTags
	switch a.TagMode {
	case repr.TAG_ALLTAGS:
		tgs = tgs.Merge(stat.Name.MetaTags)
	default:
		tgs, meta = repr.SplitIntoMetric2Tags(tgs, stat.Name.MetaTags)
		stat.Name.Tags = tgs
		stat.Name.MetaTags = meta
	}
	sort.Sort(tgs)
	stat_key := a.MapKey(stat.Name.Key+tgs.ToStringSep(repr.DOT_SEPARATOR, repr.DOT_SEPARATOR), stat.Time)
	// now the accumlator
	a.mu.RLock()
	gots, ok := a.JsonStats[stat_key]
	a.mu.RUnlock()

	if !ok {

		stat.Name.TagMode = a.TagMode

		gots = &JsonBaseStatItem{
			InType:     "graphite",
			Time:       a.ResolutionTime(stat.Time),
			InKey:      stat.Name,
			Count:      0,
			Min:        float64(stat.Min),
			Max:        float64(stat.Max),
			Last:       float64(stat.Last),
			ReduceFunc: repr.GuessAggFuncFromName(&stat.Name),
		}
	}

	gots.(*JsonBaseStatItem).Merge(stat)

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.JsonStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}

func (a *JsonAccumulate) ProcessLine(linebytes []byte) (err error) {

	spl := new(splitter.JsonStructSplitItem)

	err = json.Unmarshal(linebytes, spl)
	if err != nil {
		return err
	}

	t := time.Now()
	if spl.Time > 0 {
		if spl.Time > 2147483647 {
			t = time.Unix(0, spl.Time)
		} else {
			t = time.Unix(spl.Time, 0)
		}
	}

	//tag mode
	var tgs, metatags repr.SortingTags
	switch a.TagMode {
	case repr.TAG_ALLTAGS:
		break
	default:
		tgs, metatags = repr.Metric2FromMap(spl.Tags)

	}

	stat_key := a.MapKey(spl.Metric+tgs.ToStringSep(repr.DOT_SEPARATOR, repr.DOT_SEPARATOR), t)
	// now the accumlator
	a.mu.RLock()
	gots, ok := a.JsonStats[stat_key]
	a.mu.RUnlock()

	if !ok {
		nm := repr.StatName{Key: spl.Metric, MetaTags: metatags, Tags: tgs, TagMode: a.TagMode}
		gots = &JsonBaseStatItem{
			InType:     "json",
			Time:       a.ResolutionTime(t),
			InKey:      nm,
			Min:        JSON_ACC_MIN_FLAG,
			Max:        JSON_ACC_MIN_FLAG,
			Last:       JSON_ACC_MIN_FLAG,
			ReduceFunc: repr.GuessAggFuncFromName(&nm),
		}
	}

	// needs to lock internally if needed
	gots.Accumulate(spl.Value, 1.0, t)
	// log.Critical("key: %s Dr: %s, InTime: %s (%s), ResTime: %s", stat_key, a.Resolution.String(), t.String(), _intime, a.ResolutionTime(t).String())

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.JsonStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}
