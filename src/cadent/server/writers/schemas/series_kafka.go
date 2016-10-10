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
	"sync"
)

/** sync pools for ease of ram pressure on fast objects **/
var seriesPool sync.Pool

func getSeries() *KSeriesMetric {
	x := seriesPool.Get()
	if x == nil {
		return new(KSeriesMetric)
	}
	x.(*KSeriesMetric).encoded = []byte{}
	x.(*KSeriesMetric).err = nil
	return x.(*KSeriesMetric)
}

func putSeries(spl *KSeriesMetric) {
	seriesPool.Put(spl)
}

var unpPool sync.Pool

func getUnp() *KUnProcessedMetric {
	x := unpPool.Get()
	if x == nil {
		return new(KUnProcessedMetric)
	}
	x.(*KUnProcessedMetric).encoded = nil
	x.(*KUnProcessedMetric).err = nil
	return x.(*KUnProcessedMetric)
}

func putUnp(spl *KUnProcessedMetric) {
	unpPool.Put(spl)
}

var rawPool sync.Pool

func getRaw() *KRawMetric {
	x := rawPool.Get()
	if x == nil {
		return new(KRawMetric)
	}
	x.(*KRawMetric).encoded = nil
	x.(*KRawMetric).err = nil
	return x.(*KRawMetric)
}

func putRaw(spl *KRawMetric) {
	rawPool.Put(spl)
}

var singPool sync.Pool

func getSing() *KSingleMetric {
	x := singPool.Get()
	if x == nil {
		return new(KSingleMetric)
	}
	x.(*KSingleMetric).encoded = nil
	x.(*KSingleMetric).err = nil
	return x.(*KSingleMetric)
}

func putSing(spl *KSingleMetric) {
	singPool.Put(spl)
}

var anyPool sync.Pool

func getAny() *KMetric {
	x := anyPool.Get()
	if x == nil {
		return new(KMetric)
	}
	x.(*KMetric).encoded = nil
	x.(*KMetric).err = nil
	return x.(*KMetric)
}

func putAny(spl *KMetric) {
	anyPool.Put(spl)
}

func PutPool(ty KMessageBase) {
	switch ty.(type) {
	case *KMetric:
		putAny(ty.(*KMetric))
	case *KSeriesMetric:
		putSeries(ty.(*KSeriesMetric))
	case *KUnProcessedMetric:
		putUnp(ty.(*KUnProcessedMetric))
	case *KRawMetric:
		putRaw(ty.(*KRawMetric))
	case *KSingleMetric:
		putSing(ty.(*KSingleMetric))
	}
}

func KMetricObjectFromType(ty MessageType) KMessageBase {
	switch ty {
	case MSG_SERIES:
		return getSeries()
	case MSG_UNPROCESSED:
		return getUnp()
	case MSG_RAW:
		return getRaw()
	case MSG_ANY:
		return getAny()
	default:
		return getSing()
	}
}

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
			Name: &repr.StatName{
				Key:      kp.Raw.Metric,
				Tags:     kp.Raw.Tags,
				MetaTags: kp.Raw.MetaTags,
			},
			Time:  kp.Raw.Time,
			Sum:   kp.Raw.Value,
			Count: 1,
		}
	}
	if kp.Unprocessed != nil {
		return &repr.StatRepr{
			Name: &repr.StatName{
				Key:      kp.Unprocessed.Metric,
				Tags:     kp.Unprocessed.Tags,
				MetaTags: kp.Unprocessed.MetaTags,
			},
			Time:  kp.Unprocessed.Time,
			Min:   repr.CheckFloat(kp.Unprocessed.Min),
			Max:   repr.CheckFloat(kp.Unprocessed.Max),
			Last:  repr.CheckFloat(kp.Unprocessed.Last),
			Sum:   repr.CheckFloat(kp.Unprocessed.Sum),
			Count: kp.Unprocessed.Count,
		}
	}
	if kp.Single != nil {
		return &repr.StatRepr{
			Name: &repr.StatName{
				Key:        kp.Single.Metric,
				Tags:       kp.Single.Tags,
				MetaTags:   kp.Single.MetaTags,
				Resolution: kp.Single.Resolution,
				Ttl:        kp.Single.Ttl,
			},
			Time:  kp.Single.Time,
			Min:   repr.CheckFloat(kp.Single.Min),
			Max:   repr.CheckFloat(kp.Single.Max),
			Last:  repr.CheckFloat(kp.Single.Last),
			Sum:   repr.CheckFloat(kp.Single.Sum),
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
		Name: &repr.StatName{
			Key:        kp.Metric,
			Tags:       kp.Tags,
			MetaTags:   kp.MetaTags,
			Resolution: kp.Resolution,
			Ttl:        kp.Ttl,
		},
		Min:   repr.CheckFloat(kp.Min),
		Max:   repr.CheckFloat(kp.Max),
		Last:  repr.CheckFloat(kp.Last),
		Sum:   repr.CheckFloat(kp.Sum),
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
		Name: &repr.StatName{
			Key:        kp.SingleMetric.Metric,
			Tags:       kp.SingleMetric.Tags,
			MetaTags:   kp.SingleMetric.MetaTags,
			Resolution: kp.SingleMetric.Resolution,
			Ttl:        kp.SingleMetric.Ttl,
		},
		Time:  kp.SingleMetric.Time,
		Min:   repr.CheckFloat(kp.SingleMetric.Min),
		Max:   repr.CheckFloat(kp.SingleMetric.Max),
		Last:  repr.CheckFloat(kp.SingleMetric.Last),
		Sum:   repr.CheckFloat(kp.SingleMetric.Sum),
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
		Name: &repr.StatName{
			Key:      kp.UnProcessedMetric.Metric,
			Tags:     kp.UnProcessedMetric.Tags,
			MetaTags: kp.UnProcessedMetric.MetaTags,
		},
		Time:  kp.UnProcessedMetric.Time,
		Min:   repr.CheckFloat(kp.UnProcessedMetric.Min),
		Max:   repr.CheckFloat(kp.UnProcessedMetric.Max),
		Last:  repr.CheckFloat(kp.UnProcessedMetric.Last),
		Sum:   repr.CheckFloat(kp.UnProcessedMetric.Sum),
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
		Name: &repr.StatName{
			Key:      kp.RawMetric.Metric,
			Tags:     kp.RawMetric.Tags,
			MetaTags: kp.RawMetric.MetaTags,
		},
		Time:  kp.RawMetric.Time,
		Sum:   repr.CheckFloat(kp.RawMetric.Value),
		Count: 1,
	}
}
