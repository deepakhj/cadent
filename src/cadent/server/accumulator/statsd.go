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
   Here we accumulate statsd metrics and then push to a output format of whatever
   basically an internal statsd server
*/

package accumulator

import (
	"cadent/server/repr"
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
const STATD_ACC_NAME = "statsd_accumlator"
const STATD_ACC_MIN_LEN = 2
const STATSD_ACC_MIN_FLAG = math.MinInt64

/** counter/gauge type **/
// for sorting
type statdFloat64arr []float64

func (a statdFloat64arr) Len() int           { return len(a) }
func (a statdFloat64arr) Swap(i int, j int)  { a[i], a[j] = a[j], a[i] }
func (a statdFloat64arr) Less(i, j int) bool { return (a[i] - a[j]) < 0 } //this is the sorting statsd uses for its timings

func round(a float64) float64 {
	if a < 0 {
		return math.Ceil(a - 0.5)
	}
	return math.Floor(a + 0.5)
}

type StatsdBaseStatItem struct {
	InKey      repr.StatName
	Count      int64
	Min        float64
	Max        float64
	Sum        float64
	First      float64
	Last       float64
	InType     string
	start_time int64

	mu sync.Mutex
}

func (s *StatsdBaseStatItem) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Name:  s.InKey,
		Min:   repr.CheckFloat(repr.JsonFloat64(s.Min)),
		Max:   repr.CheckFloat(repr.JsonFloat64(s.Max)),
		Count: s.Count,
		Sum:   repr.CheckFloat(repr.JsonFloat64(s.Sum)),
		First: repr.CheckFloat(repr.JsonFloat64(s.First)),
		Last:  repr.CheckFloat(repr.JsonFloat64(s.Last)),
	}
}

// statsd has no time in the format other then "now"
func (s *StatsdBaseStatItem) StatTime() time.Time { return time.Now() }
func (s *StatsdBaseStatItem) Type() string        { return s.InType }
func (s *StatsdBaseStatItem) Key() repr.StatName  { return s.InKey }

func (s *StatsdBaseStatItem) ZeroOut() error {
	// reset the values
	s.Min = STATSD_ACC_MIN_FLAG
	s.Max = STATSD_ACC_MIN_FLAG
	s.Sum = 0.0
	s.Count = 0
	s.start_time = 0
	s.First = STATSD_ACC_MIN_FLAG
	s.Last = STATSD_ACC_MIN_FLAG
	return nil
}

func (s *StatsdBaseStatItem) Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	root := acc.GetOption("Prefix", "stats").(string)
	pref := acc.GetOption("CounterPrefix", "counters").(string) //default is none for legacy namepace
	sufix := acc.GetOption("Suffix", "").(string)
	f_key := ""
	if len(root) > 0 {
		f_key = root + "."
	}
	if len(pref) > 0 {
		f_key = f_key + pref + "."
	}
	if len(sufix) > 0 {
		f_key = f_key + sufix + "."
	}

	c_type := "c"
	val := s.Sum

	tick := time.Now().Unix() - s.start_time
	if tick == 0 {
		tick = 1.0
	}
	if s.InType == "g" || s.InType == "-g" || s.InType == "+g" {
		pref = acc.GetOption("GaugePrefix", "gauges").(string)
		f_key = root + "." + pref + "."
		if len(sufix) > 0 {
			f_key = f_key + sufix + "."
		}
		c_type = "g"
		val = s.Last // gauges get the "last" value
	}

	// reset the ticker
	s.start_time = time.Now().Unix()
	in_key := s.InKey.Key
	if c_type == "c" {
		val_p_s := val / float64(tick)
		if acc.GetOption("LegacyStatsd", true).(bool) {
			rate_pref := "stats_counts."
			if len(sufix) > 0 {
				rate_pref = rate_pref + sufix + "."
			}
			fmatter.Write(
				buffer,
				&repr.StatName{Key: f_key + in_key},
				val_p_s,
				0, // let formatter handle the time,
				c_type,
				acc.Tags(),
			)
			fmatter.Write(buffer,

				&repr.StatName{Key: rate_pref + in_key},
				val,
				0, // let formatter handle the time,
				c_type,
				acc.Tags(),
			)
			return
		} else {

			fmatter.Write(
				buffer,
				&repr.StatName{Key: f_key + "count." + in_key},
				val,
				0, // let formatter handle the time,
				c_type,
				acc.Tags(),
			)
			fmatter.Write(
				buffer,
				&repr.StatName{Key: f_key + "rate." + in_key},
				val_p_s,
				0, // let formatter handle the time,
				c_type,
				acc.Tags(),
			)
			return
		}
	}
	fmatter.Write(
		buffer,
		&repr.StatName{Key: f_key + in_key},
		val,
		0, // let formatter handle the time,
		c_type,
		acc.Tags(),
	)

}

