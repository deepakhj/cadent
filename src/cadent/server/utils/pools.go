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

var mutexPool = sync.Pool{
	New: func() interface{} {
		return new(sync.Mutex)
	},
}

func GetMutex() *sync.Mutex {
	return mutexPool.Get().(*sync.Mutex)
}

func PutMutex(mu *sync.Mutex) {
	mutexPool.Put(mu)
}

var rwMutexPool = sync.Pool{
	New: func() interface{} {
		return new(sync.RWMutex)
	},
}

func GetRWMutex() *sync.RWMutex {
	return rwMutexPool.Get().(*sync.RWMutex)
}

func PutRWMutex(mu *sync.RWMutex) {
	rwMutexPool.Put(mu)
}

var waitGroupPool = sync.Pool{
	New: func() interface{} {
		return new(sync.WaitGroup)
	},
}

func GetWaitGroup() *sync.WaitGroup {
	return waitGroupPool.Get().(*sync.WaitGroup)
}

func PutWaitGroup(mu *sync.WaitGroup) {
	mutexPool.Put(mu)
}
