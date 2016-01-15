/*
   Here we accumulate statsd metrics and then push to a output format of whatever
   basically an internal statsd server
*/

package accumulator

import (
	"fmt"
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
type statdInt64arr []int64

func (a statdInt64arr) Len() int           { return len(a) }
func (a statdInt64arr) Swap(i int, j int)  { a[i], a[j] = a[j], a[i] }
func (a statdInt64arr) Less(i, j int) bool { return (a[i] - a[j]) > 0 } //this is the sorting statsd uses for its timings

type StatsdBaseStatItem struct {
	InKey      string
	Value      float64
	InType     string
	start_time int64

	mu sync.Mutex
}

func (s *StatsdBaseStatItem) Type() string { return s.InType }
func (s *StatsdBaseStatItem) Key() string  { return s.InKey }
func (s *StatsdBaseStatItem) Out(fmatter FormatterItem, acc AccumulatorItem) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	root := acc.GetOption("Prefix", "stats").(string)
	pref := acc.GetOption("CounterPrefix", "counter").(string)
	sufix := acc.GetOption("Suffix", "timers").(string)
	f_key := root + "." + pref + "."
	if len(sufix) > 0 {
		f_key = f_key + sufix + "."
	}

	c_type := "c"
	val := s.Value

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
	}

	// reset the ticker
	s.start_time = time.Now().Unix()
	if c_type == "c" {
		val_p_s := val / float64(tick)
		if acc.GetOption("LegacyStatsd", true).(bool) {
			rate_pref := "stats_counts."
			if len(sufix) > 0 {
				rate_pref = rate_pref + sufix + "."
			}
			return []string{
				fmatter.ToString(
					f_key+s.InKey,
					val,
					0, // let formatter handle the time,
					c_type,
					acc.Tags(),
				),
				fmatter.ToString(
					rate_pref+s.InKey,
					val_p_s,
					0, // let formatter handle the time,
					c_type,
					acc.Tags(),
				),
			}
		} else {

			return []string{
				fmatter.ToString(
					f_key+"count."+s.InKey,
					val,
					0, // let formatter handle the time,
					c_type,
					acc.Tags(),
				),
				fmatter.ToString(
					f_key+"rate."+s.InKey,
					val_p_s,
					0, // let formatter handle the time,
					c_type,
					acc.Tags(),
				),
			}
		}
	}
	return []string{
		fmatter.ToString(
			f_key+s.InKey,
			val,
			0, // let formatter handle the time,
			c_type,
			acc.Tags(),
		),
	}
}

func (s *StatsdBaseStatItem) Accumulate(val float64) error {

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.start_time == 0 {
		s.start_time = time.Now().Unix()
	}

	switch {
	case s.InType == "c": //counter
		s.Value += val
	case s.InType == "g": //gauage
		s.Value = val
	case s.InType == "-g": //gauage negate
		s.Value -= val
	case s.InType == "+g": //gauage add
		s.Value += val
	}
	return nil
}

/** timer type **/

type StatsdTimerStatItem struct {
	InKey  string
	InType string
	Value  int64
	Count  int64
	Min    int64
	Max    int64
	Values statdInt64arr

	PercentThreshold []float64

	start_time int64

	mu sync.Mutex
}

func (s *StatsdTimerStatItem) Key() string  { return s.InKey }
func (s *StatsdTimerStatItem) Type() string { return s.InType }

func (s *StatsdTimerStatItem) Accumulate(val float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.start_time == 0 {
		s.start_time = time.Now().Unix()
	}
	s.Count += 1
	vv := int64(val)
	s.Value += vv
	if s.Min == STATSD_ACC_MIN_FLAG || s.Min > vv {
		s.Min = vv
	}
	if s.Max < vv {
		s.Max = vv
	}
	s.Values = append(s.Values, vv)
	return nil
}

