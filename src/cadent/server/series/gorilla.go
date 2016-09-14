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
	apadapted and manipulated from the original algo on https://github.com/dgryski/go-tsz

	but modified to handle "multiple" values the original is simply
	T,V
	but we need T,V,V,V,V ...

	we could do a multi array of T,V, T,V ... but why store time that many times?

	https://github.com/dgryski/go-tsz also uses uint32 for the base time.

	Since the Nano-second precision is nice, but "very" variable.  Meaning the delta-of-deltas
	can be very large for given point (basically base-time of (int32) appended to sub-second(int32)
	we break up the time "compression" in to 2 chunks .. the first "epoch" time and then the subsecond part.

	The highly variable part is the sub-second one, and almost always going to be not-so-compressible

	Basically we need to take an int like 1467946279766433748 and split it into 2 values

	1467946279 and 766433748

	If our resolution dump is in the "second" range (i.e. 99% of cases)
	Most of the time the deltas of the 2nd half will be "0" so we only really need to store one bit "0"

	we use the golang time module to do the splitting and re-combo, as well, it's good at it

	Since the "nanosecond" part of the time stamps is highly fluctuating, and not going to be "in order"
	(the deltas will be negative many times)
	we treat that part like a "value" as apposed to a "timestamp" and compress it like a normal float64

	Format ..

	there are "2" main time modes Full resolution for "nanoseconds"

	[4 byte header][1byte numvals][32 bit T0][32 bit Tms]
	[DelTs][XorTms0][v0][v1][...]
	[NumBit][DoDTs][XorTms0][XorV0][XorV1][...]
	...

	"Second" resolution

	[4 byte header][1byte numvals][32 bit T0]
	[DelTs][v0][v1][...]
	[NumBit][DoDTs][XorV0][XorV1][...]


	There is a third mode for "smart" encoding, much like the Gob/Protobuf formats
	if the count == 1 (or all the values are the same) we only encode the time + sum
	This requires an extra "bit" per Value/Time pairs as we need to know how much
	to encode/decode per item.  This is the default behavior for "full resolution" types

	[header things]
	[DelTs][small|fullbit][v0][v1][...]
	[NumBit][DoDTs][small|fullbit][XorV0][XorV1][...]

	the "end" of the stream is
	[0x04][0xffffff][0]
*/

package series

import (
	"cadent/server/repr"
	"cadent/server/utils"
	"errors"
	"fmt"
	"github.com/dgryski/go-bits"
	"math"
	"sync"
	"time"
)

const (
	GORILLA_BIN_SERIES_TAG_NANOSECOND = "gorn" // just a flag to note we are using this one at the start of each blob
	GORILLA_BIN_SERIES_TAG_SECOND     = "gors"
	GORILLA_BIN_SERIES_TAG_NANO_SMART = "gort"
	GORILLA_NAME                      = "gorilla"
)

var errNegativeTimeDelta = errors.New("Gorilla format cannot have out-of-order time points")

// this can only handle "future pushing times" not random times
type GorillaTimeSeries struct {
	mu *sync.Mutex

	curCount       int
	fullResolution bool // true for nanosecond, false for just second
	smartEncoding  bool // use smart(er) encoding

	Ts  uint32
	Tms uint32

	curTime  uint32
	curDelta uint32

	curTimeMs  float64
	leadingMs  uint8
	trailingMs uint8

	numValues uint8
	curVals   []float64 //want 6 vals: min, max, sum, last, count
	leading   []uint8
	trailing  []uint8

	bw utils.BitStream
}

func NewGoriallaTimeSeries(t0 int64, options *Options) *GorillaTimeSeries {

	ts, tms := splitNano(t0)

	ret := &GorillaTimeSeries{
		Ts:             ts,
		mu:             new(sync.Mutex),
		Tms:            tms,
		fullResolution: options.HighTimeResolution,
		curDelta:       0,
		curTime:        0,
		curTimeMs:      0,
		trailingMs:     0,
		curCount:       0,
		smartEncoding:  true, // default to true
		numValues:      uint8(options.NumValues),
	}

	ret.writeHeader()
	return ret
}

func (s *GorillaTimeSeries) Name() string {
	return GORILLA_NAME
}
func (s *GorillaTimeSeries) writeHeader() {
	t := ^uint8(0)
	s.leadingMs = t
	s.trailingMs = 0
	s.leading = make([]uint8, s.numValues, s.numValues)
	s.trailing = make([]uint8, s.numValues, s.numValues)
	s.curVals = make([]float64, s.numValues, s.numValues)
	for i := uint8(0); i < s.numValues; i++ {
		s.leading[i] = t
		s.trailing[i] = 0
	}
	// block header
	if s.fullResolution {
		s.bw.WriteBytes([]byte(GORILLA_BIN_SERIES_TAG_NANO_SMART))
	} else {
		s.bw.WriteBytes([]byte(GORILLA_BIN_SERIES_TAG_SECOND))
	}

	//num values
	s.bw.WriteBits(uint64(s.numValues), 8)

	s.bw.WriteBits(uint64(s.Ts), 32)

	if s.fullResolution {
		s.bw.WriteBits(uint64(s.Tms), 32)
		s.curTimeMs = float64(s.Tms)
		//log.Printf("Write Tstart: %v : %v", ts, ret.curTimeMs)
	}
	//log.Printf("Start Byte Write: %v ", ret.bw.bitsWritten)

}

func setFinished(bw *utils.BitStream) {
	// write an end-of-stream record
	bw.WriteBits(0x0f, 4)
	bw.WriteBits(0xffffffff, 32)
	bw.WriteBit(utils.ZeroBit)
}

func (s *GorillaTimeSeries) HighResolution() bool {
	return s.fullResolution
}

func (s *GorillaTimeSeries) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	setFinished(&s.bw)
}

