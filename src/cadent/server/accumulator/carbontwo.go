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
	First float64
	Last  float64
	Count int64

	mu sync.RWMutex
}

func (s *CarbonTwoBaseStatItem) Repr() *repr.StatRepr {
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
	s.First = CARBONTWO_ACC_MIN_FLAG
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
		"c",
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
	if s.First == CARBONTWO_ACC_MIN_FLAG {
		s.First = val
	}
	return nil
}

/******************************/
/** statsd accumulator **/
/******************************/

type CarbonTwoAccumulate struct {
	CarbonTwoStats map[string]StatItem
	OutFormat      FormatterItem
	InTags         []AccumulatorTags
	InKeepKeys     bool
	Resolution     time.Duration

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

func (s *CarbonTwoAccumulate) GetResolution() time.Duration {
	return s.Resolution
}

func (s *CarbonTwoAccumulate) SetOptions(ops [][]string) error {
	return nil
}

func (s *CarbonTwoAccumulate) GetOption(opt string, defaults interface{}) interface{} {
	return defaults
}

func (s *CarbonTwoAccumulate) Tags() []AccumulatorTags {
	return s.InTags
}

func (s *CarbonTwoAccumulate) SetTags(tags []AccumulatorTags) {
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
func (a *CarbonTwoAccumulate) ProcessLine(line string) (err error) {
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
	// make it name=val.name=val
	sort.Sort(tags)
	unique_key := tags.ToStringSep("=", ".")
	stat_key := a.MapKey(unique_key, t)
	var meta_tags repr.SortingTags
	// now for the "other" tags
	if l_vals > 2 {
		meta_tags = repr.SortingTagsFromArray(vals[0:(l_vals - 2)])
	}

	// now the accumlator
	a.mu.RLock()
	gots, ok := a.CarbonTwoStats[stat_key]
	a.mu.RUnlock()

	if !ok {

		// based on the stat key (if present) figure out the agg
		gots = &CarbonTwoBaseStatItem{
			InType:     "carbontwo",
			Time:       a.ResolutionTime(t),
			InKey:      repr.StatName{Key: unique_key, Tags: tags, MetaTags: meta_tags},
			Min:        CARBONTWO_ACC_MIN_FLAG,
			Max:        CARBONTWO_ACC_MIN_FLAG,
			First:      CARBONTWO_ACC_MIN_FLAG,
			Last:       CARBONTWO_ACC_MIN_FLAG,
			ReduceFunc: repr.AggFuncFromTag(tags.FindTag("stat")),
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
