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
   Here we accumulate carbontwo metrics and then push to a output format of whatever
   basically an internal carbontwo accumulator server
*/

package accumulator

import (
	"cadent/server/repr"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

/****************** RUNNERS *********************/
const CARBONTWO_ACC_NAME = "carbontwo_accumlator"
const CARBONTWO_ACC_MIN_FLAG = math.MinInt64

var errCarbonTwoNotValid = errors.New("Invalid Carbon2.0 line")
var errCarbonTwoUnitRequired = errors.New("unit Tag is required")
var errCarbonTwoMTypeRequired = errors.New("mtype Tag is required")
var errorCarbonTwoBadTag = errors.New("Bad tag in carbon 2 format (need just name=val)")

type CarbonTwoBaseStatItem struct {
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

	mu sync.RWMutex
}

func (s *CarbonTwoBaseStatItem) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  s.Time.UnixNano(),
		Name:  &s.InKey,
		Min:   repr.CheckFloat(s.Min),
		Max:   repr.CheckFloat(s.Max),
		Count: s.Count,
		Sum:   repr.CheckFloat(s.Sum),
		Last:  repr.CheckFloat(s.Last),
	}
}

// merge this item w/ another stat repr
func (s *CarbonTwoBaseStatItem) Merge(stat *repr.StatRepr) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Min == CARBONTWO_ACC_MIN_FLAG || s.Min > float64(stat.Min) {
		s.Min = float64(stat.Min)
	}
	if s.Max == CARBONTWO_ACC_MIN_FLAG || s.Max < float64(stat.Max) {
		s.Max = float64(stat.Max)
	}

	s.Count += stat.Count
	s.Sum += float64(stat.Sum)
	if s.Time.Before(stat.ToTime()) || s.Last == CARBONTWO_ACC_MIN_FLAG {
		s.Last = float64(stat.Last)
	}
	return nil
}

func (s *CarbonTwoBaseStatItem) StatTime() time.Time { return s.Time }
func (s *CarbonTwoBaseStatItem) Type() string        { return s.InType }
func (s *CarbonTwoBaseStatItem) Key() repr.StatName  { return s.InKey }

func (s *CarbonTwoBaseStatItem) ZeroOut() error {
	// reset the values
	s.Time = time.Time{}
	s.Values = repr.AggFloat64{}
	s.Min = CARBONTWO_ACC_MIN_FLAG
	s.Max = CARBONTWO_ACC_MIN_FLAG
	s.Sum = 0.0
	s.Count = 0
	s.Last = CARBONTWO_ACC_MIN_FLAG
	return nil
}

func (s *CarbonTwoBaseStatItem) Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem) {

	val := s.ReduceFunc(s.Values)

	fmatter.Write(
		buffer,
		&s.InKey,
		val,
		int32(s.StatTime().Unix()),
		s.InKey.Tags.Mtype(),
		acc.Tags(),
	)

}

