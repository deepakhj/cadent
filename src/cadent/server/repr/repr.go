/*
   An aggrigated stat object
*/

package repr

import (
	"cadent/server/utils"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type JsonFloat64 float64

func CheckFloat(fl JsonFloat64) JsonFloat64 {
	if math.IsNaN(float64(fl)) || math.IsInf(float64(fl), 0) || float64(fl) == math.MinInt64 {
		return JsonFloat64(0)
	}
	return fl
}

// needed to handle "Inf" values
func (s JsonFloat64) MarshalJSON() ([]byte, error) {
	if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) || float64(s) == math.MinInt64 {
		return []byte("0.0"), nil
	}
	return []byte(fmt.Sprintf("%v", float64(s))), nil
}

// for GC purposes
const SPACE_SEPARATOR = " "
const DOUBLE_SPACE_SEPARATOR = "  "
const DOT_SEPARATOR = "."
const COMMA_SEPARATOR = ","
const EQUAL_SEPARATOR = "="
const IS_SEPARATOR = "_is_"
const UNDERSCORE_SEPARATOR = "_"
const DASH_SEPARATOR = "-"
const NEWLINE_SEPARATOR = "\n"

var SPACE_SEPARATOR_BYTE = []byte(" ")
var DOUBLE_SPACE_SEPARATOR_BYTE = []byte("  ")
var DOT_SEPARATOR_BYTE = []byte(".")
var COMMA_SEPARATOR_BYTE = []byte(",")
var EQUAL_SEPARATOR_BYTE = []byte("=")
var IS_SEPARATOR_BYTE = []byte("_is_")
var NEWLINE_SEPARATOR_BYTES = []byte("\n")
var NEWLINE_SEPARATOR_BYTE = NEWLINE_SEPARATOR_BYTES[0]
var DASH_SEPARATOR_BYTES = []byte("-")
var UNDERSCORE_SEPARATOR_BYTES = []byte("_")

type SortingTags [][]string

// make a tag array from a string input inder a few conditions
// tag=val.tag=val.tag=val
// or
// tag_is_val.tag_is_val
// or
// tag=val,tag=val
// or
// tag_is_val,tag_is_val
// or
// tag=val tag=val
// or
// tag_is_val tag_is_val
func SortingTagsFromString(key string) SortingTags {

	var parse_tsg []string
	if strings.Contains(key, DOT_SEPARATOR) {
		parse_tsg = strings.Split(key, DOT_SEPARATOR)
	} else if strings.Contains(key, COMMA_SEPARATOR) {
		parse_tsg = strings.Split(key, COMMA_SEPARATOR)
	} else {
		parse_tsg = strings.Split(key, SPACE_SEPARATOR)
	}
	return SortingTagsFromArray(parse_tsg)
}

func SortingTagsFromArray(keys []string) SortingTags {

	outs := make(SortingTags, 0)

	for _, tgs := range keys {
		spls := strings.Split(tgs, EQUAL_SEPARATOR)
		if len(spls) < 2 {
			// try "_is_"
			spls = strings.Split(tgs, IS_SEPARATOR)
			if len(spls) < 2 {
				continue
			}
		}

		outs = append(outs, []string{spls[0], spls[1]})
	}
	return outs
}

func (p SortingTags) Len() int           { return len(p) }
func (p SortingTags) Less(i, j int) bool { return strings.Compare(p[i][0], p[j][0]) < 0 }
func (p SortingTags) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (s SortingTags) String() string {
	return s.ToStringSep("=", ".")
}

func (s SortingTags) IsEmpty() bool {
	return len(s) == 0
}

func (s SortingTags) Merge(tags SortingTags) SortingTags {
	n_tags := make(SortingTags, 0)
	for _, tag := range tags {
		got := false
		for _, o_tag := range s {
			if tag[0] == o_tag[0] {
				n_tags = append(n_tags, []string{tag[0], tag[1]})
				got = true
				break
			}
		}
		if !got {
			n_tags = append(n_tags, []string{tag[0], tag[1]})
		}
	}
	return n_tags
}

func (s SortingTags) ToStringSep(wordsep string, tagsep string) string {
	str := make([]string, len(s))
	for idx, tag := range s {
		str[idx] = strings.Join(tag, wordsep)
	}
	return strings.Join(str, tagsep)
}

func (s SortingTags) WriteBytes(buf io.Writer, wordsep []byte, tagsep []byte) {

	l := len(s)
	for idx, tag := range s {
		buf.Write([]byte(tag[0]))
		buf.Write(wordsep)
		buf.Write([]byte(tag[1]))
		if idx < l-1 {
			buf.Write(tagsep)
		}
	}
}

// these methods are for Metrics2.0 things
func (s SortingTags) Find(name string) string {
	for _, tag := range s {
		if tag[0] == name {
			return tag[1]
		}
	}
	return ""
}

// these methods are for Metrics2.0 things
func (s SortingTags) Set(name string, val string) {
	for idx, tag := range s {
		if tag[0] == name {
			s[idx][1] = val
			return
		}
	}
	s = append(s, []string{name, val})
}

// Hz, B, etc
func (s SortingTags) Unit() string {
	return s.Find("unit")
}