func (s *StatsdTimerStatItem) Out(fmatter FormatterItem, acc AccumulatorItem) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	root := acc.GetOption("Prefix", "stats").(string)
	pref := acc.GetOption("TimerPrefix", "timers").(string)
	sufix := acc.GetOption("Suffix", "timers").(string)

	f_key := root + "." + pref + "."
	if len(sufix) > 0 {
		f_key = f_key + sufix + "."
	}
	f_key = f_key + s.InKey

	std := float64(0)
	avg := float64(s.Value / s.Count)
	cumulativeValues := []int64{s.Min}

	sort.Sort(s.Values)

	for idx, v := range s.Values {
		std += math.Pow((float64(v) - avg), 2.0)
		if idx > 0 {
			cumulativeValues = append(cumulativeValues, v+cumulativeValues[idx-1])
		}

	}
	std = math.Sqrt(std / float64(s.Count))
	t_stamp := int32(0) //formatter controlled
	tick := time.Now().Unix() - s.start_time
	if tick == 0 {
		tick = 1.0
	}
	base := []string{
		fmatter.ToString(f_key+".count", float64(s.Count), t_stamp, "c", nil),
		fmatter.ToString(f_key+".count_ps", float64(s.Count)/float64(tick), t_stamp, "c", nil),
		fmatter.ToString(f_key+".mean", float64(avg), t_stamp, "g", nil),
		fmatter.ToString(f_key+".lower", float64(s.Min), t_stamp, "g", nil),
		fmatter.ToString(f_key+".upper", float64(s.Max), t_stamp, "g", nil),
		fmatter.ToString(f_key+".std", float64(std), t_stamp, "g", nil),
	}

	if s.Count > 0 {
		sum := s.Min
		mean := s.Min
		thresholdBoundary := s.Max

		for _, pct := range acc.GetOption("Thresholds", s.PercentThreshold).([]float64) {
			multi := 1.0 / 100.0
			if math.Abs(pct) < 1 {
				multi = 1.0
			}
			numInThreshold := int64(math.Abs(pct) * multi * float64(s.Count))

			if numInThreshold == 0 {
				continue
			}

			if pct > 0 {
				thresholdBoundary = cumulativeValues[numInThreshold-1]
				sum = cumulativeValues[numInThreshold-1]
			} else {
				thresholdBoundary = cumulativeValues[s.Count-numInThreshold]
				sum = cumulativeValues[s.Count-1] - cumulativeValues[s.Count-numInThreshold-1]
			}

			mean = sum / numInThreshold
			base = append(base,
				[]string{
					fmatter.ToString(fmt.Sprintf("%s.count_%d", f_key, int(pct*100)), float64(numInThreshold), t_stamp, "c", nil),
					fmatter.ToString(fmt.Sprintf("%s.mean_%d", f_key, int(pct*100)), float64(mean), t_stamp, "g", nil),
				}...,
			)
			if pct > 0 {
				base = append(
					base,
					fmatter.ToString(fmt.Sprintf("%s.upper_%d", f_key, int(pct*100)), float64(thresholdBoundary), t_stamp, "g", nil),
				)
			} else {
				base = append(
					base,
					fmatter.ToString(fmt.Sprintf("%s.lower_%d", f_key, int(pct*100)), float64(thresholdBoundary), t_stamp, "g", nil),
				)
			}
		}
	}
	// reset the ticker
	s.start_time = time.Now().Unix()
	return base
}

/******************************/
/** statsd accumulator **/
/******************************/

type StatsdAccumulate struct {
	StatsdStats map[string]StatItem
	OutFormat   FormatterItem
	InTags      []AccumulatorTags

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
			s.Prefix = op[1]
		}
		if op[0] == "globalPrefix" {
			s.Suffix = op[1]
		}
		if op[0] == "percentThreshold" {
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

func (s *StatsdAccumulate) Tags() []AccumulatorTags {
	return s.InTags
}

func (s *StatsdAccumulate) SetTags(tags []AccumulatorTags) {
	s.InTags = tags
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
	return nil
}

func (a *StatsdAccumulate) Name() (name string) { return STATD_ACC_NAME }

func (a *StatsdAccumulate) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.StatsdStats = nil
	a.StatsdStats = make(map[string]StatItem)
	return nil
}

func (a *StatsdAccumulate) Flush() []string {
	base := []string{}
	a.mu.Lock()
	for _, stats := range a.StatsdStats {
		base = append(base, stats.Out(a.OutFormat, a)...)
	}
	a.mu.Unlock()
	a.Reset()
	return base
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
				Value:            0,
				Min:              STATSD_ACC_MIN_FLAG,
				Max:              0,
				Count:            0,
				InKey:            key,
				PercentThreshold: thres,
			}
		} else {
			gots = &StatsdBaseStatItem{
				InType: c_type,
				Value:  0.0,
				InKey:  key,
			}
		}
	}
	m_val := float64(f_val) / sample
	// needs to lock internally if needed
	gots.Accumulate(m_val)

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.StatsdStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}
