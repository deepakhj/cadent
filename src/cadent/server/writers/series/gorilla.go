/*
	based on https://github.com/dgryski/go-tsz

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

	Format

	[4 byte header][BigEndian Ts uint32][BigEndian Tms uint32]
	[DelTs][DelTms][v0][v1][v2][v3][v4][v5]
	[NumBit][DoDTs][NumBit][DodTms][XorV0][XorV1][XorV2][XorV3][XorV4][XorV5]

*/

package series

import (
	"cadent/server/repr"
	"encoding/binary"
	"fmt"
	"github.com/dgryski/go-bits"
	"io"
	"log"
	"math"
	"sync"
	"time"
)

const (
	GORILLA_BIN_SERIES_TAG = "vgor" // just a flag to note we are using this one at the start of each blob
)

// make the "second" and "nanosecond" parts
func splitNano(t int64) (uint32, uint32) {
	// not "good way" of splitting a Nano-time is available so we need to
	// convert things to "time" and grab the resulting bits
	tt := time.Unix(0, t)
	return uint32(tt.Unix()), uint32(tt.Nanosecond())
}

// remake a "nano-time"
func combineSecNano(ts uint32, tns uint32) int64 {
	// not "good way" of splitting a Nano-time is available so we need to
	// convert things to "time" and grab the resulting bits
	tt := time.Unix(int64(ts), int64(tns))
	return tt.UnixNano()
}

/** shamelessly taken from https://github.com/dgryski/go-tsz/blob/master/bstream.go */

type bit bool

const (
	zero bit = false
	one  bit = true
)

// bstream is a stream of bits
type bstream struct {
	// the data stream
	stream []byte

	// how many bits are valid in current byte
	count uint8
}

func newBReader(b []byte) *bstream {
	return &bstream{stream: b, count: 8}
}

func newBWriter(size int) *bstream {
	return &bstream{stream: make([]byte, 0, size), count: 0}
}

func (b *bstream) clone() *bstream {
	d := make([]byte, len(b.stream))
	copy(d, b.stream)
	return &bstream{stream: d, count: b.count}
}

func (b *bstream) bytes() []byte {
	return b.stream
}

func (b *bstream) writeBit(bit bit) {

	if b.count == 0 {
		b.stream = append(b.stream, 0)
		b.count = 8
	}

	i := len(b.stream) - 1

	if bit {
		b.stream[i] |= 1 << (b.count - 1)
	}

	b.count--
}

func (b *bstream) writeBytes(bs []byte) int {
	c := 0
	for _, by := range bs {
		b.stream = append(b.stream, by)
		c++
	}
	return c
}

func (b *bstream) writeByte(byt byte) {

	if b.count == 0 {
		b.stream = append(b.stream, 0)
		b.count = 8
	}

	i := len(b.stream) - 1

	// fill up b.b with b.count bits from byt
	b.stream[i] |= byt >> (8 - b.count)

	b.stream = append(b.stream, 0)
	i++
	b.stream[i] = byt << b.count
}

func (b *bstream) writeBits(u uint64, nbits int) {
	u <<= (64 - uint(nbits))
	for nbits >= 8 {
		byt := byte(u >> 56)
		b.writeByte(byt)
		u <<= 8
		nbits -= 8
	}

	for nbits > 0 {
		b.writeBit((u >> 63) == 1)
		u <<= 1
		nbits--
	}
}

func (b *bstream) readBit() (bit, error) {

	if len(b.stream) == 0 {
		return false, io.EOF
	}

	if b.count == 0 {
		b.stream = b.stream[1:]
		// did we just run out of stuff to read?
		if len(b.stream) == 0 {
			return false, io.EOF
		}
		b.count = 8
	}

	b.count--
	d := b.stream[0] & 0x80
	b.stream[0] <<= 1
	return d != 0, nil
}

