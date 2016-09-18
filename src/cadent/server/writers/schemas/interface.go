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

/*
	basic "schema" interface
*/

package schemas

import (
	"cadent/server/repr"
)

type SendEncoding uint8

const (
	ENCODE_JSON SendEncoding = iota
	ENCODE_MSGP
)

func SendEncodingFromString(enc string) SendEncoding {
	switch enc {
	case "json":
		return ENCODE_JSON
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
)

func MetricTypeFromString(enc string) MessageType {
	switch enc {
	case "series":
		return MSG_SERIES
	case "unprocessed":
		return MSG_UNPROCESSED
	case "raw":
		return MSG_RAW
	default:
		return MSG_SINGLE
	}
}

// root type needed for in/out
type MessageBase interface {
	SetSendEncoding(enc SendEncoding)
	Length() int
	Encode() ([]byte, error)
	Decode([]byte) error
}

type MessageSingle interface {
	MessageBase
	Repr() *repr.StatRepr
}

type MessageSeries interface {
	MessageBase
	Reprs() []*repr.StatRepr
}
