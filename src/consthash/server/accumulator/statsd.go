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
	InKey  string
	Value  float64
	InType string

	mu sync.Mutex
}

func (s *StatsdBaseStatItem) Type() string { return s.InType }
func (s *StatsdBaseStatItem) Key() string  { return s.InKey }
func (s *StatsdBaseStatItem) Out(fmatter FormaterItem) []string {
	pref := "stats.counters"
	if s.InType == "g" {
		pref = "stats.gauges"
	}
	return []string{
		fmatter.ToString(
			pref+"."+s.InKey,
			s.Value,
			0, // let formatter handle the time,
			"c",
			nil,
		),
	}
}

func (s *StatsdBaseStatItem) Accumulate(val float64) error {

	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *StatsdTimerStatItem) Out(fmatter FormaterItem) []string {
	pref := "stats.timers"
	f_key := pref + "." + s.InKey

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

		for _, pct := range s.PercentThreshold {
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
	OutFormat   FormaterItem

	mu sync.Mutex
}

func (s *StatsdAccumulate) Stats() map[string]StatItem {
	return s.StatsdStats
}

func NewStatsdAccumulate(fmatter FormaterItem) (*StatsdAccumulate, error) {

	acc := &StatsdAccumulate{
		StatsdStats: make(map[string]StatItem),
		OutFormat:   fmatter,
	}

	return acc, nil
}

func (a *StatsdAccumulate) Name() (name string) { return STATD_ACC_NAME }

func (a *StatsdAccumulate) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.StatsdStats = nil
	a.StatsdStats = make(map[string]StatItem)

}

func (a *StatsdAccumulate) Flush() []string {
	base := []string{}
	a.mu.Lock()
	for _, stats := range a.StatsdStats {
		base = append(base, stats.Out(a.OutFormat)...)
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
