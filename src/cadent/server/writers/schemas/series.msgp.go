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
//go:generate msgp -o series_msgp.go --file series.msgp.go

package schemas

import (
	"encoding/json"
	"fmt"
)

func MetricObjectFromType(ty MessageType) MessageBase {
	switch ty {
	case MSG_SERIES:
		return new(SeriesMetric)
	case MSG_UNPROCESSED:
		return new(UnProcessedMetric)
	default:
		return new(SingleMetric)
	}
}

/**************** series **********************/
type SeriesMetric struct {
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

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *SeriesMetric) SetSendEncoding(enc SendEncoding) {
	kp.encodetype = enc
}

func (kp *SeriesMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()

	if kp != nil && kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.MarshalMsg(kp.encoded)
		default:
			kp.encoded, kp.err = json.Marshal(kp)

		}
	}
}

func (kp *SeriesMetric) Length() int {
	if kp == nil {
		return 0
	}
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *SeriesMetric) Encode() ([]byte, error) {
	if kp == nil {
		return nil, ErrMetricIsNil
	}
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *SeriesMetric) Decode(b []byte) (err error) {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err = kp.UnmarshalMsg(b)
	default:
		err = json.Unmarshal(b, kp)
	}
	return err
}

/**************** Single **********************/

type SingleMetric struct {
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

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *SingleMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.MarshalMsg(kp.encoded)
		default:
			kp.encoded, kp.err = json.Marshal(kp)

		}
	}
}

func (kp *SingleMetric) SetSendEncoding(enc SendEncoding) {
	kp.encodetype = enc
}

func (kp *SingleMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *SingleMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *SingleMetric) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.UnmarshalMsg(b)
		return err
	default:
		return json.Unmarshal(b, kp)
	}
}

type UnProcessedMetric struct {
	Time     int64      `json:"time" msg:"time"`
	Metric   string     `json:"metric" msg:"metric"`
	Min      float64    `json:"min" msg:"min"`
	Max      float64    `json:"max" msg:"max"`
	Last     float64    `json:"last" msg:"last"`
	Sum      float64    `json:"sum" msg:"sum"`
	Count    int64      `json:"count" msg:"count"`
	Tags     [][]string `json:"tags,omitempty" codec:"tags,omitempty" msg:"tags,omitempty"` // key1=value1,key2=value2...
	MetaTags [][]string `json:"metatags,omitempty" codec:"tags,omitempty" msg:"metatags,omitempty"`

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *UnProcessedMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.MarshalMsg(kp.encoded)
		default:
			kp.encoded, kp.err = json.Marshal(kp)

		}
	}
}

func (kp *UnProcessedMetric) SetSendEncoding(enc SendEncoding) {
	kp.encodetype = enc
}

func (kp *UnProcessedMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *UnProcessedMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *UnProcessedMetric) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.UnmarshalMsg(b)
		return err
	default:
		return json.Unmarshal(b, kp)
	}
}
