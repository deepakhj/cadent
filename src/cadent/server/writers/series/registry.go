/*
	Make it easy to get a new time series algo
*/

package series

import (
	"fmt"
)

func NewTimeSeries(name string, t0 int64) (TimeSeries, error) {
	switch {
	case name == "binary" || name == "gob":
		return NewGobTimeSeries(t0), nil
	case name == "zipbinary" || name == "zipgob":
		return NewZipGobTimeSeries(t0), nil
	case name == "json":
		return NewJsonTimeSeries(t0), nil
	case name == "protobuf":
		return NewProtobufTimeSeries(t0), nil
	case name == "gorilla":
		return NewGoriallaTimeSeries(t0), nil
	default:
		return nil, fmt.Errorf("Invalid time series type `%s`", name)
	}
}
