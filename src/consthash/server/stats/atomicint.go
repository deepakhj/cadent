// a simple atomic stat counter/rate
package stats

import (
	"strconv"
	"sync/atomic"
)

// An AtomicInt is an int64 to be accessed atomically.
type AtomicInt int64

// Add atomically adds n to i.
func (i *AtomicInt) Add(n int64) int64 {
	return atomic.AddInt64((*int64)(i), n)
}

// Get atomically gets the value of i.
func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

// Set the int to an arb number to a number
func (i *AtomicInt) Set(n int64) {
	atomic.StoreInt64((*int64)(i), n)
}

func (i *AtomicInt) String() string {
	return strconv.FormatInt(i.Get(), 10)
}