func (s *StatsdBaseStatItem) Accumulate(val float64, sample float64, stattime time.Time) error {
	if math.IsInf(val, 0) || math.IsNaN(val) {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.start_time == 0 {
		s.start_time = time.Now().Unix()
	}

	switch {
	case s.InType == "c": //counter
		s.Sum += val * sample
	case s.InType == "g": //gauge
		s.Sum = val
	case s.InType == "-g": //gauge negate
		s.Sum -= val
	case s.InType == "+g": //gauge add
		s.Sum += val
	}
	if s.Min == STATSD_ACC_MIN_FLAG || s.Min > val {
		s.Min = val
	}
	if s.Max == STATSD_ACC_MIN_FLAG || s.Max < val {
		s.Max = val
	}
	if s.First == STATSD_ACC_MIN_FLAG {
		s.First = val
	}
	s.Last = val
	s.Count += 1
	return nil
}

/** timer type **/

type StatsdTimerStatItem struct {
	InKey     repr.StatName
	InType    string
	Count     int64
	Min       float64
	Max       float64
	Sum       float64
	First     float64
	Last      float64
	SampleSum float64 // sample rate sum
	Values    statdFloat64arr

	PercentThreshold []float64

	start_time int64

	mu sync.Mutex
}

func (s *StatsdTimerStatItem) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  time.Now(),
		Name:  s.InKey,
		Min:   repr.CheckFloat(repr.JsonFloat64(s.Min)),
		Max:   repr.CheckFloat(repr.JsonFloat64(s.Max)),
		Count: s.Count,
		Sum:   repr.CheckFloat(repr.JsonFloat64(s.Sum)),
		First: repr.CheckFloat(repr.JsonFloat64(s.First)),
		Last:  repr.CheckFloat(repr.JsonFloat64(s.Last)),
	}
}

// time is 'now' basically for statsd
func (s *StatsdTimerStatItem) StatTime() time.Time { return time.Now() }
func (s *StatsdTimerStatItem) Key() repr.StatName  { return s.InKey }
func (s *StatsdTimerStatItem) Type() string        { return s.InType }

// stattime is not used for statsd, but there for interface goodness
func (s *StatsdTimerStatItem) Accumulate(val float64, sample float64, stattime time.Time) error {
	if math.IsInf(val, 0) || math.IsNaN(val) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.start_time == 0 {
		s.start_time = time.Now().Unix()
	}
	s.Count += 1
	s.Sum += val
	if s.Min == STATSD_ACC_MIN_FLAG || s.Min > val {
		s.Min = val
	}
	if s.Max == STATSD_ACC_MIN_FLAG || s.Max < val {
		s.Max = val
	}

	s.Last = val
	if s.First == STATSD_ACC_MIN_FLAG {
		s.First = val
	}

	//log.Debug("SUM: %v VAL: %v COUNT %v", s.Sum, val, s.Count)
	s.SampleSum += sample
	s.Values = append(s.Values, val)
	return nil
}

func (s *StatsdTimerStatItem) ZeroOut() error {
	// reset the values
	s.Values = statdFloat64arr{}
	s.Min = STATSD_ACC_MIN_FLAG
	s.Max = STATSD_ACC_MIN_FLAG
	s.Sum = 0.0
	s.Count = 0
	s.First = STATSD_ACC_MIN_FLAG
	s.Last = STATSD_ACC_MIN_FLAG
	s.start_time = 0
	s.SampleSum = 0
	return nil
}