func (b *bstream) readBytes(n uint8) ([]byte, error) {

	if len(b.stream) == 0 {
		return nil, io.EOF
	}
	if len(b.stream) < int(n) {
		return nil, io.EOF
	}

	byt := b.stream[0:n]
	b.stream = b.stream[n:]

	return byt, nil

}

func (b *bstream) readByte() (byte, error) {

	if len(b.stream) == 0 {
		return 0, io.EOF
	}

	if b.count == 0 {
		b.stream = b.stream[1:]

		if len(b.stream) == 0 {
			return 0, io.EOF
		}

		b.count = 8
	}

	if b.count == 8 {
		b.count = 0
		return b.stream[0], nil
		//b.stream = b.stream[1:]
		//return byt, nil
	}

	byt := b.stream[0]
	b.stream = b.stream[1:]

	if len(b.stream) == 0 {
		return 0, io.EOF
	}

	byt |= b.stream[0] >> b.count
	b.stream[0] <<= (8 - b.count)

	return byt, nil
}

func (b *bstream) readBits(nbits int) (uint64, error) {

	var u uint64

	for nbits >= 8 {
		byt, err := b.readByte()
		if err != nil {
			return 0, err
		}

		u = (u << 8) | uint64(byt)
		nbits -= 8
	}

	var err error
	for nbits > 0 && err != io.EOF {
		byt, err := b.readBit()
		if err != nil {
			return 0, err
		}
		u <<= 1
		if byt {
			u |= 1
		}
		nbits--
	}

	return u, nil
}

// this can only handle "future pushing times" not random times
type MultiGoriallaTimeSeries struct {
	sync.Mutex

	Ts  uint32
	Tms uint32

	curTime    uint32
	curTimeMs  uint32
	curDelta   uint32
	curDeltaMs uint32

	curVals  [6]float64 //want 6 vals, min, max, sum, first, last, count
	leading  [6]uint8
	trailing [6]uint8

	bw       bstream
	finished bool
}

func NewMultiGoriallaTimeSeries(t0 int64) *MultiGoriallaTimeSeries {

	ts, tms := splitNano(t0)

	ret := &MultiGoriallaTimeSeries{
		Ts:         ts,
		Tms:        tms,
		curDelta:   0,
		curDeltaMs: 0,
		curTime:    0,
		curTimeMs:  0,
		finished:   false,
	}
	t := ^uint8(0)
	ret.leading = [6]uint8{t, t, t, t, t, t}
	ret.trailing = [6]uint8{0, 0, 0, 0, 0, 0}
	// block header
	ret.bw.writeBytes([]byte(GORILLA_BIN_SERIES_TAG))
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, ts)
	ret.bw.writeBytes(buf)
	binary.BigEndian.PutUint32(buf, tms)
	ret.bw.writeBytes(buf)

	return ret
}

func (s *MultiGoriallaTimeSeries) setFinished() {
	// write an end-of-stream record
	s.bw.writeBits(0x0f, 4)
	s.bw.writeBits(0xffffffff, 32)
	s.bw.writeBit(zero)
}

func (s *MultiGoriallaTimeSeries) Finish() {
	s.Lock()
	defer s.Unlock()
	if !s.finished {
		s.setFinished()
		s.finished = true
	}
}

func (s *MultiGoriallaTimeSeries) UnmarshalBinary(data []byte) error {
	s.bw.stream = data
	return nil
}

