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
	Same as the Simple Bin array, but using a DEFALTE compressor as the
	"buffer" to write to

	Flate turns out to give the same compression as "zip" but is ~30% faster

*/

package series

import (
	"bytes"
	"cadent/server/repr"
	"compress/flate"
	"encoding/gob"
	"io"
	"math"
	"sync"
)

const (
	ZIP_SIMPLE_BIN_SERIES_TAG       = "zbts" // just a flag to note we are using this one at the begining of each blob
	ZIP_SIMPLE_BIN_SERIES_LOWRE_TAG = "zbtl" // just a flag to note we are using this one at the begining of each blob
	ZIPGOB_NAME                     = "zipgob"
)

// this can only handle "future pushing times" not random times
type ZipGobTimeSeries struct {
	mu   *sync.Mutex
	sTag string

	T0       int64
	curCount int

	curDelta      int64
	curTime       int64
	curCountDelta int64
	curVals       []float64
	curValDeltas  []uint64
	curValCount   int64

	fullResolution bool

	buf     *gobBuffer
	zip     *flate.Writer
	encoder *gob.Encoder
}

func NewZipGobTimeSeries(t0 int64, options *Options) *ZipGobTimeSeries {
	ret := &ZipGobTimeSeries{
		T0:             t0,
		sTag:           ZIP_SIMPLE_BIN_SERIES_TAG,
		curTime:        0,
		curDelta:       0,
		curCount:       0,
		curVals:        make([]float64, 4),
		curValDeltas:   make([]uint64, 4),
		mu:             new(sync.Mutex),
		fullResolution: options.HighTimeResolution,
		buf:            new(gobBuffer),
	}
	if !ret.fullResolution {
		ret.sTag = ZIP_SIMPLE_BIN_SERIES_LOWRE_TAG
		ts, _ := splitNano(t0)
		ret.T0 = int64(ts)
	}
	ret.zip, _ = flate.NewWriter(ret.buf, flate.BestSpeed)
	ret.encoder = gob.NewEncoder(ret.zip)
	ret.writeHeader()
	return ret
}

func (s *ZipGobTimeSeries) Name() string {
	return ZIPGOB_NAME
}

func (s *ZipGobTimeSeries) HighResolution() bool {
	return s.fullResolution
}
func (s *ZipGobTimeSeries) Count() int {
	return s.curCount
}

func (s *ZipGobTimeSeries) writeHeader() {
	// tag it

	s.encoder.Encode(s.sTag)
	// need the start time
	s.encoder.Encode(s.T0)
}

func (s *ZipGobTimeSeries) UnmarshalBinary(data []byte) error {
	n_buf := new(gobBuffer)
	n_buf.data = data
	s.buf = n_buf
	return nil
}

func (s *ZipGobTimeSeries) Bytes() []byte {
	s.zip.Flush()
	byts := s.buf.Bytes()
	s.buf.Reset()
	// need to "readd" the bits to the current buffer as Bytes
	// wipes out the read pointer
	s.buf.Write(byts)
	return byts
}

func (s *ZipGobTimeSeries) MarshalBinary() ([]byte, error) {
	s.zip.Flush()
	return s.buf.Bytes(), nil
}

func (s *ZipGobTimeSeries) Len() int {
	s.zip.Flush()
	return s.buf.Len()
}

func (s *ZipGobTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *ZipGobTimeSeries) LastTime() int64 {
	return s.curTime
}

func (s *ZipGobTimeSeries) Iter() (TimeSeriesIter, error) {
	return NewZipGobIter(bytes.NewBuffer(s.Bytes()))
}

func (s *ZipGobTimeSeries) Copy() TimeSeries {
	g := *s
	g.mu = new(sync.Mutex)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.zip.Flush()
	g.buf = new(gobBuffer)
	g.buf.Write(s.buf.Clone())

	return &g
}

// the t is the "time we want to add
func (s *ZipGobTimeSeries) AddPoint(t int64, min float64, max float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	use_t := t
	if !s.fullResolution {
		tt, _ := splitNano(t)
		use_t = int64(tt)
	}

	if s.curTime == 0 {
		s.curDelta = use_t - s.T0
		s.curVals[0] = min
		s.curVals[1] = max
		s.curVals[2] = last
		s.curVals[3] = sum
		s.curValCount = count
	} else {
		s.curDelta = use_t - s.curTime
		s.curValDeltas[0] = math.Float64bits(s.curVals[0]) ^ math.Float64bits(min)
		s.curVals[0] = min

		s.curValDeltas[1] = math.Float64bits(s.curVals[1]) ^ math.Float64bits(max)
		s.curVals[1] = max

		s.curValDeltas[2] = math.Float64bits(s.curVals[2]) ^ math.Float64bits(last)
		s.curVals[2] = last

		s.curValDeltas[3] = math.Float64bits(s.curVals[3]) ^ math.Float64bits(sum)
		s.curVals[3] = sum

		s.curCountDelta = count - s.curValCount
	}

	s.encoder.Encode(s.curDelta)
	if count == 1 || sameFloatVals(min, max, last, sum) {
		s.encoder.Encode(false)
		if s.curTime == 0 {
			s.encoder.Encode(s.curVals[3]) // just the sum
		} else {
			s.encoder.Encode(s.curValDeltas[3]) // just the sum
		}
	} else {
		s.encoder.Encode(true)
		if s.curTime == 0 {
			s.encoder.Encode(s.curVals[0])
			s.encoder.Encode(s.curVals[1])
			s.encoder.Encode(s.curVals[2])
			s.encoder.Encode(s.curVals[3])
			s.encoder.Encode(s.curValCount)

		} else {
			s.encoder.Encode(s.curValDeltas[0])
			s.encoder.Encode(s.curValDeltas[1])
			s.encoder.Encode(s.curValDeltas[2])
			s.encoder.Encode(s.curValDeltas[3])
			s.encoder.Encode(s.curCountDelta)
		}
	}
	s.curTime = use_t
	s.curValCount = count

	s.curCount += 1
	return nil
}

func (s *ZipGobTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.Last), float64(stat.Sum), stat.Count)
}

///// ITERATOR
func NewZipGobIter(buf *bytes.Buffer) (TimeSeriesIter, error) {
	// need to defalte it
	reader := flate.NewReader(buf)

	out_buffer := &bytes.Buffer{}

	io.Copy(out_buffer, reader)
	reader.Close()

	return NewGobIter(out_buffer, ZIP_SIMPLE_BIN_SERIES_TAG)
}

func NewZipGobIterFromBytes(data []byte) (TimeSeriesIter, error) {
	// need to defalte it
	reader := flate.NewReader(bytes.NewBuffer(data))

	out_buffer := &bytes.Buffer{}

	io.Copy(out_buffer, reader)
	reader.Close()

	return NewGobIter(out_buffer, ZIP_SIMPLE_BIN_SERIES_TAG)
}