// counter, rate, gauge, count, timestamp
func (s SortingTags) Mtype() string {
	got := s.Find("mtype")
	if got == "" {
		got = s.Find("target_type")
	}
	return got
}

// min, max, lower, upper, mean, std, sum, upper_\d+, lower_\d+, min_\d+, max_\d+
func (s SortingTags) Stat() string {
	return s.Find("stat")
}

type StatId uint64

type StatName struct {
	Key        string      `json:"key"`
	Tags       SortingTags `json:"tags"`
	MetaTags   SortingTags `json:"meta_tags"`
	Resolution uint32      `json:"resolution"`
	TTL        uint32      `json:"ttl"`
}

// take the various "parts" (keys, resolution, tags) and return a basic md5 hash of things
func (s *StatName) UniqueId() StatId {
	buf := fnv.New64a()

	byte_buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(byte_buf)

	fmt.Fprintf(byte_buf, "%s:%s", s.Key, s.SortedTags())
	buf.Write(byte_buf.Bytes())
	return StatId(buf.Sum64())
}

// nice "sqeeuzed" string
func (s *StatName) UniqueIdString() string {
	id := s.UniqueId()
	return strconv.FormatUint(uint64(id), 36)
}

// return an array of [ [name, val] ...] sorted by name
func (s *StatName) SortedTags() SortingTags {
	sort.Sort(s.Tags)
	return s.Tags
}

func (s *StatName) SortedMetaTags() SortingTags {
	sort.Sort(s.MetaTags)
	return s.MetaTags
}

// return an array of [ [name, val] ...] sorted by name
func (s *StatName) MergeTags(tags SortingTags) SortingTags {
	n_tags := make(SortingTags, 0)
	for _, tag := range tags {
		got := false
		for _, o_tag := range s.Tags {
			if tag[0] == o_tag[0] {
				n_tags = append(n_tags, []string{tag[0], tag[1]})
				got = true
				break
			}
		}
		if !got {
			n_tags = append(n_tags, []string{tag[0], tag[1]})
		}
	}
	s.Tags = n_tags
	return n_tags
}

// return an array of [ [name, val] ...] sorted by name
func (s *StatName) ByteSize() int64 {
	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)
	fmt.Fprintf(buf, "%s%v", s.Key, s.SortedTags())
	return int64(buf.Len())
}

func (s *StatName) IsBlank() bool {
	return len(s.Key) == 0
}

// this is a "graphite" compatible name of {key}.{name=value}.{name=value}
// with Names of the tags SORTED
func (s *StatName) Name() string {
	s_tags := s.SortedTags()
	str := make([]string, 1+len(s_tags))
	str[0] = s.Key
	for idx, tg := range s_tags {
		str[idx+1] = strings.Join(tg, "=")
	}
	return strings.Join(str, ".")
}

func (s *StatName) AggFunc() AggType {
	h_stat := s.Tags.Find("stat")
	if h_stat != "" {
		return AggTypeFromTag(h_stat)
	}
	return GuessReprValueFromKey(s.Key)
}

type StatRepr struct {
	Name StatName

	Time  time.Time   `json:"time_ns"`
	Min   JsonFloat64 `json:"min"`
	Max   JsonFloat64 `json:"max"`
	Sum   JsonFloat64 `json:"sum"`
	First JsonFloat64 `json:"first"`
	Last  JsonFloat64 `json:"last"`
	Count int64       `json:"count"`
}

// take the various "parts" (keys, resolution, tags) and return a basic md5 hash of things
func (s *StatRepr) UniqueId() uint64 {
	buf := fnv.New64a()
	fmt.Fprintf(buf, "%s:%d:%v", s.Name.Key, s.Name.Resolution, s.Name.SortedTags())
	return buf.Sum64()
}

// rough size of the object in bytes
func (s *StatRepr) ByteSize() int64 {
	if s == nil {
		return 0
	}
	return s.Name.ByteSize() + int64(8*64)
}

func (s *StatRepr) Copy() *StatRepr {
	obj := *s
	return &obj

}

func (s *StatRepr) AggValue(aggfunc AggType) float64 {
	switch aggfunc {
	case SUM:
		return float64(s.Sum)
	case MIN:
		return float64(s.Min)
	case MAX:
		return float64(s.Max)
	case FIRST:
		return float64(s.First)
	case LAST:
		return float64(s.Last)
	default:
		if s.Count > 0 {
			return float64(s.Sum) / float64(s.Count)
		}
		return math.NaN()
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
		out.Count = out.Count + stat.Count
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
	out.Sum = out.Sum + s.Sum
	return out
}

// basically a "uniqueness" key for dedupe attempts in list
func (s *StatRepr) UniqueKey() string {
	return fmt.Sprintf("%d:%d", s.Name.UniqueId(), s.Time.UnixNano())
}

// will be "true" of the Id + time are the same
func (s *StatRepr) IsSameStat(stat *StatRepr) bool {
	return s.Name.UniqueId() == stat.Name.UniqueId() && s.Time.Equal(stat.Time)
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
	m := float64(s.Sum)
	if s.Count > 0 {
		m = float64(s.Sum) / float64(s.Count)
	}
	return fmt.Sprintf("Stat: Mean: %f @ %s/%d/%d", m, s.Time, s.Name.Resolution, s.Name.TTL)
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
