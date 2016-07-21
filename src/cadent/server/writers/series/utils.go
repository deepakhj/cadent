/*
	Interface bits for time series
*/

package series

import (
	"bytes"
	"sync"
	"time"
)

// set up a byte[] sync pool for better GC allocation of byte arraies
// bufPool is a pool for staging buffers. Using a pool allows concurrency-safe
// reuse of buffers
var bytesPool sync.Pool

func getBytes(l int) []byte {
	x := bytesPool.Get()
	if x == nil {
		return make([]byte, l)
	}
	buf := x.([]byte)
	if cap(buf) < l {
		return make([]byte, l)
	}
	return buf[:l]
}

func putBytes(buf []byte) {
	bytesPool.Put(buf)
}

var bytesBufferPool sync.Pool

func getBytesBuffer() *bytes.Buffer {
	x := bytesBufferPool.Get()
	if x == nil {
		return &bytes.Buffer{}
	}
	buf := x.(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putBytesBuffer(buf *bytes.Buffer) {
	bytesBufferPool.Put(buf)
}

// for dealing w/ read buffer copies
var getSyncBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer([]byte{})
	},
}

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

// see if all the floats are the same
func sameFloatVals(min float64, max float64, first float64, last float64, sum float64) bool {
	return min == max && min == first && min == last && min == sum
}