func (s *GorillaTimeSeries) Count() int {
	return s.curCount
}
func (s *GorillaTimeSeries) MarshalBinary() ([]byte, error) {
	return s.Bytes(), nil
}

func (s *GorillaTimeSeries) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	bb := s.bw.Clone()
	setFinished(bb)

	return bb.Bytes()
}

func (s *GorillaTimeSeries) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.bw.Bytes())
}

func (s *GorillaTimeSeries) StartTime() int64 {
	return combineSecNano(s.Ts, uint32(s.Tms))
}

func (s *GorillaTimeSeries) LastTime() int64 {
	return combineSecNano(s.curTime, uint32(s.curTimeMs))
}

func (s *GorillaTimeSeries) Iter() (TimeSeriesIter, error) {
	return NewGorillaIterFromBytes(s.Bytes())
}

func (s *GorillaTimeSeries) UnmarshalBinary(data []byte) error {
	s.bw.SetStream(data)
	return nil
}

func (s *GorillaTimeSeries) Copy() TimeSeries {
	g := *s
	g.mu = new(sync.Mutex)
	s.mu.Lock()
	defer s.mu.Unlock()
	g.bw = *s.bw.Clone()
	return &g
}

// compress a float64 based on the last value added
func (s *GorillaTimeSeries) compressValue(curV float64, newV float64, leading uint8, trailing uint8) (n_leading uint8, n_trailing uint8) {

	vDelta := math.Float64bits(newV) ^ math.Float64bits(curV)

	if vDelta == 0 {
		s.bw.WriteBit(utils.ZeroBit)
		return leading, trailing
	}

	s.bw.WriteBit(utils.OneBit)

	n_leading = uint8(bits.Clz(vDelta))
	n_trailing = uint8(bits.Ctz(vDelta))

	// clamp number of leading zeros to avoid overflow when encoding
	if n_leading >= 32 {
		n_leading = 31
	}

	// TODO(dgryski): check if it's 'cheaper' to reset the leading/trailing bits instead
	if leading != ^uint8(0) && n_leading >= leading && n_trailing >= trailing {
		s.bw.WriteBit(utils.ZeroBit)
		s.bw.WriteBits(vDelta>>trailing, 64-int(leading)-int(trailing))
		n_leading = leading
		n_trailing = trailing
	} else {

		s.bw.WriteBit(utils.OneBit)
		s.bw.WriteBits(uint64(n_leading), 5)

		// Note that if leading == trailing == 0, then sigbits == 64.  But that value doesn't actually fit into the 6 bits we have.
		// Luckily, we never need to encode 0 significant bits, since that would put us in the other case (vdelta == 0).
		// So instead we write out a 0 and adjust it back to 64 on unpacking.
		sigbits := 64 - n_leading - n_trailing
		s.bw.WriteBits(uint64(sigbits), 6)
		s.bw.WriteBits(vDelta>>n_trailing, int(sigbits))
	}

	return n_leading, n_trailing
}

func (s *GorillaTimeSeries) addValue(idx int, v float64, isfirst bool) {
	if isfirst {
		s.curVals[idx] = v
		s.bw.WriteBits(math.Float64bits(v), 64)
		return
	}
	s.leading[idx], s.trailing[idx] = s.compressValue(s.curVals[idx], v, s.leading[idx], s.trailing[idx])
	s.curVals[idx] = v
}

