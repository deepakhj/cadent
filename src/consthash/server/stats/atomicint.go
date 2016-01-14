// a simple atomic stat counter/rate
package stats

import (
	"strconv"
	"sync"
)

// An AtomicInt is an int64 to be accessed atomically.
type AtomicInt struct {
	Val int64
	mu  sync.Mutex
}

// Add atomically adds n to i.
func (i *AtomicInt) Add(n int64) int64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Val += n
	return i.Val
}

// Get atomically gets the value of i.
func (i *AtomicInt) Get() int64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.Val
}

// Set the int to an arb number to a number
func (i *AtomicInt) Set(n int64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Val = n
}

func (i *AtomicInt) Equal(n int64) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.Val == n
}

func (i *AtomicInt) String() string {
	return strconv.FormatInt(i.Get(), 10)
}
