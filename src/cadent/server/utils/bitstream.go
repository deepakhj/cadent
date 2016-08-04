/** shamelessly taken from https://github.com/dgryski/go-tsz/blob/master/bstream.go */

package utils

import (
	"io"
)

type Bit bool

const (
	ZeroBit Bit = false
	OneBit  Bit = true
)

// BitStream is a stream of bits
type BitStream struct {
	// the data stream
	stream []byte

	// how many bits are valid in current byte
	count uint8

	bitsWritten int64
	bitsRead    int64
}

func NewBReader(b []byte) *BitStream {
	return &BitStream{stream: b, count: 8}
}

func NewBWriter(size int) *BitStream {
	return &BitStream{stream: make([]byte, 0, size), count: 0}
}

func (b *BitStream) SetStream(bs []byte) {
	b.stream = bs
}
func (b *BitStream) SetCount(c uint8) {
	b.count = 8
}

func (b *BitStream) Clone() *BitStream {
	d := make([]byte, len(b.stream))
	copy(d, b.stream)
	return &BitStream{stream: d, count: b.count}
}

func (b *BitStream) Bytes() []byte {
	return b.stream
}

func (b *BitStream) WriteBit(bit Bit) {

	if b.count == 0 {
		b.stream = append(b.stream, 0)
		b.count = 8
	}

	i := len(b.stream) - 1

	if bit {
		b.stream[i] |= 1 << (b.count - 1)
	}
	b.bitsWritten++
	b.count--
}

func (b *BitStream) WriteBytes(bs []byte) int {
	c := 0
	for _, by := range bs {
		b.WriteByte(by)
		c++
	}
	return c
}

func (b *BitStream) WriteByte(byt byte) error {

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
	b.bitsWritten += 8
	return nil
}

func (b *BitStream) WriteBits(u uint64, nbits int) {
	u <<= (64 - uint(nbits))
	for nbits >= 8 {
		byt := byte(u >> 56)
		b.WriteByte(byt)
		u <<= 8
		nbits -= 8
	}

	for nbits > 0 {
		b.WriteBit((u >> 63) == 1)
		u <<= 1
		nbits--
	}
	b.bitsWritten += int64(nbits)
}

func (b *BitStream) ReadBit() (Bit, error) {

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
	b.bitsRead++
	return d != 0, nil
}

func (b *BitStream) ReadBytes(n uint8) ([]byte, error) {

	if len(b.stream) == 0 {
		return nil, io.EOF
	}
	if len(b.stream) < int(n) {
		return nil, io.EOF
	}

	byts := make([]byte, n)
	var err error
	for i := uint8(0); i < n; i++ {
		byts[i], err = b.ReadByte()
		if err != nil {
			return nil, err
		}
	}
	return byts, nil

}

func (b *BitStream) ReadByte() (byte, error) {

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
		b.bitsRead += 8
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

	b.bitsRead += 8
	return byt, nil
}

func (b *BitStream) ReadBits(nbits int) (uint64, error) {

	var u uint64

	for nbits >= 8 {
		byt, err := b.ReadByte()
		if err != nil {
			return 0, err
		}

		u = (u << 8) | uint64(byt)
		nbits -= 8
	}

	var err error
	for nbits > 0 && err != io.EOF {
		byt, err := b.ReadBit()
		if err != nil {
			return 0, err
		}
		u <<= 1
		if byt {
			u |= 1
		}
		nbits--
	}
	b.bitsRead += int64(nbits)

	return u, nil
}