func (s *MultiGoriallaTimeSeries) addValue(idx int, v float64, isfirst bool) {
	if isfirst {
		s.curVals[idx] = v
		s.bw.writeBits(math.Float64bits(v), 64)
		return
	}

	val := s.curVals[idx]
	vDelta := math.Float64bits(v) ^ math.Float64bits(val)

	if vDelta == 0 {
		s.bw.writeBit(zero)
	} else {
		s.bw.writeBit(one)

		leading := uint8(bits.Clz(vDelta))
		trailing := uint8(bits.Ctz(vDelta))

		// clamp number of leading zeros to avoid overflow when encoding
		if leading >= 32 {
			leading = 31
		}

		// TODO(dgryski): check if it's 'cheaper' to reset the leading/trailing bits instead
		if s.leading[idx] != ^uint8(0) && leading >= s.leading[idx] && trailing >= s.trailing[idx] {
			s.bw.writeBit(zero)
			s.bw.writeBits(vDelta>>s.trailing[idx], 64-int(s.leading[idx])-int(s.trailing[idx]))
		} else {
			s.leading[idx], s.trailing[idx] = leading, trailing

			s.bw.writeBit(one)
			s.bw.writeBits(uint64(leading), 5)

			// Note that if leading == trailing == 0, then sigbits == 64.  But that value doesn't actually fit into the 6 bits we have.
			// Luckily, we never need to encode 0 significant bits, since that would put us in the other case (vdelta == 0).
			// So instead we write out a 0 and adjust it back to 64 on unpacking.
			sigbits := 64 - leading - trailing
			s.bw.writeBits(uint64(sigbits), 6)
			s.bw.writeBits(vDelta>>trailing, int(sigbits))
		}
	}
}

func (s *MultiGoriallaTimeSeries) AddTime(t int64) error {
	ut, utms := splitNano(t)
	if s.curTime == 0 {
		// first point
		s.curTime = ut
		s.curDelta = ut - s.Ts
		s.bw.writeBits(uint64(s.curDelta), 14)

		s.curTimeMs = utms
		s.curDeltaMs = utms - s.Tms
		s.bw.writeBits(uint64(s.curDeltaMs), 14)
		log.Printf("Start Time: T0: %d, Tms:%d %d %d (%d, %d)", s.Ts, s.Tms, s.curTime, s.curTimeMs, s.curDelta, s.curDeltaMs)
		return nil
	}

	tDelta := ut - s.curTime
	dod := int64(tDelta - s.curDelta)

	switch {
	case dod == 0:
		s.bw.writeBit(zero)
	case -63 <= dod && dod <= 64:
		s.bw.writeBits(0x02, 2) // '10'
		s.bw.writeBits(uint64(dod), 7)
	case -255 <= dod && dod <= 256:
		s.bw.writeBits(0x06, 3) // '110'
		s.bw.writeBits(uint64(dod), 9)
	case -2047 <= dod && dod <= 2048:
		s.bw.writeBits(0x0e, 4) // '1110'
		s.bw.writeBits(uint64(dod), 12)
	default:
		s.bw.writeBits(0x0f, 4) // '1111'
		s.bw.writeBits(uint64(dod), 32)
	}

	// if second resolution, this will "0" most of the time for second resolutions
	tDeltaMs := utms - s.curTimeMs
	dod = int64(tDeltaMs - s.curDeltaMs)

	switch {
	case dod == 0:
		s.bw.writeBit(zero)
	case -63 <= dod && dod <= 64:
		s.bw.writeBits(0x02, 2) // '10'
		s.bw.writeBits(uint64(dod), 7)
	case -255 <= dod && dod <= 256:
		s.bw.writeBits(0x06, 3) // '110'
		s.bw.writeBits(uint64(dod), 9)
	case -2047 <= dod && dod <= 2048:
		s.bw.writeBits(0x0e, 4) // '1110'
		s.bw.writeBits(uint64(dod), 12)
	default:
		s.bw.writeBits(0x0f, 4) // '1111'
		s.bw.writeBits(uint64(dod), 32)
	}

	s.curDelta = tDelta
	s.curTime = ut

	s.curDeltaMs = tDeltaMs
	s.curTimeMs = utms

	log.Printf("Write Time: %d %d (%d, %d)", s.curTime, s.curTimeMs, s.curDelta, s.curDeltaMs)

	return nil
}