func (s *StatsdTimerStatItem) Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	root := acc.GetOption("Prefix", "stats").(string)
	pref := acc.GetOption("TimerPrefix", "timers").(string)
	sufix := acc.GetOption("Suffix", "").(string)

	f_key := ""
	if len(root) > 0 {
		f_key = root + "."
	}
	if len(pref) > 0 {
		f_key = f_key + pref + "."
	}
	if len(sufix) > 0 {
		f_key = f_key + sufix + "."
	}
	f_key = f_key + s.InKey.Key

	std := float64(0)
	avg := s.Sum / float64(s.Count)
	cumulativeValues := []float64{s.Min}

	sort.Sort(s.Values)

	for idx, v := range s.Values {
		std += math.Pow((float64(v) - avg), 2.0)
		if idx > 0 {
			cumulativeValues = append(cumulativeValues, v+cumulativeValues[idx-1])
		}
	}
	//log.Notice("Sorted: %v", s.Values)
	//log.Notice("Cums: %v", cumulativeValues)

	std = math.Sqrt(std / float64(s.Count))
	t_stamp := int32(0) //formatter controlled
	tick := time.Now().Unix() - s.start_time
	if tick == 0 {
		tick = 1.0
	}
	min := s.Min
	if min == STATSD_ACC_MIN_FLAG {
		min = 0.0
	}
	max := s.Max
	if max == STATSD_ACC_MIN_FLAG {
		max = 0.0
	}

	fmatter.Write(buffer, &repr.StatName{Key: f_key + ".count"}, float64(s.SampleSum), t_stamp, "c", nil)
	fmatter.Write(buffer, &repr.StatName{Key: f_key + ".count_ps"}, float64(s.SampleSum)/float64(tick), t_stamp, "c", nil)
	fmatter.Write(buffer, &repr.StatName{Key: f_key + ".lower"}, min, t_stamp, "g", nil)
	fmatter.Write(buffer, &repr.StatName{Key: f_key + ".upper"}, max, t_stamp, "g", nil)
	fmatter.Write(buffer, &repr.StatName{Key: f_key + ".sum"}, s.Sum, t_stamp, "g", nil)

	if s.Count == 0 {
		fmatter.Write(buffer, &repr.StatName{Key: f_key + ".mean"}, float64(0.0), t_stamp, "g", nil)
		fmatter.Write(buffer, &repr.StatName{Key: f_key + ".std"}, float64(0.0), t_stamp, "g", nil)
		fmatter.Write(buffer, &repr.StatName{Key: f_key + ".median"}, float64(0.0), t_stamp, "g", nil)
	}
	if s.Count > 0 {
		mid := int64(math.Floor(float64(s.Count) / 2.0))
		median := float64(0.0)
		if math.Mod(float64(mid), 2.0) == 0 {
			median = s.Values[mid]
		} else if s.Count > 1 {
			median = (s.Values[mid-1] + s.Values[mid]) / 2.0
		}

		fmatter.Write(buffer, &repr.StatName{Key: f_key + ".mean"}, float64(avg), t_stamp, "g", nil)
		fmatter.Write(buffer, &repr.StatName{Key: f_key + ".std"}, float64(std), t_stamp, "g", nil)
		fmatter.Write(buffer, &repr.StatName{Key: f_key + ".median"}, float64(median), t_stamp, "g", nil)

		sum := s.Min
		mean := s.Min
		thresholdBoundary := s.Max
		numInThreshold := s.Count

		for _, pct := range acc.GetOption("Thresholds", s.PercentThreshold).([]float64) {
			// handle 0.90 or 90%
			multi := 1.0 / 100.0
			per_mul := 1.0
			if math.Abs(pct) < 1 {
				multi = 1.0
				per_mul = 100.0
			}
			//log.Notice("NumInThreash: %v", numInThreshold)
			p_name := strings.Replace(fmt.Sprintf("%d", int(math.Abs(pct)*per_mul)), ".", "", -1)
			if s.Count > 1 {
				numInThreshold = int64(round(math.Abs(pct) * multi * float64(s.Count)))
				if numInThreshold == 0 {
					continue
				}

				if pct > 0 {
					thresholdBoundary = s.Values[numInThreshold-1]
					sum = cumulativeValues[numInThreshold-1]
				} else {
					thresholdBoundary = s.Values[s.Count-numInThreshold]
					sum = cumulativeValues[s.Count-1] - cumulativeValues[s.Count-numInThreshold-1]
				}

				mean = sum / float64(numInThreshold)
			}

			fmatter.Write(buffer, &repr.StatName{Key: fmt.Sprintf("%s.count_%s", f_key, p_name)}, float64(numInThreshold), t_stamp, "c", nil)
			fmatter.Write(buffer, &repr.StatName{Key: fmt.Sprintf("%s.mean_%s", f_key, p_name)}, float64(mean), t_stamp, "g", nil)
			fmatter.Write(buffer, &repr.StatName{Key: fmt.Sprintf("%s.sum_%s", f_key, p_name)}, float64(sum), t_stamp, "g", nil)

			if pct > 0 {
				fmatter.Write(buffer, &repr.StatName{Key: fmt.Sprintf("%s.upper_%s", f_key, p_name)}, float64(thresholdBoundary), t_stamp, "g", nil)
			} else {
				fmatter.Write(buffer, &repr.StatName{Key: fmt.Sprintf("%s.lower_%s", f_key, p_name)}, float64(thresholdBoundary), t_stamp, "g", nil)
			}
		}
	}
	// reset the ticker
	s.start_time = time.Now().Unix()
}