func (s *GorillaTimeSeries) AddTime(t int64) error {

	ut, utms := splitNano(t)
	if s.curTime == 0 {
		// first point
		s.curTime = ut
		s.curDelta = ut - s.Ts
		s.bw.WriteBits(uint64(s.curDelta), 14)

		// ns part needs to be a float, so compress against the "0th" ns time marker
		if s.fullResolution {
			s.leadingMs, s.trailingMs = s.compressValue(s.curTimeMs, float64(utms), s.leadingMs, s.trailingMs)
			s.curTimeMs = float64(utms)
		}
		//log.Printf("WT0: T0: %v, (%v:%v) : %v", s.curTime, uint32(s.curTimeMs), uint32(utms), s.curDelta)
		//log.Printf("ByteWRIte: %v ", s.bw.bitsWritten)
		return nil
	}

	tDelta := ut - s.curTime
	dod := int32(tDelta - s.curDelta)
	if tDelta < 0 {
		return errNegativeTimeDelta
	}

	switch {
	case dod == 0:
		s.bw.WriteBit(utils.ZeroBit)
	case -63 <= dod && dod <= 64:
		s.bw.WriteBits(0x02, 2) // '10'
		s.bw.WriteBits(uint64(dod), 7)
	case -255 <= dod && dod <= 256:
		s.bw.WriteBits(0x06, 3) // '110'
		s.bw.WriteBits(uint64(dod), 9)
	case -2047 <= dod && dod <= 2048:
		s.bw.WriteBits(0x0e, 4) // '1110'
		s.bw.WriteBits(uint64(dod), 12)
	default:
		s.bw.WriteBits(0x0f, 4) // '1111'
		s.bw.WriteBits(uint64(dod), 32)
	}
	s.curDelta = tDelta
	s.curTime = ut

	// quick exit
	if !s.fullResolution {
		return nil
	}

	// if second resolution, this will "0" most of the time for second resolutions
	// due to the nature of the this part, we need to compress it via the Float64 compressor
	// as the deltas can produce negatives here as well as be wildly fluctuating
	s.leadingMs, s.trailingMs = s.compressValue(s.curTimeMs, float64(utms), s.leadingMs, s.trailingMs)
	s.curTimeMs = float64(utms)
	//log.Printf("WTN: T0: %v, %v : %v", s.curTime, uint32(s.curTimeMs), s.curDelta)

	//log.Printf("Write Time: %d %d (%d, %d)", s.curTime, s.curTimeMs, s.curDelta, s.curDeltaMs)

	return nil
}

func (s *GorillaTimeSeries) AddPoint(t int64, min float64, max float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := s.curTime == 0
	err := s.AddTime(t)
	if err != nil {
		return err
	}
	// now for the smarty pants
	if s.smartEncoding {
		// we have to write a true/false bit for the value to see if "false" not small
		if count == 1 || sameFloatVals(min, max, last, sum) {
			s.bw.WriteBit(utils.ZeroBit)
			// use the sum 3th slot in case we get another
			s.addValue(3, sum, start)
			s.curCount++
			return nil
		} else {
			s.bw.WriteBit(utils.OneBit)
		}
	}

	switch s.numValues {
	case 5:
		s.addValue(0, min, start)
		s.addValue(1, max, start)
		s.addValue(2, last, start)
		s.addValue(3, sum, start)
		s.addValue(4, float64(count), start)
	case 4:
		s.addValue(0, min, start)
		s.addValue(1, max, start)
		s.addValue(2, last, start)
		s.addValue(3, sum, start)
	case 3:
		s.addValue(0, min, start)
		s.addValue(1, max, start)
		s.addValue(2, last, start)
	case 2:
		s.addValue(0, min, start)
		s.addValue(1, max, start)
	default:
		s.addValue(0, min, start)
	}
	//log.Printf("Bytes Write: %v", s.bw.bitsWritten)
	s.curCount++
	return nil
}

func (s *GorillaTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.Last), float64(stat.Sum), stat.Count)
}

type GorillaIter struct {
	Ts uint32

	fullResolution bool
	smartEncoding  bool

	curSmartEnc utils.Bit

	curTime   uint32
	curVals   []float64
	numValues uint8

	curTimeMs  float64
	leadingMs  uint8
	trailingMs uint8

	br       utils.BitStream
	leading  []uint8
	trailing []uint8

	start    bool
	finished bool

	tDelta   uint32
	tDeltaMs uint32
	err      error
}

