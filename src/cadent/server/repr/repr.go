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
   An aggregated stat object

   a note  on TagMode ::

    the default behavior is to have metrics2 "identifier" tags, and all other tags are meta tags
    sometimes this is not desired, so if the StatName.SetTagMode("all") will make all tags
    identifiers

    Note: the basic "structs" come from the Protobuf generator
*/

package repr

import (
	"cadent/server/utils"
	//"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StatId uint64

type NilJsonFloat64 float64

var nullBytes = []byte("null")

// a little hash pool for GC pressure easing

var fn64avPool sync.Pool

func GetFnv64a() hash.Hash64 {
	x := fn64avPool.Get()
	if x == nil {
		return fnv.New64a()
	}
	out := x.(hash.Hash64)
	out.Reset()
	return out
}

func PutFnv64a(spl hash.Hash64) {
	fn64avPool.Put(spl)
}

// needed to handle "Inf" values
func (s NilJsonFloat64) MarshalJSON() ([]byte, error) {
	if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) || float64(s) == math.MinInt64 {
		return nullBytes, nil
	}
	return []byte(fmt.Sprintf("%v", float64(s))), nil
}

func CheckFloat(fl float64) float64 {
	if math.IsNaN(fl) || math.IsInf(fl, 0) || fl == math.MinInt64 {
		return 0
	}
	return fl
}

func JsonFloat64(fl float64) []byte {
	if math.IsNaN(fl) || math.IsInf(fl, 0) || fl == math.MinInt64 {
		return nil
	}
	return []byte(fmt.Sprintf("%v", float64(fl)))
}

func (s *StatName) SetTagMode(mode string) {
	s.TagMode = TAG_METRICS2
	switch mode {
	case "all":
		s.TagMode = TAG_ALLTAGS
	}
	s.XXX_uniqueId = 0
	s.XXX_uniqueIdstr = ""

}

func (s *StatName) SetKey(name string) {
	s.Key = name
	if name != s.Key {
		s.XXX_uniqueId = 0
		s.XXX_uniqueIdstr = ""
	}
}

func (s *StatName) Copy() *StatName {
	cp := &StatName{
		Key:             s.Key,
		Ttl:             s.Ttl,
		Resolution:      s.Resolution,
		Tags:            s.Tags,
		MetaTags:        s.MetaTags,
		XXX_uniqueId:    0,
		XXX_uniqueIdstr: "",
		TagMode:         s.TagMode,
	}
	return cp
}

// take the various "parts" (keys, resolution, tags) and return a basic md5 hash of things
func (s *StatName) UniqueId() StatId {
	if s.XXX_uniqueId > 0 {
		return StatId(s.XXX_uniqueId)
	}
	buf := GetFnv64a()
	defer PutFnv64a(buf)

	byte_buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(byte_buf)

	// all tags or just intrinsic tags
	switch s.TagMode {
	case TAG_ALLTAGS:
		if len(s.MetaTags) > 0 {
			fmt.Fprintf(byte_buf, "%s:%s:%s", s.Key, s.SortedTags(), s.SortedMetaTags())
		} else {
			fmt.Fprintf(byte_buf, "%s:%s", s.Key, s.SortedTags())
		}
		buf.Write(byte_buf.Bytes())
	default:
		fmt.Fprintf(byte_buf, "%s:%s", s.Key, s.SortedTags())
		buf.Write(byte_buf.Bytes())
	}

	s.XXX_uniqueId = buf.Sum64()
	return StatId(s.XXX_uniqueId)
}

// include both meta and tags and key
// usefull for "indexers" that don't want to re-index things, but if the meta
// tags change we need to add those to the xref
func (s *StatName) UniqueIdAllTags() StatId {
	buf := GetFnv64a()
	defer PutFnv64a(buf)

	byte_buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(byte_buf)

	fmt.Fprintf(byte_buf, "%s:%s:%s", s.Key, s.SortedTags(), s.SortedMetaTags())
	buf.Write(byte_buf.Bytes())
	return StatId(buf.Sum64())
}

// nice "sqeeuzed" string
// keep it in the object as the computation can yield many GC things from the Fprintf above
func (s *StatName) UniqueIdString() string {
	if s.XXX_uniqueIdstr == "" {
		id := s.UniqueId()
		s.XXX_uniqueIdstr = strconv.FormatUint(uint64(id), 36)
	}
	return s.XXX_uniqueIdstr
}

func (s *StatName) StringToUniqueId(inid string) StatId {
	t_int, err := strconv.ParseUint(inid, 36, 64)
	if err != nil {
		return StatId(0)
	}
	return StatId(t_int)
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
			if tag.Name == o_tag.Name {
				n_tags = append(n_tags, &Tag{tag.Name, tag.Value})
				got = true
				break
			}
		}
		if !got {
			n_tags = append(n_tags, &Tag{Name: tag.Name, Value: tag.Value})
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
		str[idx+1] = tg.Join(EQUAL_SEPARATOR)
	}
	return strings.Join(str, ".")
}

