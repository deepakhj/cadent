package schemas

type SendEncoding uint8

const (
	ENCODE_JSON SendEncoding = iota
	ENCODE_MSGP
	ENCODE_PROTOBUF
)

func SendEncodingFromString(enc string) SendEncoding {
	switch enc {
	case "json":
		return ENCODE_JSON
	case "protobuf":
		return ENCODE_PROTOBUF
	default:
		return ENCODE_MSGP
	}
}

type MessageType uint8

const (
	MSG_SERIES MessageType = iota
	MSG_SINGLE
	MSG_UNPROCESSED
	MSG_RAW
	MSG_ANY
)

func MetricTypeFromString(enc string) MessageType {
	switch enc {
	case "series":
		return MSG_SERIES
	case "unprocessed":
		return MSG_UNPROCESSED
	case "raw":
		return MSG_RAW
	case "any":
		return MSG_ANY
	default:
		return MSG_SINGLE
	}
}
