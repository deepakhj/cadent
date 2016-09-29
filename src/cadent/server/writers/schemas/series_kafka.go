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
)

// root type needed for in/out
type KMessageBase interface {
	SetSendEncoding(enc SendEncoding)
	Id() string
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
	case MSG_ANY:
		return new(KMetric)
	default:
		return new(KSingleMetric)
	}
}

/*************** Envelope message, containing the "type" with it ****/
type KMetric struct {
	AnyMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KMetric) Id() string {
	if kp.Raw != nil {
		return kp.Raw.Metric
	}
	if kp.Unprocessed != nil {
		return kp.Unprocessed.Metric
	}
	if kp.Single != nil {
		return kp.Single.Uid
	}
	if kp.Series != nil {
		return kp.Series.Uid
	}
	return ""
}

func (kp *KMetric) Repr() *repr.StatRepr {
	if kp.Raw != nil {
		return &repr.StatRepr{
			Name: repr.StatName{
				Key:      kp.Raw.Metric,
				Tags:     kp.Raw.TagToSorted(),
				MetaTags: kp.Raw.MetaTagToSorted(),
			},
			Time:  ToTime(kp.Raw.Time),
			Sum:   repr.JsonFloat64(kp.Raw.Value),
			Count: 1,
		}
	}
	if kp.Unprocessed != nil {
		return &repr.StatRepr{
			Name: repr.StatName{
				Key:      kp.Unprocessed.Metric,
				Tags:     kp.Unprocessed.TagToSorted(),
				MetaTags: kp.Unprocessed.MetaTagToSorted(),
			},
			Time:  ToTime(kp.Unprocessed.Time),
			Min:   repr.JsonFloat64(kp.Unprocessed.Min),
			Max:   repr.JsonFloat64(kp.Unprocessed.Max),
			Last:  repr.JsonFloat64(kp.Unprocessed.Last),
			Sum:   repr.JsonFloat64(kp.Unprocessed.Sum),
			Count: kp.Unprocessed.Count,
		}
	}
	if kp.Single != nil {
		return &repr.StatRepr{
			Name: repr.StatName{
				Key:        kp.Single.Metric,
				Tags:       kp.Single.TagToSorted(),
				MetaTags:   kp.Single.MetaTagToSorted(),
				Resolution: kp.Single.Resolution,
				TTL:        kp.Single.Ttl,
			},
			Time:  ToTime(kp.Single.Time),
			Min:   repr.JsonFloat64(kp.Single.Min),
			Max:   repr.JsonFloat64(kp.Single.Max),
			Last:  repr.JsonFloat64(kp.Single.Last),
			Sum:   repr.JsonFloat64(kp.Single.Sum),
			Count: kp.Single.Count,
		}
	}
	if kp.Series != nil {
		return nil
	}
	return nil
}

func (kp *KMetric) SetSendEncoding(enc SendEncoding) {
	kp.encodetype = enc
}

// encodes the 'base' message
func (kp *KMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()

	if kp != nil && kp.encoded == nil && kp.err == nil {

		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.AnyMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.AnyMetric.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp)
			return

		}
	}
}

func (kp *KMetric) Length() int {
	if kp == nil {
		return 0
	}
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KMetric) Encode() ([]byte, error) {
	if kp == nil {
		return nil, ErrMetricIsNil
	}
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KMetric) Decode(b []byte) (err error) {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err = kp.AnyMetric.UnmarshalMsg(b)
	case ENCODE_PROTOBUF:
		err = kp.AnyMetric.Unmarshal(b)
	default:
		err = json.Unmarshal(b, &kp.AnyMetric)
	}

	return err
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

func (kp *KSeriesMetric) Id() string {
	return kp.SeriesMetric.Uid
}

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
			kp.encoded, kp.err = kp.SeriesMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.SeriesMetric.Marshal()
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
		err = kp.SeriesMetric.Unmarshal(b)
	default:
		err = json.Unmarshal(b, &kp.SeriesMetric)
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

func (kp *KSingleMetric) Id() string {
	return kp.SingleMetric.Uid
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
		Time:  ToTime(kp.SingleMetric.Time),
		Min:   repr.JsonFloat64(kp.SingleMetric.Min),
		Max:   repr.JsonFloat64(kp.SingleMetric.Max),
		Last:  repr.JsonFloat64(kp.SingleMetric.Last),
		Sum:   repr.JsonFloat64(kp.SingleMetric.Sum),
		Count: kp.SingleMetric.Count,
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
			kp.encoded, kp.err = kp.SingleMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.SingleMetric.Marshal()
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
		return kp.SingleMetric.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.SingleMetric)
	}
}

type KUnProcessedMetric struct {
	UnProcessedMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KUnProcessedMetric) Id() string {
	return kp.UnProcessedMetric.Metric
}

func (kp *KUnProcessedMetric) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Name: repr.StatName{
			Key:      kp.UnProcessedMetric.Metric,
			Tags:     kp.UnProcessedMetric.TagToSorted(),
			MetaTags: kp.UnProcessedMetric.MetaTagToSorted(),
		},
		Time:  ToTime(kp.UnProcessedMetric.Time),
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
			kp.encoded, kp.err = kp.UnProcessedMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.UnProcessedMetric.Marshal()
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
		return kp.UnProcessedMetric.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.UnProcessedMetric)
	}
}

type KRawMetric struct {
	RawMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KRawMetric) Id() string {
	return kp.RawMetric.Metric
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
			kp.encoded, kp.err = kp.RawMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.RawMetric.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(&kp.RawMetric)

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
		return kp.RawMetric.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.RawMetric)
	}
}

func (kp *KRawMetric) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Name: repr.StatName{
			Key:      kp.RawMetric.Metric,
			Tags:     kp.RawMetric.TagToSorted(),
			MetaTags: kp.RawMetric.MetaTagToSorted(),
		},
		Time:  ToTime(kp.RawMetric.Time),
		Sum:   repr.JsonFloat64(kp.RawMetric.Value),
		Count: 1,
	}
}