func NewGorillaIterFromBStream(br *utils.BitStream) (*GorillaIter, error) {

	br.SetCount(8)

	// read the header
	// 4byte flag, 2 uint32s for TimeS and TimeMs
	head, err := br.ReadBytes(uint8(len(GORILLA_BIN_SERIES_TAG_NANOSECOND)))
	if err != nil {
		return nil, err
	}
	//log.Printf("Start Byte Read: %v ", br.bitsRead)
	hh := string(head)
	if hh != GORILLA_BIN_SERIES_TAG_NANOSECOND && hh != GORILLA_BIN_SERIES_TAG_SECOND && hh != GORILLA_BIN_SERIES_TAG_NANO_SMART {
		return nil, fmt.Errorf("Not a valid Gorilla Series")
	}

	//determine resolution
	fullrez := true
	if hh == GORILLA_BIN_SERIES_TAG_SECOND {
		fullrez = false
	}

	smartEncoding := true

	// read the numvals
	numvals, err := br.ReadBits(8)
	if err != nil {
		return nil, err
	}
	//log.Printf("Start Byte Read: %v ", br.bitsRead)

	t0, err := br.ReadBits(32)
	if err != nil {
		return nil, err
	}
	//log.Printf("Start Byte Read: %v ", br.bitsRead)

	ret := &GorillaIter{
		Ts:             uint32(t0),
		br:             *br,
		numValues:      uint8(numvals),
		fullResolution: fullrez,
		smartEncoding:  smartEncoding,
		start:          true,
	}

	// first nanosecond part stored as uint32
	if ret.fullResolution {

		v, err := ret.br.ReadBits(32)
		if err != nil {
			return nil, err
		}

		ret.curTimeMs = float64(v)
	}
	//log.Printf("ReadStart: %v : : %v", ret.Ts, ret.curTimeMs)
	//log.Printf("Start Byte Read: %v ", ret.br.bitsRead)
	ret.trailing = make([]uint8, ret.numValues, ret.numValues)
	ret.leading = make([]uint8, ret.numValues, ret.numValues)
	ret.curVals = make([]float64, ret.numValues, ret.numValues)
	return ret, nil

}

func NewGorillaIterFromBytes(b []byte) (*GorillaIter, error) {
	return NewGorillaIterFromBStream(utils.NewBReader(b))
}

func (it *GorillaIter) readTimeDelta() bool {
	if it.start {
		// read first t
		tDelta, err := it.br.ReadBits(14)
		if err != nil {
			it.err = err
			return false
		}
		it.tDelta = uint32(tDelta)
		it.curTime = it.Ts + it.tDelta

		ok := true

		// first val is compressed against the start Tms
		if it.fullResolution {
			// note: we already read the "main" bits (yes a bit cludgy, but the reading is not thread safe)
			it.start = false
			ok, it.curTimeMs, it.leadingMs, it.trailingMs = it.uncompressValue(it.curTimeMs, it.leadingMs, it.trailingMs)
			it.start = true
		}
		//log.Printf("RTT0 : %v, %v :: %v: %v", it.Ts,  it.curTime, it.tDelta, it.curTimeMs)
		//log.Printf("ByteRead: %v ", it.br.bitsRead)
		return ok
	}

	// read delta-of-delta
	var d byte
	for i := 0; i < 4; i++ {
		d <<= 1
		bit, err := it.br.ReadBit()

		if err != nil {
			it.err = err
			return false
		}
		if bit == utils.ZeroBit {
			break
		}
		d |= 1
	}

	var dod int32
	var sz uint
	switch d {
	case 0x00:
	// dod == 0
	case 0x02:
		sz = 7
	case 0x06:
		sz = 9
	case 0x0e:
		sz = 12
	case 0x0f:
		bits, err := it.br.ReadBits(32)
		if err != nil {
			it.err = err
			return false
		}
		// end of stream
		if bits == 0xffffffff {
			it.finished = true
			return false
		}

		dod = int32(bits)
	}

	if sz != 0 {
		bits, err := it.br.ReadBits(int(sz))
		if err != nil {
			it.err = err
			return false
		}
		if bits > (1 << (sz - 1)) {
			// or something
			bits = bits - (1 << sz)
		}
		dod = int32(bits)
	}

	tDelta := it.tDelta + uint32(dod)

	it.tDelta = tDelta
	it.curTime = it.curTime + it.tDelta

	//log.Printf("Read TIME: %v Delta: %v DoD: %v", it.curTime, it.tDelta, dod)
	// quick exit
	if !it.fullResolution {
		return true
	}

	// nano time is treated like a float64 value
	var ok bool
	ok, it.curTimeMs, it.leadingMs, it.trailingMs = it.uncompressValue(it.curTimeMs, it.leadingMs, it.trailingMs)
	//log.Printf("Read TT %v: %v : %v", it.curTime, it.curTimeMs, it.tDelta)
	return ok
}

