/*
	Make it easy to get a new time series algo .. not exactly the optimal
	"registry" technique but it works
*/

package series

import (
	"fmt"
)

const (
	GOB uint8 = iota
	ZIPGOB
	JSON
	PROTOBUF
	GORILLA
	CODEC
	MSGPACK
	BINC
	CBOR
	REPR
)

func NameFromId(id uint8) string {
	switch id {
	case GOB:
		return "gob"
	case ZIPGOB:
		return "zipbob"
	case JSON:
		return "json"
	case PROTOBUF:
		return "protobuf"
	case GORILLA:
		return "gorilla"
	case MSGPACK:
		return "msgpack"
	case CODEC:
		return "codec"
	case BINC:
		return "binc"
	case CBOR:
		return "cbor"
	case REPR:
		return "repr"
	default:
		return ""
	}
}

func IdFromName(nm string) uint8 {
	switch nm {
	case "gob":
		return GOB
	case "zipgob":
		return ZIPGOB
	case "json":
		return JSON
	case "protobuf":
		return PROTOBUF
	case "gorilla":
		return GORILLA
	case "msgpack":
		return MSGPACK
	case "codec":
		return CODEC
	case "binc":
		return BINC
	case "cbor":
		return CBOR
	case "REPR":
		return REPR
	default:
		return 0
	}
}

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
	case name == "msgpack":
		return NewMsgpackTimeSeries(t0, options), nil
	case name == "binc":
		options.Handler = "binc"
		return NewCodecTimeSeries(t0, options), nil
	case name == "cbor":
		options.Handler = "cbor"
		return NewCodecTimeSeries(t0, options), nil
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
	case name == "msgpack":
		return NewMsgpackIterFromBytes(data)
	case name == "binc" || name == "cbor":
		return NewCodecIterFromBytes(data)
	case name == "repr":
		return NewReprIterFromBytes(data)
	default:
		return nil, fmt.Errorf("Invalid time series type `%s`", name)
	}
}
