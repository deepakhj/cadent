/*
	The Metric Msgpack Blob https://github.com/tinylib/msgp
*/

package series

import (
	"bytes"
	"cadent/server/repr"
	"cadent/server/writers/series/msgpacker"
	"fmt"
	"github.com/tinylib/msgp/msgp"
	"io"
	"sync"
	"time"
)

const (
	MSGPACK_SERIES_TAG_LOWRES  = "mspl"
	MSGPACK_SERIES_TAG_HIGHRES = "msph"
)

type MsgpackTimeSeries struct {
	mu sync.Mutex

	T0      int64
	curTime int64
	fullRes bool
	ct      int
	buf     *bytes.Buffer
	writer  *msgp.Writer
}

func NewMsgpackTimeSeries(t0 int64, options *Options) *MsgpackTimeSeries {

	ret := &MsgpackTimeSeries{
		T0:  t0,
		ct:  0,
		buf: new(bytes.Buffer),
	}
	ret.writer = msgp.NewWriter(ret.buf)

	t_head := MSGPACK_SERIES_TAG_LOWRES
	if options.HighTimeResolution {
		t_head = MSGPACK_SERIES_TAG_HIGHRES
	}
	ret.fullRes = options.HighTimeResolution
	// encode the header flag
	ret.buf.Write([]byte(t_head))

	return ret
}

func (s *MsgpackTimeSeries) HighResolution() bool {
	return s.fullRes
}

func (s *MsgpackTimeSeries) Count() int {
	return s.ct
}

func (s *MsgpackTimeSeries) UnmarshalBinary(data []byte) error {
	s.buf = bytes.NewBuffer(data)
	return nil
}

func (s *MsgpackTimeSeries) MarshalBinary() ([]byte, error) {
	return s.buf.Bytes(), nil
}

func (s *MsgpackTimeSeries) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writer.Flush()
	return s.buf.Bytes()
}

func (s *MsgpackTimeSeries) Len() int {
	return s.buf.Len()
}

func (s *MsgpackTimeSeries) Iter() (TimeSeriesIter, error) {
	return NewMsgpackIterFromBytes(s.Bytes())
}

func (s *MsgpackTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *MsgpackTimeSeries) LastTime() int64 {
	return s.curTime
}

// the t is the "time we want to add
func (s *MsgpackTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	use_t := t
	if !s.fullRes {
		ts, _ := splitNano(t)
		use_t = int64(ts)
	}
	// if the count is 1, then we only need "one" value that makes any sense .. the sum
	if count == 1 {
		tmp := &msgpacker.StatSmall{
			Time: use_t,
			Val:  sum,
		}
		p_stat := &msgpacker.Stat{
			StatType:  false,
			SmallStat: tmp,
		}
		//err = msgp.Encode(s.buf, p_stat)
		err = p_stat.EncodeMsg(s.writer)
		if err != nil {
			return err
		}
		s.ct++

	} else {

		tmp := &msgpacker.FullStat{
			Time:  use_t,
			Min:   min,
			Max:   max,
			First: first,
			Last:  last,
			Sum:   sum,
			Count: count,
		}
		p_stat := &msgpacker.Stat{
			StatType: true,
			Stat:     tmp,
		}
		//err = msgp.Encode(s.buf, p_stat)
		err = p_stat.EncodeMsg(s.writer)

		if err != nil {
			return err
		}
		s.ct++
	}
	if t > s.curTime {
		s.curTime = t
	}
	if t < s.T0 {
		s.T0 = t
	}
	return nil
}

func (s *MsgpackTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type MsgpackIter struct {
	buf            *bytes.Reader
	reader         *msgp.Reader
	curIdx         int
	curStat        *msgpacker.Stat
	fullResolution bool

	finished bool
	err      error
}

func NewMsgpackIterFromBytes(data []byte) (TimeSeriesIter, error) {

	it := new(MsgpackIter)
	it.curStat = new(msgpacker.Stat)

	// grab the header item
	it.buf = bytes.NewReader(data)
	//fmt.Printf("NEXT: %v %n\n\n", it.buf.Len())
	t_head := make([]byte, 4)
	n, err := it.buf.Read(t_head)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, fmt.Errorf("Msgpack: Invalid Header")
	}

	//fmt.Printf("NEXT: %v %n\n\n", it.buf.Len())

	switch string(t_head) {
	case MSGPACK_SERIES_TAG_LOWRES:
		it.fullResolution = false
	case MSGPACK_SERIES_TAG_HIGHRES:
		it.fullResolution = true
	default:
		return nil, fmt.Errorf("Msgpack: Invalid Header %s", t_head)
	}

	it.reader = msgp.NewReader(it.buf)

	return it, nil
}

func (it *MsgpackIter) Next() bool {
	if it.finished {
		return false
	}
	//it.curStat = msgpacker.Stat
	// decode a stat until there are no more
	//err := msgp.Decode(it.buf, it.curStat)
	err := it.curStat.DecodeMsg(it.reader)
	//fmt.Printf("NEXT: %v %s", err, it.buf.Len())
	// we are done
	if err == io.EOF {
		it.finished = true
		return false
	}
	if err != nil {
		it.err = err
		return false
	}
	it.curIdx++
	return true
}

func (it *MsgpackIter) Values() (int64, float64, float64, float64, float64, float64, int64) {

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

func (it *MsgpackIter) ReprValue() *repr.StatRepr {
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

func (it *MsgpackIter) Error() error {
	return it.err
}