// The main Gorialla Algo in here
func (s *MultiGoriallaTimeSeries) AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error {
	s.Lock()
	defer s.Unlock()

	if s.curTime == 0 {
		s.AddTime(t)
		s.addValue(0, min, true)
		s.addValue(1, max, true)
		s.addValue(2, first, true)
		s.addValue(3, last, true)
		s.addValue(4, sum, true)
		s.addValue(5, float64(count), true)
		return nil
	}
	s.AddTime(t)
	s.addValue(0, min, false)
	s.addValue(1, max, false)
	s.addValue(2, first, false)
	s.addValue(3, last, false)
	s.addValue(4, sum, false)
	s.addValue(5, float64(count), false)

	return nil
}

func (s *MultiGoriallaTimeSeries) AddStat(stat *repr.StatRepr) error {
	return s.AddPoint(stat.Time.UnixNano(), float64(stat.Min), float64(stat.Max), float64(stat.First), float64(stat.Last), float64(stat.Sum), stat.Count)
}

func (s *MultiGoriallaTimeSeries) MarshalBinary() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return s.bw.bytes(), nil
}

func (s *MultiGoriallaTimeSeries) Len() int {
	s.Lock()
	defer s.Unlock()
	return len(s.bw.bytes())
}

func (s *MultiGoriallaTimeSeries) StartTime() int64 {
	return combineSecNano(s.Ts, s.Tms)
}

func (s *MultiGoriallaTimeSeries) LastTime() int64 {
	return combineSecNano(s.curTime, s.curTimeMs)
}

func (s *MultiGoriallaTimeSeries) Iter() (TimeSeriesIter, error) {

	s.Lock()
	w := s.bw.clone()
	s.Unlock()

	//s.setFinished()

	iter, err := NewGorillaIterFromBStream(w)
	return iter, err
}

type GorillaIter struct {
	Ts  uint32
	Tms uint32

	curTime   uint32
	curTimeMs uint32
	curVals   [6]float64

	br       bstream
	leading  [6]uint8
	trailing [6]uint8

	start    bool
	finished bool

	tDelta   uint32
	tDeltaMs uint32
	err      error
}

func NewGorillaIterFromBStream(br *bstream) (*GorillaIter, error) {

	br.count = 8
	// read the header to make sure
	head, err := br.readBytes(uint8(len(GORILLA_BIN_SERIES_TAG)))
	if err != nil {
		return nil, err
	}
	if string(head) != GORILLA_BIN_SERIES_TAG {
		return nil, fmt.Errorf("Not a valid Gorilla Series")
	}

	buf, err := br.readBytes(4)
	t0 := binary.BigEndian.Uint32(buf)
	if err != nil {
		return nil, err
	}
	buf, err = br.readBytes(4)
	tms := binary.BigEndian.Uint32(buf)
	if err != nil {
		return nil, err
	}

	ret := &GorillaIter{
		Ts:  uint32(t0),
		Tms: uint32(tms),
		br:  *br,
	}
	ret.trailing = [6]uint8{0, 0, 0, 0, 0, 0}
	ret.leading = [6]uint8{0, 0, 0, 0, 0, 0}
	return ret, nil

}

func NewGorillaIter(b []byte) (*GorillaIter, error) {
	return NewGorillaIterFromBStream(newBReader(b))
}

