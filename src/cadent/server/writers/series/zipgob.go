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
	"sync"
)

const (
	ZIP_SIMPLE_BIN_SERIES_TAG       = "zbts" // just a flag to note we are using this one at the begining of each blob
	ZIP_SIMPLE_BIN_SERIES_LOWRE_TAG = "zbtl" // just a flag to note we are using this one at the begining of each blob
)

// this can only handle "future pushing times" not random times
type ZipGobTimeSeries struct {
	mu   sync.Mutex
	sTag string

	T0       int64
	curCount int

	curDelta       int64
	curTime        int64
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

// the t is the "time we want to add
func (s *ZipGobTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error {
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
	s.mu.Lock()
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
	s.mu.Unlock()
	s.curCount++
	return nil
}

func (s *ZipGobTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
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
