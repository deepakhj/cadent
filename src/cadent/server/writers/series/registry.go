/*
	Make it easy to get a new time series algo
*/

package series

import (
	"fmt"
)

func NewTimeSeries(name string, t0 int64, options *Options) (TimeSeries, error) {
	if options == nil {
		options = NewDefaultOptions()
	}
	switch {
	case name == "binary" || name == "gob":
		return NewGobTimeSeries(t0, options), nil
	case name == "zipbinary" || name == "zipgob":
		return NewZipGobTimeSeries(t0, options), nil
	case name == "json":
		return NewJsonTimeSeries(t0, options), nil
	case name == "protobuf":
		return NewProtobufTimeSeries(t0, options), nil
	case name == "gorilla":
		return NewGoriallaTimeSeries(t0, options), nil
	case name == "repr":
		return NewReprTimeSeries(t0, options), nil
	default:
		return nil, fmt.Errorf("Invalid time series type `%s`", name)
	}
}

func NewIter(name string, data []byte) (TimeSeriesIter, error) {

	switch {
	case name == "binary" || name == "gob":
		return NewGobIterFromBytes(data)
	case name == "zipbinary" || name == "zipgob":
		return NewZipGobIterFromBytes(data)
	case name == "json":
		return NewJsonIterFromBytes(data)
	case name == "protobuf":
		return NewProtobufIterFromBytes(data)
	case name == "gorilla":
		return NewGorillaIterFromBytes(data)
	case name == "repr":
		return NewReprIterFromBytes(data)
	default:
		return nil, fmt.Errorf("Invalid time series type `%s`", name)
	}
}
