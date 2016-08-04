package utils

import (
	"bytes"
	"sync"
)

// set up a byte[] sync pool for better GC allocation of byte arraies
// bufPool is a pool for staging buffers. Using a pool allows concurrency-safe
// reuse of buffers
var bytesPool sync.Pool

func GetBytes(l int) []byte {
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

func PutBytes(buf []byte) {
	bytesPool.Put(buf)
}

var bytesBufferPool sync.Pool

func GetBytesBuffer() *bytes.Buffer {
	x := bytesBufferPool.Get()
	if x == nil {
		return &bytes.Buffer{}
	}
	buf := x.(*bytes.Buffer)
	buf.Reset()
	return buf
}

func PutBytesBuffer(buf *bytes.Buffer) {
	bytesBufferPool.Put(buf)
}

// for dealing w/ read buffer copies
var GetSyncBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer([]byte{})
	},
}
