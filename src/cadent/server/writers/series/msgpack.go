/*
	The Metric MessagePack Blob (http://msgpack.org/)


*/

package series

import (
	"bytes"
	"cadent/server/repr"
	"cadent/server/writers/series/codec"
	msgpack "github.com/ugorji/go/codec"
	"sync"
	"time"
)

type MsgPackStats []jsonStat

type MsgPackTimeSeries struct {
	mu sync.Mutex

	T0      int64
	curTime int64
	Stats   *codec.CodecStats
}

func NewMsgPackTimeSeries(t0 int64, options *Options) *MsgPackTimeSeries {

	ret := &MsgPackTimeSeries{
		T0:    t0,
		Stats: new(codec.CodecStats),
	}
	ret.Stats.FullTimeResolution = options.HighTimeResolution
	return ret
}

func (s *MsgPackTimeSeries) HighResolution() bool {
	return s.Stats.FullTimeResolution
}

func (s *MsgPackTimeSeries) Count() int {
	return len(s.Stats.Stats)
}

func (s *MsgPackTimeSeries) UnmarshalBinary(data []byte) error {
	var h msgpack.Handle = new(msgpack.MsgpackHandle)
	var dec *msgpack.Decoder = msgpack.NewDecoderBytes(data, h)
	return dec.Decode(s.Stats)
}

func (s *MsgPackTimeSeries) MarshalBinary() ([]byte, error) {
	bw := new(bytes.Buffer)
	var h msgpack.Handle = new(msgpack.MsgpackHandle)
	var enc *msgpack.Encoder = msgpack.NewEncoder(bw, h)
	var err error = enc.Encode(s.Stats)
	if err != nil {
		return nil, err
	}
	return bw.Bytes(), nil
}

func (s *MsgPackTimeSeries) Bytes() []byte {
	d, _ := s.MarshalBinary()
	return d
}

func (s *MsgPackTimeSeries) Len() int {
	b, _ := s.MarshalBinary()
	return len(b)
}

func (s *MsgPackTimeSeries) Iter() (iter TimeSeriesIter, err error) {
	s.mu.Lock()
	d := make([]*codec.CodecStat, len(s.Stats.Stats))
	copy(d, s.Stats.Stats)
	s.mu.Unlock()

	iter, err = NewMsgPackIter(d)
	iter.(*MsgPackIter).fullResolution = s.Stats.FullTimeResolution
	return iter, err
}

func (s *MsgPackTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *MsgPackTimeSeries) LastTime() int64 {
	return s.curTime
}

// the t is the "time we want to add
func (s *MsgPackTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	use_t := t
	if !s.Stats.FullTimeResolution {
		ts, _ := splitNano(t)
		use_t = int64(ts)
	}
	// if the count is 1, then we only have "one" value that makes any sense .. the sum
	if count == 1 || sameFloatVals(min, max, first, last, sum) {
		tmp := &codec.CodecStatSmall{
			Time: use_t,
			Val:  sum,
		}
		p_stat := &codec.CodecStat{
			StatType:  false,
			SmallStat: tmp,
		}
		s.Stats.Stats = append(s.Stats.Stats, p_stat)

	} else {

		tmp := &codec.CodecFullStat{
			Time:  use_t,
			Min:   min,
			Max:   max,
			First: first,
			Last:  last,
			Sum:   sum,
			Count: count,
		}
		p_stat := &codec.CodecStat{
			StatType: true,
			Stat:     tmp,
		}
		s.Stats.Stats = append(s.Stats.Stats, p_stat)
	}
	s.curTime = t
	return nil
}

func (s *MsgPackTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type MsgPackIter struct {
	Stats          []*codec.CodecStat
	curIdx         int
	statLen        int
	curStat        *codec.CodecStat
	fullResolution bool

	curTime int64

	count int64

	finished bool
	err      error
}

func NewMsgPackIterFromBytes(data []byte) (iter TimeSeriesIter, err error) {
	stats := new(codec.CodecStats)

	iter, err = NewMsgPackIter(stats.Stats)
	if err != nil {
		return nil, err
	}
	iter.(*MsgPackIter).fullResolution = stats.FullTimeResolution
	return iter, nil
}

func NewMsgPackIter(stats []*codec.CodecStat) (*MsgPackIter, error) {
	it := &MsgPackIter{
		Stats:   stats,
		curIdx:  0,
		statLen: len(stats),
	}
	return it, nil
}

func (it *MsgPackIter) Next() bool {
	if it.finished || it.curIdx >= it.statLen {
		return false
	}
	it.curStat = it.Stats[it.curIdx]
	it.curIdx++
	return true
}

func (it *MsgPackIter) Values() (int64, float64, float64, float64, float64, float64, int64) {

	if it.curStat.StatType {
		t := it.curStat.Stat.Time
		if !it.fullResolution {
			t = combineSecNano(uint32(t), 0)
		}
		return t,
			float64(it.curStat.Stat.Min),
			float64(it.curStat.Stat.Max),
			float64(it.curStat.Stat.First),
			float64(it.curStat.Stat.Last),
			float64(it.curStat.Stat.Sum),
			it.curStat.Stat.Count
	}

	v := float64(it.curStat.SmallStat.Val)
	t := it.curStat.SmallStat.Time
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

func (it *MsgPackIter) ReprValue() *repr.StatRepr {
	if it.curStat.StatType {
		var t time.Time
		if it.fullResolution {
			t = time.Unix(0, it.curStat.Stat.Time)
		} else {
			t = time.Unix(it.curStat.Stat.Time, 0)
		}
		return &repr.StatRepr{
			Time:  t,
			Min:   repr.JsonFloat64(it.curStat.Stat.Min),
			Max:   repr.JsonFloat64(it.curStat.Stat.Max),
			Last:  repr.JsonFloat64(it.curStat.Stat.Last),
			First: repr.JsonFloat64(it.curStat.Stat.First),
			Sum:   repr.JsonFloat64(it.curStat.Stat.Sum),
			Count: it.curStat.Stat.Count,
		}
	}
	v := repr.JsonFloat64(it.curStat.SmallStat.Val)
	var t time.Time
	if it.fullResolution {
		t = time.Unix(0, it.curStat.SmallStat.Time)
	} else {
		t = time.Unix(it.curStat.SmallStat.Time, 0)
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

func (it *MsgPackIter) Error() error {
	return it.err
}