func (s *CarbonTwoBaseStatItem) Accumulate(val float64, sample float64, stattime time.Time) error {
	if math.IsInf(val, 0) || math.IsNaN(val) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Values = append(s.Values, val)
	if s.Min == CARBONTWO_ACC_MIN_FLAG || s.Min > val {
		s.Min = val
	}
	if s.Max == CARBONTWO_ACC_MIN_FLAG || s.Max < val {
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

type CarbonTwoAccumulate struct {
	CarbonTwoStats map[string]StatItem
	OutFormat      FormatterItem
	InTags         repr.SortingTags
	InKeepKeys     bool
	Resolution     time.Duration
	TagMode        repr.TagMode

	mu sync.RWMutex
}

func NewCarbonTwoAccumulate() (*CarbonTwoAccumulate, error) {
	return new(CarbonTwoAccumulate), nil
}

// based on the resolution we need to aggregate around a
// "key+time bucket" mix.  to figure out the time bucket
// we simply use the resolution -- time % resolution
func (s *CarbonTwoAccumulate) ResolutionTime(t time.Time) time.Time {
	return t.Truncate(s.Resolution)
}

func (s *CarbonTwoAccumulate) MapKey(name string, t time.Time) string {
	return fmt.Sprintf("%s-%d", name, s.ResolutionTime(t).UnixNano())
}

func (s *CarbonTwoAccumulate) SetResolution(dur time.Duration) error {
	s.Resolution = dur
	return nil
}
func (s *CarbonTwoAccumulate) SetTagMode(mode repr.TagMode) error {
	s.TagMode = mode
	return nil
}

func (s *CarbonTwoAccumulate) GetResolution() time.Duration {
	return s.Resolution
}

func (s *CarbonTwoAccumulate) SetOptions(ops [][]string) error {
	return nil
}

func (s *CarbonTwoAccumulate) GetOption(opt string, defaults interface{}) interface{} {
	return defaults
}

func (s *CarbonTwoAccumulate) Tags() repr.SortingTags {
	return s.InTags
}

func (s *CarbonTwoAccumulate) SetTags(tags repr.SortingTags) {
	s.InTags = tags
}

func (s *CarbonTwoAccumulate) SetKeepKeys(k bool) error {
	s.InKeepKeys = k
	return nil
}

func (s *CarbonTwoAccumulate) Init(fmatter FormatterItem) error {
	s.OutFormat = fmatter
	fmatter.SetAccumulator(s)
	s.CarbonTwoStats = make(map[string]StatItem)
	s.SetOptions([][]string{})
	return nil
}

func (s *CarbonTwoAccumulate) Stats() map[string]StatItem {
	return s.CarbonTwoStats
}

func (a *CarbonTwoAccumulate) Name() (name string) { return CARBONTWO_ACC_NAME }

func (a *CarbonTwoAccumulate) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// keep or reset
	if a.InKeepKeys {
		for idx := range a.CarbonTwoStats {
			a.CarbonTwoStats[idx].ZeroOut()
		}
	} else {
		a.CarbonTwoStats = nil
		a.CarbonTwoStats = make(map[string]StatItem)
	}

	return nil
}

func (a *CarbonTwoAccumulate) Flush(buf io.Writer) *flushedList {
	fl := new(flushedList)

	a.mu.RLock()
	for _, stats := range a.CarbonTwoStats {
		stats.Write(buf, a.OutFormat, a)
		fl.AddStat(stats.Repr())
	}
	a.mu.RUnlock()
	a.Reset()
	return fl
}

func (a *CarbonTwoAccumulate) FlushList() *flushedList {
	fl := new(flushedList)

	a.mu.RLock()
	for _, stats := range a.CarbonTwoStats {
		fl.AddStat(stats.Repr())
	}
	a.mu.RUnlock()
	a.Reset()
	return fl
}

// process a "repr" metric as if it was injected as a Graphite like item
// basically this is a post parsed string line
func (a *CarbonTwoAccumulate) ProcessRepr(stat *repr.StatRepr) error {

	tgs := stat.Name.SortedTags()

	// obey tag mode
	var meta repr.SortingTags
	switch a.TagMode {
	case repr.TAG_ALLTAGS:
		tgs = tgs.Merge(stat.Name.MetaTags)
		stat.Name.MetaTags = repr.SortingTags{}
	default:
		tgs, meta = repr.SplitIntoMetric2Tags(tgs, stat.Name.MetaTags)
		stat.Name.Tags = tgs
		stat.Name.MetaTags = meta
	}

	sort.Sort(tgs)
	unique_key := tgs.ToStringSep(repr.IS_SEPARATOR, repr.DOT_SEPARATOR)
	stat_key := a.MapKey(unique_key, stat.ToTime())

	// now the accumlator
	a.mu.RLock()
	gots, ok := a.CarbonTwoStats[stat_key]
	a.mu.RUnlock()

	if !ok {

		// based on the stat key (if present) figure out the agg
		gots = &CarbonTwoBaseStatItem{
			InType:     "carbontwo",
			Time:       a.ResolutionTime(stat.ToTime()),
			InKey:      *stat.Name,
			Min:        CARBONTWO_ACC_MIN_FLAG,
			Max:        CARBONTWO_ACC_MIN_FLAG,
			Last:       CARBONTWO_ACC_MIN_FLAG,
			ReduceFunc: repr.AggFuncFromTag(tgs.Stat()),
		}
	}

	// now for some trickery.  If the count is 1 then we assume not
	// a "pre-accumulated" repr, but basically a metric/value and need to properly accumulate
	if stat.Count == 1 {
		gots.Accumulate(float64(stat.Sum), 1.0, stat.ToTime())
	} else {
		gots.(*CarbonTwoBaseStatItem).Merge(stat)
	}

	// log.Critical("key: %s Dr: %s, InTime: %s (%s), ResTime: %s", stat_key, a.Resolution.String(), t.String(), _intime, a.ResolutionTime(t).String())

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.CarbonTwoStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}

/*
<tag> <tag> <tag>  <metatags> <metatags> <metatags> <value> <time>
 the <tags> for the unique "key" for the metric
 note there are TWO SPACES between <tag> and <metatag>
 If there is not a '2 space" split, then we assume no metatags
 in fact we eventually "store" this as <tag>.<tag>.... strings for "paths"
 so the ordering here is important
 there must be a <unit> and <mtype> tag in the tags list

	mtype values
	rate	a number per second (implies that unit ends on ‘/s’)
	count	a number per a given interval (such as a statsd flushInterval)
	gauge	values at each point in time
	counter	keeps increasing over time (but might wrap/reset at some point) i.e. a gauge with the added notion of “i usually want to derive this to see the rate”
	timestamp
*/
func (a *CarbonTwoAccumulate) ProcessLine(lineb []byte) (err error) {
	line := string(lineb)

	stats_arr := strings.Split(line, "  ")
	var key string
	var vals []string

	if len(stats_arr) == 1 {
		key = stats_arr[0]
		t_vs := strings.Fields(line)
		vals = t_vs[1:]
	} else {
		key = stats_arr[0]
		vals = strings.Fields(stats_arr[1])
	}

	if len(vals) < 2 {
		return fmt.Errorf("Accumulate: Invalid CarbonTwo line `%s`", line)
	}

	l_vals := len(vals)
	_intime := vals[l_vals-1] // should be unix timestamp
	_val := vals[l_vals-2]

	// if time messes up just use "now"
	t := time.Now()
	i, err := strconv.ParseInt(_intime, 10, 64)
	if err == nil {
		// nano or second tstamps
		if i > 2147483647 {
			t = time.Unix(0, i)
		} else {
			t = time.Unix(i, 0)
		}
	}

	f_val, err := strconv.ParseFloat(_val, 64)
	if err != nil {
		return fmt.Errorf("Accumulate: Bad Value | Invalid CarbonTwo line `%s`", line)
	}

	//need to get the sorted key tag values for things
	tags := repr.SortingTagsFromString(key)
	if tags.Unit() == "" {
		return errCarbonTwoUnitRequired
	}
	if tags.Mtype() == "" {
		return errCarbonTwoMTypeRequired
	}

	// the sorted tags give us a string of goodies
	// make it name.val.name.val
	var meta_tags repr.SortingTags
	// now for the "other" tags
	if l_vals > 2 {
		meta_tags = repr.SortingTagsFromArray(vals[0:(l_vals - 2)])
	}
	//tag mode

	switch a.TagMode {
	case repr.TAG_ALLTAGS:
		tags = tags.Merge(meta_tags)
		meta_tags = repr.SortingTags{} // null it out
	default:
		break
	}

	sort.Sort(tags)
	unique_key := tags.ToStringSep(repr.IS_SEPARATOR, repr.DOT_SEPARATOR)
	stat_key := a.MapKey(unique_key, t)

	// now the accumlator
	a.mu.RLock()
	gots, ok := a.CarbonTwoStats[stat_key]
	a.mu.RUnlock()

	if !ok {

		// based on the stat key (if present) figure out the agg
		gots = &CarbonTwoBaseStatItem{
			InType:     "carbontwo",
			Time:       a.ResolutionTime(t),
			InKey:      repr.StatName{Key: unique_key, Tags: tags, MetaTags: meta_tags, TagMode: a.TagMode},
			Min:        CARBONTWO_ACC_MIN_FLAG,
			Max:        CARBONTWO_ACC_MIN_FLAG,
			Last:       CARBONTWO_ACC_MIN_FLAG,
			ReduceFunc: repr.AggFuncFromTag(tags.Stat()),
		}
	}

	// needs to lock internally if needed
	gots.Accumulate(f_val, 1.0, t)
	// log.Critical("key: %s Dr: %s, InTime: %s (%s), ResTime: %s", stat_key, a.Resolution.String(), t.String(), _intime, a.ResolutionTime(t).String())

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.CarbonTwoStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}