func (it *GorillaIter) readTimeDelta() bool {
	if it.curTime == 0 {
		it.start = true
		// read first t
		tDelta, err := it.br.readBits(14)
		if err != nil {
			it.err = err
			return false
		}
		it.tDelta = uint32(tDelta)
		it.curTime = it.Ts + it.tDelta

		// read first t
		tDeltaMs, err := it.br.readBits(14)
		if err != nil {
			it.err = err
			return false
		}
		it.tDeltaMs = uint32(tDeltaMs)
		it.curTimeMs = it.Tms + it.tDeltaMs
		return true
	}

	// read delta-of-delta
	var d byte
	for i := 0; i < 4; i++ {
		d <<= 1
		bit, err := it.br.readBit()

		if err != nil {
			it.err = err
			return false
		}
		if bit == zero {
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
		bits, err := it.br.readBits(32)
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
		bits, err := it.br.readBits(int(sz))
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

	// nano second part
	// read delta-of-delta
	var dms byte
	for i := 0; i < 4; i++ {
		dms <<= 1
		bit, err := it.br.readBit()
		if err != nil {
			it.err = err
			return false
		}
		if bit == zero {
			break
		}
		dms |= 1
	}

	var dodms int32
	var szms uint
	switch dms {
	case 0x00:
	// dod == 0
	case 0x02:
		szms = 7
	case 0x06:
		szms = 9
	case 0x0e:
		szms = 12
	case 0x0f:
		bits, err := it.br.readBits(32)
		if err != nil {
			it.err = err
			return false
		}

		// end of stream
		if bits == 0xffffffff {
			it.finished = true
			return false
		}

		dodms = int32(bits)
	}

	if szms != 0 {
		bits, err := it.br.readBits(int(sz))
		if err != nil {
			it.err = err
			return false
		}
		if bits > (1 << (szms - 1)) {
			// or something
			bits = bits - (1 << sz)
		}
		dodms = int32(bits)
	}

	tDeltaMs := it.tDeltaMs + uint32(dodms)

	it.tDeltaMs = tDeltaMs
	it.curTimeMs = it.curTimeMs + it.tDeltaMs

	return true
}

func (it *GorillaIter) readValue(idx uint8) bool {

	if it.start {

		v, err := it.br.readBits(64)
		if err != nil {
			it.err = err
			return false
		}

		it.curVals[idx] = math.Float64frombits(v)
	}

	// read compressed value
	bit, err := it.br.readBit()
	if err != nil {
		it.err = err
		return false
	}

	// no value change
	if bit == zero {
		// it.val = it.val
		return true
	}

	bit, err = it.br.readBit()
	if err != nil {
		it.err = err
		return false
	}
	if bit == zero {
		// reuse leading/trailing zero bits
		// it.leading, it.trailing = it.leading, it.trailing
	} else {
		bits, err := it.br.readBits(5)
		if err != nil {
			it.err = err
			return false
		}
		it.leading[idx] = uint8(bits)

		bits, err = it.br.readBits(6)
		if err != nil {
			it.err = err
			return false
		}
		mbits := uint8(bits)
		// 0 significant bits here means we overflowed and we actually need 64; see comment in encoder
		if mbits == 0 {
			mbits = 64
		}
		it.trailing[idx] = 64 - it.leading[idx] - mbits
	}

	mbits := int(64 - it.leading[idx] - it.trailing[idx])
	bits, err := it.br.readBits(mbits)
	if err != nil {
		it.err = err
		return false
	}
	vbits := math.Float64bits(it.curVals[idx])
	vbits ^= (bits << it.trailing[idx])
	it.curVals[idx] = math.Float64frombits(vbits)
	return true
}

func (it *GorillaIter) Next() bool {

	if it.err != nil || it.finished {
		return false
	}

	ok := it.readTimeDelta()

	if !ok {
		return false
	}
	for i := uint8(0); i < 6; i++ {
		ok = it.readValue(i)
		if !ok {
			return false
		}
	}

	if it.start {
		it.start = false
	}
	return true
}

func (it *GorillaIter) Values() (int64, float64, float64, float64, float64, float64, int64) {
	return combineSecNano(it.curTime, it.curTimeMs), it.curVals[0], it.curVals[1], it.curVals[2], it.curVals[3], it.curVals[4], int64(it.curVals[5])
}

func (it *GorillaIter) ReprValue() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  time.Unix(int64(it.curTime), int64(it.curTimeMs)),
		Min:   repr.JsonFloat64(it.curVals[0]),
		Max:   repr.JsonFloat64(it.curVals[1]),
		Last:  repr.JsonFloat64(it.curVals[2]),
		First: repr.JsonFloat64(it.curVals[3]),
		Sum:   repr.JsonFloat64(it.curVals[4]),
		Count: int64(it.curVals[5]),
	}
}

func (it *GorillaIter) Error() error {
	return it.err
}
