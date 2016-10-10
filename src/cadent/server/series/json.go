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
	The Metric Json Blob

	format is

	[
		{
			'Time': int64,
			'Count': int64,
			'Max': float64,
			'Min': float64,
			'Last': float64,
			'Sum': float64
		},...
	]



*/

package series

import (
	"cadent/server/repr"
	"encoding/json"
	"sync"
)

const (
	SIMPLE_JSON_SERIES_TAG = "jarr" // just a flag to note we are using this one at the begining of each blob
	JSON_NAME              = "json"
)

// sort-hand keys for space purposes
type jsonStat struct {
	Time  int64   `json:"t"`
	Min   float64 `json:"n"`
	Max   float64 `json:"m"`
	Sum   float64 `json:"s"`
	Last  float64 `json:"l"`
	Count int64   `json:"c"`
}

type JsonStats []jsonStat

type JsonTimeSeries struct {
	mu *sync.Mutex

	T0      int64
	curTime int64
	Stats   JsonStats
}

func NewJsonTimeSeries(t0 int64, options *Options) *JsonTimeSeries {
	ret := &JsonTimeSeries{
		T0:    t0,
		mu:    new(sync.Mutex),
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

func (s *JsonTimeSeries) Copy() TimeSeries {

	g := *s
	g.mu = new(sync.Mutex)
	s.mu.Lock()
	defer s.mu.Unlock()

	g.Stats = make(JsonStats, len(s.Stats))
	copy(g.Stats, s.Stats)

	return &g
}

// the t is the "time we want to add
func (s *JsonTimeSeries) AddPoint(t int64, min float64, max float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Stats = append(s.Stats, jsonStat{
		Time:  t,
		Min:   repr.CheckFloat(min),
		Max:   repr.CheckFloat(max),
		Last:  repr.CheckFloat(last),
		Sum:   repr.CheckFloat(sum),
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
	return s.AddPoint(stat.Time, float64(stat.Min), float64(stat.Max), float64(stat.Last), float64(stat.Sum), stat.Count)
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type JsonIter struct {
	Stats   JsonStats
	curIdx  int
	statLen int
	curStat *jsonStat

	curTime int64
	min     float64
	max     float64
	last    float64
	sum     float64
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

func (it *JsonIter) Values() (int64, float64, float64, float64, float64, int64) {
	return it.curStat.Time, it.curStat.Min, it.curStat.Max, it.curStat.Last, it.curStat.Sum, it.curStat.Count
}

func (it *JsonIter) ReprValue() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  it.curStat.Time,
		Min:   it.curStat.Min,
		Max:   it.curStat.Max,
		Last:  it.curStat.Last,
		Sum:   it.curStat.Sum,
		Count: it.curStat.Count,
	}
}

func (it *JsonIter) Error() error {
	return it.err
}