/******************************/
/** statsd accumulator **/
/******************************/

type StatsdAccumulate struct {
	StatsdStats map[string]StatItem
	OutFormat   FormatterItem
	InTags      repr.SortingTags
	InKeepKeys  bool

	// statsd like options
	LegacyStatsd  bool
	Prefix        string
	Suffix        string
	GaugePrefix   string
	TimerPrefix   string
	CounterPrefix string
	Thresholds    []float64

	mu sync.Mutex
}

func (s *StatsdAccumulate) SetOptions(ops [][]string) error {

	s.GaugePrefix = "gauges"
	s.CounterPrefix = "counters"
	s.TimerPrefix = "timers"
	s.Thresholds = []float64{0.90, 0.95, 0.99}
	s.Suffix = ""
	s.Prefix = "stats"
	s.LegacyStatsd = true

	for _, op := range ops {
		if len(op) != 2 {
			return fmt.Errorf("Options require two arguments")
		}
		if op[0] == "legacyNamespace" {
			ok, err := strconv.ParseBool(op[1])
			if err != nil {
				return err
			}
			s.LegacyStatsd = ok
		}
		if op[0] == "prefixGauge" {
			s.GaugePrefix = op[1]
		}
		if op[0] == "prefixTimer" {
			s.TimerPrefix = op[1]
		}
		if op[0] == "prefixCounter" {
			s.CounterPrefix = op[1]
		}
		if op[0] == "globalSuffix" {
			s.Suffix = op[1]
		}
		if op[0] == "globalPrefix" {
			s.Prefix = op[1]
		}
		if op[0] == "percentThreshold" {
			s.Thresholds = []float64{}
			vals := strings.Split(op[1], ",")

			for _, v := range vals {
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return err
				}
				s.Thresholds = append(s.Thresholds, float64(f))
			}
		}
	}
	return nil
}

func (s *StatsdAccumulate) GetOption(opt string, defaults interface{}) interface{} {
	if opt == "GaugePrefix" {
		return s.GaugePrefix
	}
	if opt == "TimerPrefix" {
		return s.TimerPrefix
	}
	if opt == "CounterPrefix" {
		return s.CounterPrefix
	}
	if opt == "Prefix" {
		return s.Prefix
	}
	if opt == "Suffix" {
		return s.Suffix
	}
	if opt == "Thresholds" {
		return s.Thresholds
	}
	if opt == "LegacyStatsd" {
		return s.LegacyStatsd
	}
	return defaults
}

// noops for statsd
func (s *StatsdAccumulate) SetResolution(dur time.Duration) error {
	return nil
}

func (s *StatsdAccumulate) GetResolution() time.Duration {
	return time.Duration(time.Second)
}

func (s *StatsdAccumulate) Tags() repr.SortingTags {
	return s.InTags
}

func (s *StatsdAccumulate) SetTags(tags repr.SortingTags) {
	s.InTags = tags
}

