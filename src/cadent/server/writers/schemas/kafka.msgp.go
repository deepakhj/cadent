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
	message pack or json "generator" options


*/
//go:generate msgp -o kafka_msgp.go --file kafka.msgp.go

package schemas

import (
	"encoding/json"
	"fmt"
)

type KafkaEncodingType uint8

const (
	KAFKA_ENCODE_JSON KafkaEncodingType = iota
	KAFKA_ENCODE_MSGP
)

func KafkaEncodingFromString(enc string) KafkaEncodingType {
	switch enc {
	case "json":
		return KAFKA_ENCODE_JSON
	default:
		return KAFKA_ENCODE_MSGP
	}
}

/** kafka put object **/
type KafkaSeriesMetric struct {
	Type       string     `json:"type" codec:"type" msg:"type"`
	Id         uint64     `json:"id" codec:"id" msg:"id"`
	Uid        string     `json:"uid" codec:"uid" msg:"uid"`
	Time       int64      `json:"time" codec:"time" msg:"time"`
	Metric     string     `json:"metric" codec:"metric" msg:"metric"`
	Encoding   string     `json:"encoding" codec:"encoding" msg:"encoding"`
	Data       []byte     `json:"data" codec:"data" msg:"data"`
	Resolution uint32     `json:"resolution" codec:"resolution" msg:"resolution"`
	TTL        uint32     `json:"ttl" codec:"ttl" msg:"ttl"`
	Tags       [][]string `json:"tags,omitempty" codec:"tags,omitempty" msg:"tags,omitempty"` // key1=value1,key2=value2...
	MetaTags   [][]string `json:"metatags,omitempty" codec:"tags,omitempty" msg:"metatags,omitempty"`

	encodetype KafkaEncodingType
	encoded    []byte
	err        error
}

func (kp *KafkaSeriesMetric) SetEncoding(enc KafkaEncodingType) {
	kp.encodetype = enc
}

func (kp *KafkaSeriesMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()

	if kp != nil && kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case KAFKA_ENCODE_MSGP:
			kp.encoded, kp.err = kp.MarshalMsg(kp.encoded)
		default:
			kp.encoded, kp.err = json.Marshal(kp)

		}
	}
}

func (kp *KafkaSeriesMetric) Length() int {
	if kp == nil {
		return 0
	}
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KafkaSeriesMetric) Encode() ([]byte, error) {
	if kp == nil {
		return nil, ErrMetricIsNil
	}
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

/**** Flat metrics ****/

/** kafka put object **/
type KafkaSingleMetric struct {
	Type       string     `json:"type" msg:"type"`
	Id         uint64     `json:"id" msg:"id"`
	Uid        string     `json:"uid" msg:"uid"`
	Time       int64      `json:"time" msg:"time"`
	Metric     string     `json:"metric" msg:"metric"`
	Min        float64    `json:"min" msg:"min"`
	Max        float64    `json:"max" msg:"max"`
	Last       float64    `json:"last" msg:"last"`
	Sum        float64    `json:"sum" msg:"sum"`
	Count      int64      `json:"count" msg:"count"`
	Resolution uint32     `json:"resolution" msg:"resolution"`
	TTL        uint32     `json:"ttl" msg:"ttl"`
	Tags       [][]string `json:"tags,omitempty" codec:"tags,omitempty" msg:"tags,omitempty"` // key1=value1,key2=value2...
	MetaTags   [][]string `json:"metatags,omitempty" codec:"tags,omitempty" msg:"metatags,omitempty"`

	encodetype KafkaEncodingType
	encoded    []byte
	err        error
}

func (kp *KafkaSingleMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case KAFKA_ENCODE_MSGP:
			kp.encoded, kp.err = kp.MarshalMsg(kp.encoded)
		default:
			kp.encoded, kp.err = json.Marshal(kp)

		}
	}
}

func (kp *KafkaSingleMetric) SetEncoding(enc KafkaEncodingType) {
	kp.encodetype = enc
}

func (kp *KafkaSingleMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KafkaSingleMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}
