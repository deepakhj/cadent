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

const (
	PROTOBUF_NAME = "protbuf"
)

// add sorting option for ProtStats

// this can only handle "future pushing times" not random times
type ProtobufTimeSeries struct {
	mu sync.Mutex

	T0      int64
	curTime int64
	Stats   *ProtStats
}

func NewProtobufTimeSeries(t0 int64, options *Options) *ProtobufTimeSeries {

	ret := &ProtobufTimeSeries{
		T0:    t0,
		Stats: new(ProtStats),
	}
	ret.Stats.FullTimeResolution = options.HighTimeResolution
	return ret
}
func (s *ProtobufTimeSeries) Name() string {
	return PROTOBUF_NAME
}
func (s *ProtobufTimeSeries) HighResolution() bool {
	return s.Stats.FullTimeResolution
}

func (s *ProtobufTimeSeries) Count() int {
	return len(s.Stats.Stats)
}

func (s *ProtobufTimeSeries) UnmarshalBinary(data []byte) error {
	err := proto.Unmarshal(data, s.Stats)
	return err
}

func (s *ProtobufTimeSeries) MarshalBinary() ([]byte, error) {
	return proto.Marshal(s.Stats)
}

func (s *ProtobufTimeSeries) Bytes() []byte {
	d, _ := s.MarshalBinary()
	return d
}

func (s *ProtobufTimeSeries) Len() int {
	b, _ := s.MarshalBinary()
	return len(b)
}

func (s *ProtobufTimeSeries) Iter() (iter TimeSeriesIter, err error) {
	s.mu.Lock()
	d := make([]*ProtStat, len(s.Stats.Stats))
	copy(d, s.Stats.Stats)
	s.mu.Unlock()

	iter, err = NewProtobufIter(d)
	iter.(*ProtobufIter).fullResolution = s.Stats.FullTimeResolution
	return iter, err
}

func (s *ProtobufTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *ProtobufTimeSeries) LastTime() int64 {
	return s.curTime
}

// the t is the "time we want to add
func (s *ProtobufTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	use_t := t
	if !s.Stats.FullTimeResolution {
		ts, _ := splitNano(t)
		use_t = int64(ts)
	}
	// if the count is 1, then we only have "one" value that makes any sense .. the sum
	if count == 1 || sameFloatVals(min, max, first, last, sum) {
		tmp := &ProtStatSmall{
			Time: use_t,
			Val:  sum,
		}
		p_stat := &ProtStat{
			StatType:  false,
			SmallStat: tmp,
		}
		s.Stats.Stats = append(s.Stats.Stats, p_stat)

	} else {

		tmp := &ProtStatFull{
			Time:  use_t,
			Min:   min,
			Max:   max,
			First: first,
			Last:  last,
			Sum:   sum,
			Count: count,
		}
		p_stat := &ProtStat{
			StatType: true,
			Stat:     tmp,
		}
		s.Stats.Stats = append(s.Stats.Stats, p_stat)
	}
	if t > s.curTime {
		s.curTime = t
	}
	if t < s.T0 {
		s.T0 = t
	}
	return nil
}

func (s *ProtobufTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type ProtobufIter struct {
	Stats          []*ProtStat
	curIdx         int
	statLen        int
	curStat        *ProtStat
	fullResolution bool

	curTime int64

	count int64

	finished bool
	err      error
}

func NewProtobufIterFromBytes(data []byte) (iter TimeSeriesIter, err error) {
	stats := new(ProtStats)
	err = proto.Unmarshal(data, stats)
	if err != nil {
		return nil, err
	}
	iter, err = NewProtobufIter(stats.Stats)
	if err != nil {
		return nil, err
	}
	iter.(*ProtobufIter).fullResolution = stats.FullTimeResolution
	return iter, nil
}

func NewProtobufIter(stats []*ProtStat) (*ProtobufIter, error) {
	it := &ProtobufIter{
		Stats:   stats,
		curIdx:  0,
		statLen: len(stats),
	}
	return it, nil
}

func (it *ProtobufIter) Next() bool {
	if it.finished || it.curIdx >= it.statLen {
		return false
	}
	it.curStat = it.Stats[it.curIdx]
	it.curIdx++
	return true
}

func (it *ProtobufIter) Values() (int64, float64, float64, float64, float64, float64, int64) {

	if it.curStat.GetStatType() {
		t := it.curStat.GetStat().GetTime()
		if !it.fullResolution {
			t = combineSecNano(uint32(t), 0)
		}
		return t,
			float64(it.curStat.GetStat().GetMin()),
			float64(it.curStat.GetStat().GetMax()),
			float64(it.curStat.GetStat().GetFirst()),
			float64(it.curStat.GetStat().GetLast()),
			float64(it.curStat.GetStat().GetSum()),
			it.curStat.GetStat().GetCount()
	}

	v := float64(it.curStat.GetSmallStat().GetVal())
	t := it.curStat.GetSmallStat().GetTime()
	if !it.fullResolution {
		t = combineSecNano(uint32(t), 0)
	}
	return t,
		v,
		v,
		v,
		v,
		v,
		1
}

func (it *ProtobufIter) ReprValue() *repr.StatRepr {
	if it.curStat.GetStatType() {
		var t time.Time
		if it.fullResolution {
			t = time.Unix(0, it.curStat.GetStat().GetTime())
		} else {
			t = time.Unix(it.curStat.GetStat().GetTime(), 0)
		}
		return &repr.StatRepr{
			Time:  t,
			Min:   repr.JsonFloat64(it.curStat.GetStat().GetMin()),
			Max:   repr.JsonFloat64(it.curStat.GetStat().GetMax()),
			Last:  repr.JsonFloat64(it.curStat.GetStat().GetLast()),
			First: repr.JsonFloat64(it.curStat.GetStat().GetFirst()),
			Sum:   repr.JsonFloat64(it.curStat.GetStat().GetSum()),
			Count: it.curStat.GetStat().GetCount(),
		}
	}
	v := repr.JsonFloat64(it.curStat.GetSmallStat().GetVal())
	var t time.Time
	if it.fullResolution {
		t = time.Unix(0, it.curStat.GetSmallStat().GetTime())
	} else {
		t = time.Unix(it.curStat.GetSmallStat().GetTime(), 0)
	}
	return &repr.StatRepr{
		Time:  t,
		Min:   v,
		Max:   v,
		Last:  v,
		First: v,
		Sum:   v,
		Count: 1,
	}
}

func (it *ProtobufIter) Error() error {
	return it.err
}
