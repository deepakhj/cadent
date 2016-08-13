/*
	The Metric Json Blob

	format is

	[
		{
			'Time': int64,
			'Count': int64,
			'Max': float64,
			'Min': float64,
			'Last': float64,
			'First': float64,
			'Sum': float64
		},...
	]



*/

package series

import (
	"cadent/server/repr"
	"encoding/json"
	"sync"
	"time"
)

const (
	SIMPLE_JSON_SERIES_TAG = "jarr" // just a flag to note we are using this one at the begining of each blob
	JSON_NAME              = "json"
)

// sort-hand keys for space purposes
type jsonStat struct {
	Time  int64            `json:"t"`
	Min   repr.JsonFloat64 `json:"n"`
	Max   repr.JsonFloat64 `json:"m"`
	Sum   repr.JsonFloat64 `json:"s"`
	First repr.JsonFloat64 `json:"f"`
	Last  repr.JsonFloat64 `json:"l"`
	Count int64            `json:"c"`
}

type JsonStats []jsonStat

type JsonTimeSeries struct {
	mu sync.Mutex

	T0      int64
	curTime int64
	Stats   JsonStats
}

func NewJsonTimeSeries(t0 int64, options *Options) *JsonTimeSeries {
	ret := &JsonTimeSeries{
		T0:    t0,
		Stats: make(JsonStats, 0),
	}
	return ret
}

func (s *JsonTimeSeries) Name() string {
	return JSON_NAME
}
func (s *JsonTimeSeries) HighResolution() bool {
	return true
}

func (s *JsonTimeSeries) Count() int {
	return len(s.Stats)
}

func (s *JsonTimeSeries) UnmarshalBinary(data []byte) error {
	err := json.Unmarshal(data, s.Stats)
	return err
}

func (s *JsonTimeSeries) MarshalBinary() ([]byte, error) {
	return json.Marshal(s.Stats)
}

// this does not "finish" the series
func (s *JsonTimeSeries) Bytes() []byte {
	d, _ := s.MarshalBinary()
	return d
}

func (s *JsonTimeSeries) Len() int {
	b, _ := s.MarshalBinary()
	return len(b)
}

func (s *JsonTimeSeries) Iter() (iter TimeSeriesIter, err error) {
	s.mu.Lock()
	d := make(JsonStats, len(s.Stats))
	copy(d, s.Stats)
	s.mu.Unlock()

	iter, err = NewJsonIter(d)
	return iter, err
}

func (s *JsonTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *JsonTimeSeries) LastTime() int64 {
	return s.curTime
}

// the t is the "time we want to add
func (s *JsonTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Stats = append(s.Stats, jsonStat{
		Time:  t,
		Min:   repr.JsonFloat64(min),
		Max:   repr.JsonFloat64(max),
		First: repr.JsonFloat64(first),
		Last:  repr.JsonFloat64(last),
		Sum:   repr.JsonFloat64(sum),
		Count: count,
	})
	if t > s.curTime {
		s.curTime = t
	}
	if t < s.T0 {
		s.T0 = t
	}
	return nil
}

func (s *JsonTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type JsonIter struct {
	Stats   JsonStats
	curIdx  int
	statLen int
	curStat *jsonStat

	curTime int64
	min     repr.JsonFloat64
	max     repr.JsonFloat64
	first   repr.JsonFloat64
	last    repr.JsonFloat64
	sum     repr.JsonFloat64
	count   int64

	finished bool
	err      error
}

func NewJsonIter(stats JsonStats) (*JsonIter, error) {
	it := &JsonIter{
		Stats:   stats,
		curIdx:  0,
		statLen: len(stats),
	}
	return it, nil
}

func NewJsonIterFromBytes(data []byte) (iter TimeSeriesIter, err error) {
	stats := new(JsonStats)
	err = json.Unmarshal(data, stats)
	if err != nil {
		return nil, err
	}
	return NewJsonIter(*stats)
}

func (it *JsonIter) Next() bool {
	if it.finished || it.curIdx >= it.statLen {
		return false
	}
	it.curStat = &it.Stats[it.curIdx]
	it.curIdx++
	return true
}

func (it *JsonIter) Values() (int64, float64, float64, float64, float64, float64, int64) {
	return it.curStat.Time, float64(it.curStat.Min), float64(it.curStat.Max), float64(it.curStat.First), float64(it.curStat.Last), float64(it.curStat.Sum), it.curStat.Count
}

func (it *JsonIter) ReprValue() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  time.Unix(0, it.curStat.Time),
		Min:   it.curStat.Min,
		Max:   it.curStat.Max,
		Last:  it.curStat.Last,
		First: it.curStat.First,
		Sum:   it.curStat.Sum,
		Count: it.curStat.Count,
	}
}

func (it *JsonIter) Error() error {
	return it.err
}
