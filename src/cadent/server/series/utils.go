/*
	Interface bits for time series
*/

package series

import (
	"time"
)

// make the "second" and "nanosecond" parts
func splitNano(t int64) (uint32, uint32) {
	// not "good way" of splitting a Nano-time is available so we need to
	// convert things to "time" and grab the resulting bits

	// already in seconds
	if t <= 2147483647 {
		return uint32(t), 0
	}
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