func (s *StatName) AggType() uint32 {
	h_stat := s.Tags.Find("stat")
	if h_stat != "" {
		return AggTypeFromTag(h_stat)
	}
	return GuessReprValueFromKey(s.Key)
}

func (s *StatName) AggFunc() AGG_FUNC {
	h_stat := s.Tags.Find("stat")
	if h_stat != "" {
		return AggFuncFromTag(h_stat)
	}
	return GuessAggFuncFromKey(s.Key)
}

func (s *StatName) MergeMetric2Tags(itgs SortingTags) {
	switch s.TagMode {
	case TAG_ALLTAGS:
		s.Tags = s.Tags.Merge(itgs)
		s.MetaTags = SortingTags{}
	default:
		s.Tags, s.MetaTags = SplitIntoMetric2Tags(MergeMetric2Tags(itgs, s.Tags, s.MetaTags))
	}
	// need to invalidate the unique ids as the tags may have changed
	s.XXX_uniqueId = 0
	s.XXX_uniqueIdstr = ""
}

// need to overload this to get "NANs" to encode properly, we don't want panics on
// over/under flows errs
/*
const jsonTMPL = `{"name":%s,"time":%d,"min":%v,"max":%v,"sum":%v,"last":%v,"count":%v}`

func (s StatRepr) MarshalJSON() ([]byte, error) {

	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)

	nm, err := json.Marshal(s.Name)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(
		buf,
		jsonTMPL,
		nm,
		s.Time,
		JsonFloat64(s.Min),
		JsonFloat64(s.Max),
		JsonFloat64(s.Sum),
		JsonFloat64(s.Last),
		s.Count,
	)
	return buf.Bytes(), nil
}
*/
func (s *StatRepr) ToTime() time.Time {
	// nano or second tstamps
	if s.Time > 2147483647 {
		return time.Unix(0, s.Time)
	}

	return time.Unix(s.Time, 0)
}

func (s *StatRepr) ToUnix() uint32 {
	// nano or second tstamps
	if s.Time > 2147483647 {
		return uint32(time.Unix(0, s.Time).Unix())
	}

	return uint32(time.Unix(s.Time, 0).Unix())
}

// take the various "parts" (keys, resolution, tags) and return a basic fmv64a hash of things
func (s *StatRepr) UniqueId() uint64 {
	buf := GetFnv64a()
	defer PutFnv64a(buf)

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
	cp := &StatRepr{
		Name:  s.Name.Copy(),
		Min:   s.Min,
		Max:   s.Max,
		Count: s.Count,
		Last:  s.Last,
		Sum:   s.Sum,
	}
	return cp

}

func (s *StatRepr) AggValue(aggfunc uint32) float64 {
	switch aggfunc {
	case SUM:
		return float64(s.Sum)
	case MIN:
		return float64(s.Min)
	case MAX:
		return float64(s.Max)
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
// and Last according to that order
func (s *StatRepr) Merge(stat *StatRepr) *StatRepr {
	if stat.Time <= s.Time {
		out := s.Copy()
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

func (s *StatRepr) SetKey(name string) {
	s.Name.SetKey(name)
}

// basically a "uniqueness" key for dedupe attempts in list
func (s *StatRepr) UniqueKey() string {
	return fmt.Sprintf("%d:%d", s.Name.UniqueId(), s.Time)
}

// will be "true" of the Id + time are the same
func (s *StatRepr) IsSameStat(stat *StatRepr) bool {
	return s.Name.UniqueId() == stat.Name.UniqueId() && s.Time == stat.Time
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

/*
func (s *StatRepr) String() string {
	m := float64(s.Sum)
	if s.Count > 0 {
		m = float64(s.Sum) / float64(s.Count)
	}
	return fmt.Sprintf("Stat: Mean: %f @ %s/%d/%d", m, s.Time, s.Name.Resolution, s.Name.Ttl)
}
*/

// time sort
type StatReprSlice []*StatRepr

func (p StatReprSlice) Len() int { return len(p) }
func (p StatReprSlice) Less(i, j int) bool {
	return p[i] != nil && p[j] != nil && p[i].Time < p[j].Time
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
	t_t := stat.ToTime()
	if t_t.Second() > s.MaxTime.Second() {
		s.MaxTime = t_t
	}
	// first one added is the first time
	if s.MinTime.Second() == 0 {
		s.MinTime = t_t
	}
	return stat
}

func (s *ReprList) Len() int {
	return len(s.Reprs)
}

func (s *ReprList) ToString() string {
	return fmt.Sprintf("StatReprList[%d]", len(s.Reprs))
}