func (it *GorillaIter) uncompressValue(curV float64, o_leading uint8, o_trailing uint8) (ok bool, v float64, leading uint8, trailing uint8) {
	if it.start {
		tbits, err := it.br.ReadBits(64)
		if err != nil {
			it.err = err
			return false, 0.0, o_leading, o_trailing
		}
		return true, math.Float64frombits(tbits), o_leading, o_trailing
	}

	// compressed float value
	bit, err := it.br.ReadBit()
	if err != nil {
		it.err = err
		return false, curV, o_leading, o_trailing
	}

	// no value change
	if bit == utils.ZeroBit {
		return true, curV, o_leading, o_trailing
	}

	bit, err = it.br.ReadBit()
	if err != nil {
		it.err = err
		return false, curV, o_leading, o_trailing
	}
	if bit == utils.ZeroBit {
		// reuse leading/trailing zero bits
		leading = o_leading
		trailing = o_trailing
	} else {
		bits, err := it.br.ReadBits(5)
		if err != nil {
			it.err = err
			return false, v, o_leading, o_trailing
		}
		leading = uint8(bits)

		bits, err = it.br.ReadBits(6)
		if err != nil {
			it.err = err
			return false, v, o_leading, o_trailing
		}
		mbits := uint8(bits)
		// 0 significant bits here means we overflowed and we actually need 64; see comment in encoder
		if mbits == 0 {
			mbits = 64
		}
		trailing = 64 - leading - mbits
	}

	nbits := int(64 - leading - trailing)
	bits, err := it.br.ReadBits(nbits)
	if err != nil {
		it.err = err
		return false, curV, leading, trailing
	}
	vbits := math.Float64bits(curV)
	vbits ^= (bits << trailing)
	return true, math.Float64frombits(vbits), leading, trailing
}

func (it *GorillaIter) readValue(idx uint8) bool {
	var ok bool
	ok, it.curVals[idx], it.leading[idx], it.trailing[idx] = it.uncompressValue(it.curVals[idx], it.leading[idx], it.trailing[idx])
	return ok
}

func (it *GorillaIter) Next() bool {

	if it.err != nil || it.finished {
		return false
	}

	ok := it.readTimeDelta()

	if !ok {
		return false
	}

	// if in the "smart" mode, need the full or small bit
	it.curSmartEnc = utils.OneBit
	var err error
	if it.smartEncoding {
		it.curSmartEnc, err = it.br.ReadBit()
		if err != nil {
			it.err = err
			return false
		}

	}
	if it.curSmartEnc == utils.ZeroBit {
		// the "4" is the sum index
		ok = it.readValue(3)
		if !ok {
			return false
		}
	} else {
		for i := uint8(0); i < it.numValues; i++ {
			ok = it.readValue(i)
			if !ok {
				return false
			}
		}
	}
	//log.Printf("Bytes Read: %v", it.br.bitsRead)

	it.start = false
	return true
}

func (it *GorillaIter) Values() (int64, float64, float64, float64, float64, int64) {
	//log.Printf("Values Time: %v : %v", it.curTime, it.curTimeMs)
	if it.curSmartEnc == utils.ZeroBit {
		v := it.curVals[3]
		return combineSecNano(it.curTime, uint32(it.curTimeMs)), v, v, v, v, 1
	}
	return combineSecNano(it.curTime, uint32(it.curTimeMs)), it.curVals[0], it.curVals[1], it.curVals[2], it.curVals[3], int64(it.curVals[4])
}

func (it *GorillaIter) ReprValue() *repr.StatRepr {
	if it.curSmartEnc == utils.ZeroBit {
		v := repr.JsonFloat64(it.curVals[3])
		return &repr.StatRepr{
			Time:  time.Unix(int64(it.curTime), int64(it.curTimeMs)),
			Min:   v,
			Max:   v,
			Last:  v,
			Sum:   v,
			Count: 1,
		}
	}
	return &repr.StatRepr{
		Time:  time.Unix(int64(it.curTime), int64(it.curTimeMs)),
		Min:   repr.JsonFloat64(it.curVals[0]),
		Max:   repr.JsonFloat64(it.curVals[1]),
		Last:  repr.JsonFloat64(it.curVals[2]),
		Sum:   repr.JsonFloat64(it.curVals[3]),
		Count: int64(it.curVals[4]),
	}
}

func (it *GorillaIter) Error() error {
	return it.err
}
