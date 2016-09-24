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
	Encode/Decode bits for Kafka to send metric messages around
*/

package schemas

import (
	"cadent/server/repr"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
)

// root type needed for in/out
type KMessageBase interface {
	SetSendEncoding(enc SendEncoding)
	Length() int
	Encode() ([]byte, error)
	Decode([]byte) error
}

type KMessageSingle interface {
	KMessageBase
	Repr() *repr.StatRepr
}

type KMessageSeries interface {
	KMessageBase
	Reprs() []*repr.StatRepr
}

func KMetricObjectFromType(ty MessageType) KMessageBase {
	switch ty {
	case MSG_SERIES:
		return new(KSeriesMetric)
	case MSG_UNPROCESSED:
		return new(KUnProcessedMetric)
	case MSG_RAW:
		return new(KRawMetric)
	default:
		return new(KSingleMetric)
	}
}

/**************** series **********************/
type KSeriesMetric struct {
	SeriesMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

/*
func (kp *SingleMetric) Reprs() *repr.StatReprSlice {

	// need to convert the mighty

	return &repr.StatRepr{
		Name: repr.StatName{
			Key:        kp.Metric,
			Tags:       kp.Tags,
			MetaTags:   kp.MetaTags,
			Resolution: kp.Resolution,
			TTL:        kp.TTL,
		},
		Min:   repr.JsonFloat64(kp.Min),
		Max:   repr.JsonFloat64(kp.Max),
		Last:  repr.JsonFloat64(kp.Last),
		Sum:   repr.JsonFloat64(kp.Sum),
		Count: kp.Count,
	}
}*/

func (kp *KSeriesMetric) SetSendEncoding(enc SendEncoding) {
	kp.encodetype = enc
}

func (kp *KSeriesMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()

	if kp != nil && kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			_, kp.err = kp.SeriesMetric.MarshalTo(kp.encoded)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = proto.Marshal(&kp.SeriesMetric)
		default:
			kp.encoded, kp.err = json.Marshal(kp.SeriesMetric)

		}
	}
}

func (kp *KSeriesMetric) Length() int {
	if kp == nil {
		return 0
	}
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KSeriesMetric) Encode() ([]byte, error) {
	if kp == nil {
		return nil, ErrMetricIsNil
	}
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KSeriesMetric) Decode(b []byte) (err error) {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err = kp.UnmarshalMsg(b)
	case ENCODE_PROTOBUF:
		err = proto.Unmarshal(b, &kp.SeriesMetric)
	default:
		err = json.Unmarshal(b, kp)
	}
	return err
}

/**************** Single **********************/

type KSingleMetric struct {
	SingleMetric

	Type       string
	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KSingleMetric) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Name: repr.StatName{
			Key:        kp.SingleMetric.Metric,
			Tags:       kp.SingleMetric.TagToSorted(),
			MetaTags:   kp.SingleMetric.MetaTagToSorted(),
			Resolution: kp.SingleMetric.Resolution,
			TTL:        kp.SingleMetric.Ttl,
		},
		Min:   repr.JsonFloat64(kp.SingleMetric.Min),
		Max:   repr.JsonFloat64(kp.SingleMetric.Max),
		Last:  repr.JsonFloat64(kp.SingleMetric.Last),
		Sum:   repr.JsonFloat64(kp.SingleMetric.Sum),
		Count: kp.Count,
	}
}

func (kp *KSingleMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			_, kp.err = kp.SingleMetric.MarshalTo(kp.encoded)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = proto.Marshal(&kp.SingleMetric)
		default:
			kp.encoded, kp.err = json.Marshal(kp.SingleMetric)

		}
	}
}

func (kp *KSingleMetric) SetSendEncoding(enc SendEncoding) {
	kp.encodetype = enc
}

func (kp *KSingleMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KSingleMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KSingleMetric) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.SingleMetric.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return proto.Unmarshal(kp.encoded, &kp.SingleMetric)
	default:
		return json.Unmarshal(b, kp.SingleMetric)
	}
}

type KUnProcessedMetric struct {
	UnProcessedMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KUnProcessedMetric) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Name: repr.StatName{
			Key:      kp.UnProcessedMetric.Metric,
			Tags:     kp.UnProcessedMetric.TagToSorted(),
			MetaTags: kp.UnProcessedMetric.MetaTagToSorted(),
		},
		Min:   repr.JsonFloat64(kp.UnProcessedMetric.Min),
		Max:   repr.JsonFloat64(kp.UnProcessedMetric.Max),
		Last:  repr.JsonFloat64(kp.UnProcessedMetric.Last),
		Sum:   repr.JsonFloat64(kp.UnProcessedMetric.Sum),
		Count: kp.UnProcessedMetric.Count,
	}
}

func (kp *KUnProcessedMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			_, kp.err = kp.UnProcessedMetric.MarshalTo(kp.encoded)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = proto.Marshal(&kp.UnProcessedMetric)
		default:
			kp.encoded, kp.err = json.Marshal(kp.UnProcessedMetric)

		}
	}
}

func (kp *KUnProcessedMetric) SetSendEncoding(enc SendEncoding) {
	kp.encodetype = enc
}

func (kp *KUnProcessedMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KUnProcessedMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KUnProcessedMetric) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.UnProcessedMetric.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return proto.Unmarshal(kp.encoded, &kp.UnProcessedMetric)
	default:
		return json.Unmarshal(b, kp.UnProcessedMetric)
	}
}

type KRawMetric struct {
	RawMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KRawMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			_, kp.err = kp.RawMetric.MarshalTo(kp.encoded)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = proto.Marshal(&kp.RawMetric)
		default:
			kp.encoded, kp.err = json.Marshal(kp.RawMetric)

		}
	}
}

func (kp *KRawMetric) SetSendEncoding(enc SendEncoding) {
	kp.encodetype = enc
}

func (kp *KRawMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KRawMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KRawMetric) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.RawMetric.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return proto.Unmarshal(kp.encoded, &kp.RawMetric)
	default:
		return json.Unmarshal(b, kp.RawMetric)
	}
}
