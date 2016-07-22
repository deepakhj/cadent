/*
	The Metric Blob Reader/Writer

	This simply is a binary buffer of

	deltaT, min, max, first, last, sum, count

	DeltaT is the time delta deltas from a start time

	(CurrentTime - LastDelta - StartTime)

	format is

	[tag][T0][fullbit, deltaT, min, max, first, last, sum, count],[ ....]

	if fullbit == false/0

	[tag][T0][fullbit, deltaT, sum],[ ....]


	the fullbit is basically used for "lowres" things (or all values are the same) and if the count of
	the incomeing stat is "1" only one value makes any sense

*/

package series

import (
	"bytes"
	"cadent/server/repr"
	"encoding/gob"
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	SIMPLE_BIN_SERIES_TAG = "gobn" // just a flag to note we are using this one at the begining of each blob
)

// from
// https://github.com/golang/go/blob/0da4dbe2322eb3b6224df35ce3e9fc83f104762b/src/encoding/gob/encode.go
// encBuffer is an extremely simple, fast implementation of a write-only byte buffer.
// It never returns a non-nil error, but Write returns an error value so it matches io.Writer.
type gobBuffer struct {
	data []byte
}

func (e *gobBuffer) WriteByte(c byte) {
	e.data = append(e.data, c)
}

func (e *gobBuffer) Write(p []byte) (int, error) {
	e.data = append(e.data, p...)
	return len(p), nil
}

func (e *gobBuffer) Len() int {
	return len(e.data)
}

func (e *gobBuffer) Bytes() []byte {
	return e.data
}

func (e *gobBuffer) Reset() {
	e.data = e.data[0:0]
}

// this can only handle "future pushing times" not random times
type GobTimeSeries struct {
	mu sync.Mutex

	T0 int64

	curDelta int64
	curTime  int64

	buf      *gobBuffer
	encoder  *gob.Encoder
	curBytes int
	curCount int
}

func NewGobTimeSeries(t0 int64, options *Options) *GobTimeSeries {
	ret := &GobTimeSeries{
		T0:       t0,
		curTime:  0,
		curDelta: 0,
		curBytes: 0,
		curCount: 0,
		buf:      new(gobBuffer),
	}
	ret.encoder = gob.NewEncoder(ret.buf)
	ret.writeHeader()
	return ret
}

func (s *GobTimeSeries) writeHeader() {
	// tag it
	s.encoder.Encode(SIMPLE_BIN_SERIES_TAG)
	// need the start time
	s.encoder.Encode(s.T0)
}

func (s *GobTimeSeries) UnmarshalBinary(data []byte) error {
	n_buf := new(gobBuffer)
	n_buf.data = data
	s.buf = n_buf
	return nil
}

func (s *GobTimeSeries) Bytes() []byte {
	s.mu.Lock()
	byts := s.buf.Bytes()
	s.mu.Unlock()
	return byts
}

func (s *GobTimeSeries) Count() int {
	return s.curCount
}

func (s *GobTimeSeries) MarshalBinary() ([]byte, error) {
	return s.Bytes(), nil
}

func (s *GobTimeSeries) Len() int {
	return s.buf.Len()
}

func (s *GobTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *GobTimeSeries) LastTime() int64 {
	return s.curTime
}

func (s *GobTimeSeries) Iter() (TimeSeriesIter, error) {
	byts := s.Bytes()
	return NewGobIter(bytes.NewBuffer(byts), SIMPLE_BIN_SERIES_TAG)
}

// the t is the "time we want to add
func (s *GobTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.curTime == 0 {
		s.curDelta = t - s.T0
	} else {
		s.curDelta = t - s.curTime
	}

	s.curTime = t
	s.encoder.Encode(s.curDelta)
	if count == 1 || sameFloatVals(min, max, first, last, sum) {
		s.encoder.Encode(false)
		s.encoder.Encode(sum) // just the sum
		s.curBytes += 64 + 1
	} else {
		s.encoder.Encode(true)
		s.encoder.Encode(min)
		s.encoder.Encode(max)
		s.encoder.Encode(first)
		s.encoder.Encode(last)
		s.encoder.Encode(sum)
		s.encoder.Encode(count)
		s.curBytes += 64*7 + 1
	}
	s.curCount += 1
	return nil
}

func (s *GobTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type GobIter struct {
	T0      int64
	curTime int64

	tDelta int64
	min    float64
	max    float64
	first  float64
	last   float64
	sum    float64
	count  int64

	decoder *gob.Decoder

	finished bool
	err      error
}

func NewGobIter(buf *bytes.Buffer, tag string) (*GobIter, error) {
	it := &GobIter{}
	it.decoder = gob.NewDecoder(buf)
	// pull the flag
	st := ""
	err := it.decoder.Decode(&st)
	if err != nil {
		return nil, err
	}
	if st != tag {
		return nil, fmt.Errorf("This is not a GobTimeSeries blob")
	}

	// need to pull the start time
	err = it.decoder.Decode(&it.T0)
	return it, err
}

func NewGobIterFromBytes(data []byte) (*GobIter, error) {
	return NewGobIter(bytes.NewBuffer(data), SIMPLE_BIN_SERIES_TAG)
}

func (it *GobIter) Next() bool {
	if it == nil || it.finished {
		return false
	}
	var err error
	var t_delta int64
	err = it.decoder.Decode(&t_delta)
	if err != nil {
		it.finished = true
		it.err = err
		return false
	}

	it.tDelta = it.tDelta + int64(t_delta)
	it.curTime = it.T0 + it.tDelta

	//log.Printf("Delta Read: %d: %d: %d", t_delta, it.tDelta, it.curTime)

	// check the full/small bit
	var f_bit bool
	err = it.decoder.Decode(&f_bit)
	if err != nil {
		it.finished = true
		it.err = err
		return false
	}

	// if the full.small bit is false, just one val to decode
	if !f_bit {
		var val float64
		err = it.decoder.Decode(&val)
		if err != nil {
			it.finished = true
			it.err = err
			return false
		}
		it.min = val
		it.max = val
		it.first = val
		it.last = val
		it.count = 1
		it.sum = val
		return true
	}

	err = it.decoder.Decode(&it.min)
	if err != nil {
		it.finished = true
		it.err = err
		return false
	}
	err = it.decoder.Decode(&it.max)
	if err != nil {
		it.finished = true
		it.err = err
		return false
	}
	err = it.decoder.Decode(&it.first)
	if err != nil {
		it.finished = true
		it.err = err
		return false
	}
	err = it.decoder.Decode(&it.last)
	if err != nil {
		it.finished = true
		it.err = err
		return false
	}
	err = it.decoder.Decode(&it.sum)
	if err != nil {
		it.finished = true
		it.err = err
		return false
	}
	err = it.decoder.Decode(&it.count)
	if err != nil {
		it.finished = true
		it.err = err
		return false
	}
	return true
}

func (it *GobIter) Values() (int64, float64, float64, float64, float64, float64, int64) {
	return it.curTime, it.min, it.max, it.first, it.last, it.sum, it.count
}

func (it *GobIter) ReprValue() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  time.Unix(0, it.curTime),
		Min:   repr.JsonFloat64(it.min),
		Max:   repr.JsonFloat64(it.max),
		Last:  repr.JsonFloat64(it.last),
		First: repr.JsonFloat64(it.first),
		Sum:   repr.JsonFloat64(it.sum),
		Count: it.count,
	}
}

func (it *GobIter) Error() error {
	// skip ioEOFs as that's ok means we're done
	if it.err == io.EOF {
		return nil
	}
	return it.err
}
