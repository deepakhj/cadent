/*
	The Metric Codec Blob http://github.com/ugorji/go/codec
*/

package series

import (
	"bytes"
	"cadent/server/repr"
	"cadent/server/writers/series/codec"
	"fmt"
	cdec "github.com/ugorji/go/codec"
	"io"
	"sync"
	"time"
)

const (
	CODEC_BINC_SERIES_TAG_HIGHRES = "cdbh"
	CODEC_BINC_SERIES_TAG_LOWRES  = "cdbl"
	CODEC_CBOR_SERIES_TAG_HIGHRES = "cdch"
	CODEC_CBOR_SERIES_TAG_LOWRES  = "cdcl"
	CODEC_MSGP_SERIES_TAG_HIGHRES = "cdmh"
	CODEC_MSGP_SERIES_TAG_LOWRES  = "cdml"
	CODEC_JSON_SERIES_TAG_HIGHRES = "cdjh"
	CODEC_JSON_SERIES_TAG_LOWRES  = "cdjl"
)

type CodecTimeSeries struct {
	mu sync.Mutex

	T0      int64
	curTime int64
	fullRes bool
	handle  cdec.Handle
	ct      int
	buf     *bytes.Buffer
	enc     *cdec.Encoder
}

func NewCodecTimeSeries(t0 int64, options *Options) *CodecTimeSeries {

	ret := &CodecTimeSeries{
		T0:  t0,
		ct:  0,
		buf: new(bytes.Buffer),
	}

	t_head := ""
	switch options.Handler {
	case "binc":
		ret.handle = new(cdec.BincHandle)
		t_head = "cdb"
	case "cbor":
		ret.handle = new(cdec.CborHandle)
		t_head = "cdc"
	case "json":
		ret.handle = new(cdec.JsonHandle)
		t_head = "cdj"
	default:
		ret.handle = new(cdec.MsgpackHandle)
		t_head = "cdm"
	}
	ret.enc = cdec.NewEncoder(ret.buf, ret.handle)
	ret.fullRes = options.HighTimeResolution
	if options.HighTimeResolution {
		t_head += "h"
	} else {
		t_head += "l"
	}

	// encode the header flag
	ret.buf.Write([]byte(t_head))

	return ret
}

func (s *CodecTimeSeries) HighResolution() bool {
	return s.fullRes
}

func (s *CodecTimeSeries) Count() int {
	return s.ct
}

func (s *CodecTimeSeries) UnmarshalBinary(data []byte) error {
	s.buf = bytes.NewBuffer(data)
	return nil
}

func (s *CodecTimeSeries) MarshalBinary() ([]byte, error) {
	return s.buf.Bytes(), nil
}

func (s *CodecTimeSeries) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Bytes()
}

func (s *CodecTimeSeries) Len() int {
	return s.buf.Len()
}

func (s *CodecTimeSeries) Iter() (TimeSeriesIter, error) {
	return NewCodecIterFromBytes(s.Bytes())
}

func (s *CodecTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *CodecTimeSeries) LastTime() int64 {
	return s.curTime
}

// the t is the "time we want to add
func (s *CodecTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	use_t := t
	if !s.fullRes {
		ts, _ := splitNano(t)
		use_t = int64(ts)
	}
	// if the count is 1, then we only need "one" value that makes any sense .. the sum
	if count == 1 {
		tmp := &codec.StatSmall{
			Time: use_t,
			Val:  sum,
		}
		p_stat := &codec.Stat{
			StatType:  false,
			SmallStat: tmp,
		}
		err = s.enc.Encode(p_stat)
		if err != nil {
			return err
		}
		s.ct++

	} else {

		tmp := &codec.FullStat{
			Time:  use_t,
			Min:   min,
			Max:   max,
			First: first,
			Last:  last,
			Sum:   sum,
			Count: count,
		}
		p_stat := &codec.Stat{
			StatType: true,
			Stat:     tmp,
		}

		err = s.enc.Encode(p_stat)
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

func (s *CodecTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type CodecIter struct {
	handle cdec.Handle
	dec    *cdec.Decoder

	curIdx         int
	curStat        *codec.Stat
	fullResolution bool

	finished bool
	err      error
}

func NewCodecIterFromBytes(data []byte) (TimeSeriesIter, error) {
	it := new(CodecIter)
	it.curStat = new(codec.Stat)
	// grab the header item
	buf := bytes.NewReader(data)
	t_head := make([]byte, 4)
	n, err := buf.Read(t_head)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, fmt.Errorf("Codec: Invalid Header")
	}

	switch string(t_head) {
	case CODEC_BINC_SERIES_TAG_HIGHRES:
		it.handle = new(cdec.BincHandle)
		it.fullResolution = true
	case CODEC_BINC_SERIES_TAG_LOWRES:
		it.handle = new(cdec.BincHandle)
		it.fullResolution = false
	case CODEC_CBOR_SERIES_TAG_HIGHRES:
		it.handle = new(cdec.CborHandle)
		it.fullResolution = true
	case CODEC_CBOR_SERIES_TAG_LOWRES:
		it.handle = new(cdec.CborHandle)
		it.fullResolution = false
	case CODEC_JSON_SERIES_TAG_HIGHRES:
		it.handle = new(cdec.JsonHandle)
		it.fullResolution = true
	case CODEC_JSON_SERIES_TAG_LOWRES:
		it.handle = new(cdec.JsonHandle)
		it.fullResolution = false
	case CODEC_MSGP_SERIES_TAG_HIGHRES:
		it.handle = new(cdec.MsgpackHandle)
		it.fullResolution = true
	case CODEC_MSGP_SERIES_TAG_LOWRES:
		it.handle = new(cdec.MsgpackHandle)
		it.fullResolution = false
	default:
		return nil, fmt.Errorf("Codec: Invalid Header Codec %s", t_head)
	}

	it.dec = cdec.NewDecoder(buf, it.handle)

	return it, nil
}

func (it *CodecIter) Next() bool {
	if it.finished {
		return false
	}
	// decode a stat until there are no more
	err := it.dec.Decode(it.curStat)
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

func (it *CodecIter) Values() (int64, float64, float64, float64, float64, float64, int64) {

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

func (it *CodecIter) ReprValue() *repr.StatRepr {
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

func (it *CodecIter) Error() error {
	return it.err
}
