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
	"io"
	"sync"
	"time"
)

const (
	SIMPLE_BIN_SERIES_TAG        = "gobn" // just a flag to note we are using this one at the begining of each blob
	SIMPLE_BIN_SERIES_TAG_LOWRES = "gobl" // just a flag to note we are using this one at the begining of each blob
	GOB_NAME                     = "gob"
)

// from
// https://github.com/golang/go/blob/0da4dbe2322eb3b6224df35ce3e9fc83f104762b/src/encoding/gob/encode.go
// encBuffer is an extremely simple, fast implementation of a write-only byte buffer.
// It never returns a non-nil error, but Write returns an error value so it matches io.Writer.
type gobBuffer struct {
	data []byte
}

func (e *gobBuffer) WriteByte(c byte) (err error) {
	e.data = append(e.data, c)
	return
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

	T0             int64
	fullResolution bool // true for nanosecond, false for just second
	sTag           string

	curDelta int64
	curTime  int64

	buf      *gobBuffer
	encoder  *gob.Encoder
	curCount int
}

func NewGobTimeSeries(t0 int64, options *Options) *GobTimeSeries {
	ret := &GobTimeSeries{
		T0:             t0,
		fullResolution: options.HighTimeResolution,
		curTime:        0,
		curDelta:       0,
		curCount:       0,
		sTag:           SIMPLE_BIN_SERIES_TAG,
		buf:            new(gobBuffer),
	}

	if !ret.fullResolution {
		ts, _ := splitNano(t0)
		ret.T0 = int64(ts)
		ret.sTag = SIMPLE_BIN_SERIES_TAG_LOWRES
	}

	ret.encoder = gob.NewEncoder(ret.buf)
	ret.writeHeader()
	return ret
}

func (s *GobTimeSeries) Name() string {
	return GOB_NAME
}
func (s *GobTimeSeries) writeHeader() {
	// tag it
	s.encoder.Encode(s.sTag)

	// need the start time
	// fullrez gets full int64, otherwise just int32 for second
	// the encoder does the job of squeezeing it
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
	return NewGobIter(bytes.NewBuffer(byts), s.sTag)
}

func (s *GobTimeSeries) HighResolution() bool {
	return s.fullResolution
}

// the t is the "time we want to add
func (s *GobTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	use_t := t
	if !s.fullResolution {
		tt, _ := splitNano(t)
		use_t = int64(tt)
	}

	if s.curTime == 0 {
		s.curDelta = use_t - s.T0
	} else {
		s.curDelta = use_t - s.curTime
	}

	s.curTime = use_t
	s.encoder.Encode(s.curDelta)
	if count == 1 || sameFloatVals(min, max, first, last, sum) {
		s.encoder.Encode(false)
		s.encoder.Encode(sum) // just the sum
	} else {
		s.encoder.Encode(true)
		s.encoder.Encode(min)
		s.encoder.Encode(max)
		s.encoder.Encode(first)
		s.encoder.Encode(last)
		s.encoder.Encode(sum)
		s.encoder.Encode(count)
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
	T0             int64
	curTime        int64
	fullResolution bool // true for nanosecond, false for just second

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
	if st != SIMPLE_BIN_SERIES_TAG_LOWRES && st != ZIP_SIMPLE_BIN_SERIES_LOWRE_TAG {
		it.fullResolution = true
	}

	// need to pull the start time
	err = it.decoder.Decode(&it.T0)
	return it, err
}

func NewGobIterFromBytes(data []byte) (*GobIter, error) {
	return NewGobIter(bytes.NewBuffer(data), "")
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
	t := it.curTime
	if !it.fullResolution {
		t = combineSecNano(uint32(it.curTime), 0)
	}
	return t, it.min, it.max, it.first, it.last, it.sum, it.count
}

func (it *GobIter) ReprValue() *repr.StatRepr {
	var t time.Time
	if it.fullResolution {
		t = time.Unix(0, it.curTime)
	} else {
		t = time.Unix(it.curTime, 0)
	}
	return &repr.StatRepr{
		Time:  t,
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