func (s *StatsdAccumulate) SetKeepKeys(k bool) error {
	s.InKeepKeys = k
	return nil
}

func (s *StatsdAccumulate) Stats() map[string]StatItem {
	return s.StatsdStats
}

func NewStatsdAccumulate() (*StatsdAccumulate, error) {
	return new(StatsdAccumulate), nil
}

func (a *StatsdAccumulate) Init(fmatter FormatterItem) error {
	a.OutFormat = fmatter
	fmatter.SetAccumulator(a)
	a.StatsdStats = make(map[string]StatItem)
	a.SetOptions([][]string{})
	return nil
}

func (a *StatsdAccumulate) Name() (name string) { return STATD_ACC_NAME }

func (a *StatsdAccumulate) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.InKeepKeys {
		for idx := range a.StatsdStats {
			a.StatsdStats[idx].ZeroOut()
		}
	} else {
		a.StatsdStats = nil
		a.StatsdStats = make(map[string]StatItem)
	}
	return nil
}

func (a *StatsdAccumulate) Flush(buf io.Writer) *flushedList {
	fl := new(flushedList)
	a.mu.Lock()
	for _, stats := range a.StatsdStats {
		stats.Write(buf, a.OutFormat, a)
		fl.AddStat(stats.Repr())
	}
	a.mu.Unlock()
	a.Reset()
	return fl
}

func (a *StatsdAccumulate) ProcessLine(line string) (err error) {
	//<key>:<value>|<type>|@<sample>

	stats_arr := strings.Split(line, ":")

	if len(stats_arr) < STATD_ACC_MIN_LEN {
		return fmt.Errorf("Accumulate: Invalid Statds line `%s`", line)
	}

	// val|type|@sample
	key := stats_arr[0]

	val_type := strings.Split(stats_arr[1], "|")
	c_type := "c"

	if len(val_type) > 1 {
		c_type = val_type[1]
	}

	sample := float64(1.0)
	val := val_type[0]

	//special gauge types based on val
	if c_type == "g" {
		if strings.Contains("-", val) {
			c_type = "-g"
		} else if strings.Contains("+", val) {
			c_type = "+g"
		}
	}

	f_val, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fmt.Errorf("Accumulate: Bad Value | Invalid Statds line `%s`", line)
	}

	if len(val_type) == 3 {
		sample_val := strings.Split(val_type[2], "@")
		val = sample_val[0]
		if len(sample_val) != 2 {
			return fmt.Errorf("Accumulate: Sample | Invalid Statds line `%s`", line)
		}
		sample, err = strconv.ParseFloat(sample_val[1], 64)
		if err != nil {
			return fmt.Errorf("Accumulate: Bad Sample number | Invalid Statds line `%s`", line)
		}
	}

	stat_key := key + "|" + c_type
	// now the accumlator
	a.mu.Lock()
	gots, ok := a.StatsdStats[stat_key]
	a.mu.Unlock()

	if !ok {
		if c_type == "ms" || c_type == "h" {
			thres := make([]float64, 3)
			thres[0] = 0.9
			thres[1] = 0.95
			thres[2] = 0.99

			//thres =
			gots = &StatsdTimerStatItem{
				InType:           "ms",
				Sum:              0,
				Min:              STATSD_ACC_MIN_FLAG,
				Max:              STATSD_ACC_MIN_FLAG,
				First:            STATSD_ACC_MIN_FLAG,
				Last:             STATSD_ACC_MIN_FLAG,
				Count:            0,
				InKey:            repr.StatName{Key: key},
				PercentThreshold: thres,
				SampleSum:        0,
			}
		} else {
			gots = &StatsdBaseStatItem{
				InType: c_type,
				Sum:    0.0,
				Min:    STATSD_ACC_MIN_FLAG,
				Max:    STATSD_ACC_MIN_FLAG,
				First:  STATSD_ACC_MIN_FLAG,
				Last:   STATSD_ACC_MIN_FLAG,
				InKey:  repr.StatName{Key: key},
			}
		}
	}
	m_val := float64(f_val)
	// needs to lock internally if needed
	gots.Accumulate(m_val, 1.0/sample, time.Time{})

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.StatsdStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}
