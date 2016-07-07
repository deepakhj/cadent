/*
	The Metric ProtoBuf Blob

	see protobug.pb.go


*/

package series

import (
	"cadent/server/repr"
	"github.com/golang/protobuf/proto"
	"sync"
	"time"
)

// this can only handle "future pushing times" not random times
type SimpleProtobufTimeSeries struct {
	mu sync.Mutex

	T0      int64
	curTime int64
	Stats   *ProtStats
}

func NewSimpleProtobufTimeSeries(t0 int64) *SimpleProtobufTimeSeries {

	ret := &SimpleProtobufTimeSeries{
		T0:    t0,
		Stats: new(ProtStats),
	}
	return ret
}

func (s *SimpleProtobufTimeSeries) UnmarshalBinary(data []byte) error {
	err := proto.Unmarshal(data, s.Stats)
	return err
}

// the t is the "time we want to add
func (s *SimpleProtobufTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tmp := &ProtStat{
		Time:  proto.Int64(t),
		Min:   proto.Float64(min),
		Max:   proto.Float64(max),
		First: proto.Float64(first),
		Last:  proto.Float64(last),
		Sum:   proto.Float64(sum),
		Count: proto.Int64(count),
	}
	s.Stats.Stats = append(s.Stats.Stats, tmp)
	s.curTime = t
	return nil
}

func (s *SimpleProtobufTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
}

func (s *SimpleProtobufTimeSeries) MarshalBinary() ([]byte, error) {
	return proto.Marshal(s.Stats)
}

func (s *SimpleProtobufTimeSeries) Len() int {
	b, _ := s.MarshalBinary()
	return len(b)
}

func (s *SimpleProtobufTimeSeries) Iter() (iter TimeSeriesIter, err error) {
	s.mu.Lock()
	d := make([]*ProtStat, len(s.Stats.Stats))
	copy(d, s.Stats.Stats)
	s.mu.Unlock()

	iter, err = NewSimpleProtobufIter(d)
	return iter, err
}

func (s *SimpleProtobufTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *SimpleProtobufTimeSeries) LastTime() int64 {
	return s.curTime
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type SimpleProtobufIter struct {
	Stats   []*ProtStat
	curIdx  int
	statLen int
	curStat *ProtStat

	curTime int64

	count int64

	finished bool
	err      error
}

func NewSimpleProtobufIter(stats []*ProtStat) (*SimpleProtobufIter, error) {
	it := &SimpleProtobufIter{
		Stats:   stats,
		curIdx:  0,
		statLen: len(stats),
	}
	return it, nil
}

func (it *SimpleProtobufIter) Next() bool {
	if it.finished || it.curIdx >= it.statLen {
		return false
	}
	it.curStat = it.Stats[it.curIdx]
	it.curIdx++
	return true
}

func (it *SimpleProtobufIter) Values() (int64, float64, float64, float64, float64, float64, int64) {
	return it.curStat.GetTime(),
		float64(it.curStat.GetMin()),
		float64(it.curStat.GetMax()),
		float64(it.curStat.GetFirst()),
		float64(it.curStat.GetLast()),
		float64(it.curStat.GetSum()),
		it.curStat.GetCount()
}

func (it *SimpleProtobufIter) ReprValue() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  time.Unix(0, it.curStat.GetTime()),
		Min:   repr.JsonFloat64(it.curStat.GetMin()),
		Max:   repr.JsonFloat64(it.curStat.GetMax()),
		Last:  repr.JsonFloat64(it.curStat.GetLast()),
		First: repr.JsonFloat64(it.curStat.GetFirst()),
		Sum:   repr.JsonFloat64(it.curStat.GetSum()),
		Count: it.curStat.GetCount(),
	}
}

func (it *SimpleProtobufIter) Error() error {
	return it.err
}
