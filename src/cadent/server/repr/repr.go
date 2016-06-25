/*
   An aggrigated stat object
*/

package repr

import (
	"fmt"
	"math"
	"time"
)

type JsonFloat64 float64

func CheckFloat(fl JsonFloat64) JsonFloat64 {
	if math.IsInf(float64(fl), 0) || float64(fl) == math.MinInt64 {
		return JsonFloat64(0)
	}
	return fl
}

// needed to handle "Inf" values
func (s JsonFloat64) MarshalJSON() ([]byte, error) {
	if math.IsInf(float64(s), 0) || float64(s) == math.MinInt64 {
		return []byte("0.0"), nil
	}
	return []byte(fmt.Sprintf("%v", float64(s))), nil
}

type StatRepr struct {
	Key        string      `json:"key"`
	StatKey    string      `json:"stat_key"`
	Min        JsonFloat64 `json:"min"`
	Max        JsonFloat64 `json:"max"`
	Sum        JsonFloat64 `json:"sum"`
	Mean       JsonFloat64 `json:"mean"`
	First      JsonFloat64 `json:"first"`
	Last       JsonFloat64 `json:"last"`
	Count      int64       `json:"count"`
	Time       time.Time   `json:"time_ns"`
	Resolution float64     `json:"resolution"`
	TTL        int64       `json:"ttl"`
}

// rough size of the object in bytes
func (s *StatRepr) ByteSize() int64 {
	return int64(len(s.Key)) + int64(len(s.StatKey)) + 104 // obtained from `reflect.TypeOf(StatRepr{}).Size()`
}

func (s *StatRepr) Copy() *StatRepr {
	return &StatRepr{
		Time:       s.Time,
		StatKey:    s.StatKey,
		Key:        s.Key,
		Mean: s.Mean,
		Min:        s.Min,
		Max:        s.Max,
		Sum:        s.Sum,
		First:      s.First,
		Last:       s.Last,
		Count:      s.Count,
		Resolution: s.Resolution,
		TTL:        s.TTL,
	}
}

// merge a stat together,
// the "time" is chosen as the most future time
// And the First and Last according to that order
func (s *StatRepr) Merge(stat *StatRepr) *StatRepr {
	if stat.Time.UnixNano() <= s.Time.UnixNano() {
		out := s.Copy()
		out.First = stat.First
		if out.Min > stat.Min {
			out.Min = stat.Min
		}
		if out.Max < stat.Max {
			out.Max = stat.Max
		}
		out.Count = stat.Count
		out.Mean = (out.Mean + stat.Mean) / 2.0
		out.Sum = out.Sum + stat.Sum
		return out
	}

	out := stat.Copy()
	out.First = s.First
	if out.Min > s.Min {
		out.Min = s.Min
	}
	if out.Max < s.Max {
		out.Max = s.Max
	}
	out.Count = s.Count
	out.Mean = (out.Mean + s.Mean) / 2.0
	out.Sum = out.Sum + s.Sum
	return out
}

func (s *StatRepr) String() string {
	return fmt.Sprintf("Stat: Mean: %f @ %s/%f/%d", s.Mean, s.Time, s.Resolution, s.TTL)
}

// These two structure is to allow a list of stats in a large queue
// That Queue (which has a LRU bounded size) can then get cycled through
// and "Written" somewhere, or used as a temporary store in case a writing
// backend "slows down" or stops responding for a while
//
type ReprList struct {
	MinTime time.Time  `json:"min_time"`
	MaxTime time.Time  `json:"max_time"`
	Reprs   []StatRepr `json:"stats"`
}

func (s *ReprList) Add(stat StatRepr) StatRepr {
	s.Reprs = append(s.Reprs, stat)
	if stat.Time.Second() > s.MaxTime.Second() {
		s.MaxTime = stat.Time
	}
	// first one added is the first time
	if s.MinTime.Second() == 0 {
		s.MinTime = stat.Time
	}
	return stat
}

func (s *ReprList) Len() int {
	return len(s.Reprs)
}

func (s *ReprList) ToString() string {
	return fmt.Sprintf("StatReprList[%d]", len(s.Reprs))
}
