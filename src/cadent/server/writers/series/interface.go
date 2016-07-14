/*
	Interface bits for time series
*/

package series

import (
	"cadent/server/repr"
)

type TimeSeries interface {
	UnmarshalBinary([]byte) error
	AddPoint(t int64, min float64, max float64, first float64, last float64, sum float64, count int64) error
	AddStat(*repr.StatRepr) error

	//NOTE these two methods render the write buffer dead, and thus the "time series" complete
	// use "IterClone" or "ByteClone" to get the current
	// read buffer and not effect the write buffer
	Iter() (TimeSeriesIter, error)
	MarshalBinary() ([]byte, error)

	// grab the current buffer, which has the side effect of having to "re-write" a new buffer
	// in the internal object from the clone, so there is a penalty for this
	Bytes() []byte

	// num points in the mix
	Count() int

	Len() int
	StartTime() int64
	LastTime() int64
}

type TimeSeriesIter interface {
	Next() bool
	Values() (int64, float64, float64, float64, float64, float64, int64)
	ReprValue() *repr.StatRepr
	Error() error
}
