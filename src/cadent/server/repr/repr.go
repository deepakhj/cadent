/*
   An aggrigated stat object
*/

package repr

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
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

type SortingTags [][]string

func (p SortingTags) Len() int           { return len(p) }
func (p SortingTags) Less(i, j int) bool { return strings.Compare(p[i][0], p[j][0]) > 0 }
func (p SortingTags) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type StatId uint64

type StatName struct {
	Key     string      `json:"key"`
	StatKey string      `json:"stat_key"`
	Tags    SortingTags `json:"tags"`
}

// take the various "parts" (keys, resolution, tags) and return a basic md5 hash of things
func (s *StatName) UniqueId() StatId {
	buf := fnv.New64a()
	fmt.Fprintf(buf, "%s:%s:%v", s.Key, s.StatKey, s.SortedTags())
	return StatId(buf.Sum64())
}

// return an array of [ [name, val] ...] sorted by name
func (s *StatName) SortedTags() SortingTags {
	sort.Sort(s.Tags)
	return s.Tags
}

// return an array of [ [name, val] ...] sorted by name
func (s *StatName) ByteSize() int {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s%s%v", s.Key, s.StatKey, s.SortedTags())
	return buf.Len()
}

func (s *StatName) IsBlank() bool {
	return len(s.Key) == 0
}

type StatRepr struct {
	Name StatName

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

// take the various "parts" (keys, resolution, tags) and return a basic md5 hash of things
func (s *StatRepr) UniqueId() uint64 {
	buf := fnv.New64a()
	fmt.Fprintf(buf, "%s:%s:%f:%v", s.Name.Key, s.Name.StatKey, s.Resolution, s.Name.SortedTags())
	return buf.Sum64()
}

// rough size of the object in bytes
func (s *StatRepr) ByteSize() int64 {
	if s == nil {
		return 0
	}
	return int64(len(s.Name.Key)) + int64(len(s.Name.StatKey)) + 104 // obtained from `reflect.TypeOf(StatRepr{}).Size()`
}

func (s *StatRepr) Copy() *StatRepr {
	obj := *s
	return &obj

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
		out.Count = out.Count + stat.Count
		out.Mean = (out.Sum + s.Sum) / JsonFloat64(out.Count)
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
	out.Count = out.Count + s.Count
	out.Mean = (out.Sum + s.Sum) / JsonFloat64(out.Count)
	out.Sum = out.Sum + s.Sum
	return out
}

// basically a "uniqueness" key for dedupe attempts in list
func (s *StatRepr) UniqueKey() string {
	return fmt.Sprintf("%s:%d:%f", s.Name.Key, s.Time.UnixNano(), s.Resolution)
}

// will be "true" of the key + resolution + time are the same
func (s *StatRepr) IsSameStat(stat *StatRepr) bool {
	return s.Name.Key == s.Name.StatKey && s.Resolution == stat.Resolution && s.Time.Equal(stat.Time)
}

// if this stat is in a list
func (s *StatRepr) ContainsSelf(stats []*StatRepr) bool {
	for _, s2 := range stats {
		if s.IsSameStat(s2) {
			return true
		}
	}
	return false
}

func (s *StatRepr) String() string {
	m := float64(s.Mean)
	if s.Count > 0 && m == 0 {
		m = float64(s.Sum) / float64(s.Count)
	}
	return fmt.Sprintf("Stat: Mean: %f @ %s/%f/%d", m, s.Time, s.Resolution, s.TTL)
}

// time sort
type StatReprSlice []*StatRepr

func (p StatReprSlice) Len() int { return len(p) }
func (p StatReprSlice) Less(i, j int) bool {
	return p[i] != nil && p[j] != nil && p[i].Time.Before(p[j].Time)
}
func (p StatReprSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

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
