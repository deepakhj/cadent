// a simple atomic stat counter/rate
package stats

import (
	"expvar"
	"strconv"
)

// An AtomicInt is an int64 to be accessed atomically.
type AtomicInt struct {
	Val *expvar.Int
}

func NewAtomic(name string) *AtomicInt {
	var att *AtomicInt
	gots := expvar.Get(name)
	if gots == nil {
		att = &AtomicInt{
			Val: expvar.NewInt(name),
		}
		att.Set(0)
	} else {
		att = &AtomicInt{
			Val: gots.(*expvar.Int),
		}
	}
	return att
}

// Add atomically adds n to i.
func (i *AtomicInt) Add(n int64) int64 {
	i.Val.Add(n)
	return i.Get()
}

// Get atomically gets the value of i.
func (i *AtomicInt) Get() int64 {
	ret, _ := strconv.Atoi(i.Val.String())
	return int64(ret)
}

// Set the int to an arb number to a number
func (i *AtomicInt) Set(n int64) {
	i.Val.Set(n)
}

func (i *AtomicInt) Equal(n int64) bool {
	return i.Get() == n
}

func (i *AtomicInt) String() string {
	return i.Val.String()
}
