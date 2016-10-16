// Code generated by protoc-gen-gogo.
// source: objects.proto
// DO NOT EDIT!

/*
	Package metrics is a generated protocol buffer package.

	It is generated from these files:
		objects.proto

	It has these top-level messages:
		DataPoint
		DataPoints
		RawDataPoint
		RenderItem
		RawRenderItem
*/
package metrics

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import repr "cadent/server/repr"
import _ "github.com/gogo/protobuf/gogoproto"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type DataPoint struct {
	Time  uint32  `protobuf:"varint,1,opt,name=time,proto3" json:"time,omitempty"`
	Value float64 `protobuf:"fixed64,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *DataPoint) Reset()                    { *m = DataPoint{} }
func (m *DataPoint) String() string            { return proto.CompactTextString(m) }
func (*DataPoint) ProtoMessage()               {}
func (*DataPoint) Descriptor() ([]byte, []int) { return fileDescriptorObjects, []int{0} }

type DataPoints struct {
	Points []*DataPoint `protobuf:"bytes,1,rep,name=points" json:"points,omitempty"`
}

func (m *DataPoints) Reset()                    { *m = DataPoints{} }
func (m *DataPoints) String() string            { return proto.CompactTextString(m) }
func (*DataPoints) ProtoMessage()               {}
func (*DataPoints) Descriptor() ([]byte, []int) { return fileDescriptorObjects, []int{1} }

type RawDataPoint struct {
	Time  uint32  `protobuf:"varint,1,opt,name=time,proto3" json:"time,omitempty"`
	Sum   float64 `protobuf:"fixed64,3,opt,name=sum,proto3" json:"sum,omitempty"`
	Min   float64 `protobuf:"fixed64,4,opt,name=min,proto3" json:"min,omitempty"`
	Max   float64 `protobuf:"fixed64,5,opt,name=max,proto3" json:"max,omitempty"`
	Last  float64 `protobuf:"fixed64,6,opt,name=last,proto3" json:"last,omitempty"`
	Count int64   `protobuf:"varint,7,opt,name=count,proto3" json:"count,omitempty"`
}

func (m *RawDataPoint) Reset()                    { *m = RawDataPoint{} }
func (m *RawDataPoint) String() string            { return proto.CompactTextString(m) }
func (*RawDataPoint) ProtoMessage()               {}
func (*RawDataPoint) Descriptor() ([]byte, []int) { return fileDescriptorObjects, []int{2} }

// the basic metric json blob for find
type RenderItem struct {
	Target     string       `protobuf:"bytes,1,opt,name=target,proto3" json:"target,omitempty"`
	DataPoints []*DataPoint `protobuf:"bytes,2,rep,name=dataPoints" json:"dataPoints,omitempty"`
}

func (m *RenderItem) Reset()                    { *m = RenderItem{} }
func (m *RenderItem) String() string            { return proto.CompactTextString(m) }
func (*RenderItem) ProtoMessage()               {}
func (*RenderItem) Descriptor() ([]byte, []int) { return fileDescriptorObjects, []int{3} }

type RawRenderItem struct {
	Metric    string          `protobuf:"bytes,1,opt,name=metric,proto3" json:"metric,omitempty"`
	Id        string          `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	RealStart uint32          `protobuf:"varint,3,opt,name=realStart,proto3" json:"realStart,omitempty"`
	RealEnd   uint32          `protobuf:"varint,4,opt,name=realEnd,proto3" json:"realEnd,omitempty"`
	Start     uint32          `protobuf:"varint,5,opt,name=start,proto3" json:"start,omitempty"`
	End       uint32          `protobuf:"varint,6,opt,name=end,proto3" json:"end,omitempty"`
	Step      uint32          `protobuf:"varint,7,opt,name=step,proto3" json:"step,omitempty"`
	AggFunc   uint32          `protobuf:"varint,8,opt,name=aggFunc,proto3" json:"aggFunc,omitempty"`
	Tags      []*repr.Tag     `protobuf:"bytes,9,rep,name=tags" json:"tags,omitempty"`
	MetaTags  []*repr.Tag     `protobuf:"bytes,10,rep,name=meta_tags,json=metaTags" json:"meta_tags,omitempty"`
	Data      []*RawDataPoint `protobuf:"bytes,11,rep,name=data" json:"data,omitempty"`
}

func (m *RawRenderItem) Reset()                    { *m = RawRenderItem{} }
func (m *RawRenderItem) String() string            { return proto.CompactTextString(m) }
func (*RawRenderItem) ProtoMessage()               {}
func (*RawRenderItem) Descriptor() ([]byte, []int) { return fileDescriptorObjects, []int{4} }

func init() {
	proto.RegisterType((*DataPoint)(nil), "metrics.DataPoint")
	proto.RegisterType((*DataPoints)(nil), "metrics.DataPoints")
	proto.RegisterType((*RawDataPoint)(nil), "metrics.RawDataPoint")
	proto.RegisterType((*RenderItem)(nil), "metrics.RenderItem")
	proto.RegisterType((*RawRenderItem)(nil), "metrics.RawRenderItem")
}
func (m *DataPoint) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *DataPoint) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Time != 0 {
		data[i] = 0x8
		i++
		i = encodeVarintObjects(data, i, uint64(m.Time))
	}
	if m.Value != 0 {
		data[i] = 0x11
		i++
		i = encodeFixed64Objects(data, i, uint64(math.Float64bits(float64(m.Value))))
	}
	return i, nil
}

func (m *DataPoints) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *DataPoints) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Points) > 0 {
		for _, msg := range m.Points {
			data[i] = 0xa
			i++
			i = encodeVarintObjects(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func (m *RawDataPoint) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *RawDataPoint) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Time != 0 {
		data[i] = 0x8
		i++
		i = encodeVarintObjects(data, i, uint64(m.Time))
	}
	if m.Sum != 0 {
		data[i] = 0x19
		i++
		i = encodeFixed64Objects(data, i, uint64(math.Float64bits(float64(m.Sum))))
	}
	if m.Min != 0 {
		data[i] = 0x21
		i++
		i = encodeFixed64Objects(data, i, uint64(math.Float64bits(float64(m.Min))))
	}
	if m.Max != 0 {
		data[i] = 0x29
		i++
		i = encodeFixed64Objects(data, i, uint64(math.Float64bits(float64(m.Max))))
	}
	if m.Last != 0 {
		data[i] = 0x31
		i++
		i = encodeFixed64Objects(data, i, uint64(math.Float64bits(float64(m.Last))))
	}
	if m.Count != 0 {
		data[i] = 0x38
		i++
		i = encodeVarintObjects(data, i, uint64(m.Count))
	}
	return i, nil
}

func (m *RenderItem) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *RenderItem) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Target) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintObjects(data, i, uint64(len(m.Target)))
		i += copy(data[i:], m.Target)
	}
	if len(m.DataPoints) > 0 {
		for _, msg := range m.DataPoints {
			data[i] = 0x12
			i++
			i = encodeVarintObjects(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func (m *RawRenderItem) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *RawRenderItem) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Metric) > 0 {
		data[i] = 0xa
		i++
		i = encodeVarintObjects(data, i, uint64(len(m.Metric)))
		i += copy(data[i:], m.Metric)
	}
	if len(m.Id) > 0 {
		data[i] = 0x12
		i++
		i = encodeVarintObjects(data, i, uint64(len(m.Id)))
		i += copy(data[i:], m.Id)
	}
	if m.RealStart != 0 {
		data[i] = 0x18
		i++
		i = encodeVarintObjects(data, i, uint64(m.RealStart))
	}
	if m.RealEnd != 0 {
		data[i] = 0x20
		i++
		i = encodeVarintObjects(data, i, uint64(m.RealEnd))
	}
	if m.Start != 0 {
		data[i] = 0x28
		i++
		i = encodeVarintObjects(data, i, uint64(m.Start))
	}
	if m.End != 0 {
		data[i] = 0x30
		i++
		i = encodeVarintObjects(data, i, uint64(m.End))
	}
	if m.Step != 0 {
		data[i] = 0x38
		i++
		i = encodeVarintObjects(data, i, uint64(m.Step))
	}
	if m.AggFunc != 0 {
		data[i] = 0x40
		i++
		i = encodeVarintObjects(data, i, uint64(m.AggFunc))
	}
	if len(m.Tags) > 0 {
		for _, msg := range m.Tags {
			data[i] = 0x4a
			i++
			i = encodeVarintObjects(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	if len(m.MetaTags) > 0 {
		for _, msg := range m.MetaTags {
			data[i] = 0x52
			i++
			i = encodeVarintObjects(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	if len(m.Data) > 0 {
		for _, msg := range m.Data {
			data[i] = 0x5a
			i++
			i = encodeVarintObjects(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func encodeFixed64Objects(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Objects(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintObjects(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func NewPopulatedDataPoint(r randyObjects, easy bool) *DataPoint {
	this := &DataPoint{}
	this.Time = uint32(r.Uint32())
	this.Value = float64(r.Float64())
	if r.Intn(2) == 0 {
		this.Value *= -1
	}
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

func NewPopulatedDataPoints(r randyObjects, easy bool) *DataPoints {
	this := &DataPoints{}
	if r.Intn(10) != 0 {
		v1 := r.Intn(5)
		this.Points = make([]*DataPoint, v1)
		for i := 0; i < v1; i++ {
			this.Points[i] = NewPopulatedDataPoint(r, easy)
		}
	}
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

func NewPopulatedRawDataPoint(r randyObjects, easy bool) *RawDataPoint {
	this := &RawDataPoint{}
	this.Time = uint32(r.Uint32())
	this.Sum = float64(r.Float64())
	if r.Intn(2) == 0 {
		this.Sum *= -1
	}
	this.Min = float64(r.Float64())
	if r.Intn(2) == 0 {
		this.Min *= -1
	}
	this.Max = float64(r.Float64())
	if r.Intn(2) == 0 {
		this.Max *= -1
	}
	this.Last = float64(r.Float64())
	if r.Intn(2) == 0 {
		this.Last *= -1
	}
	this.Count = int64(r.Int63())
	if r.Intn(2) == 0 {
		this.Count *= -1
	}
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

func NewPopulatedRenderItem(r randyObjects, easy bool) *RenderItem {
	this := &RenderItem{}
	this.Target = randStringObjects(r)
	if r.Intn(10) != 0 {
		v2 := r.Intn(5)
		this.DataPoints = make([]*DataPoint, v2)
		for i := 0; i < v2; i++ {
			this.DataPoints[i] = NewPopulatedDataPoint(r, easy)
		}
	}
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

func NewPopulatedRawRenderItem(r randyObjects, easy bool) *RawRenderItem {
	this := &RawRenderItem{}
	this.Metric = randStringObjects(r)
	this.Id = randStringObjects(r)
	this.RealStart = uint32(r.Uint32())
	this.RealEnd = uint32(r.Uint32())
	this.Start = uint32(r.Uint32())
	this.End = uint32(r.Uint32())
	this.Step = uint32(r.Uint32())
	this.AggFunc = uint32(r.Uint32())
	if r.Intn(10) != 0 {
		v3 := r.Intn(5)
		this.Tags = make([]*repr.Tag, v3)
		for i := 0; i < v3; i++ {
			this.Tags[i] = repr.NewPopulatedTag(r, easy)
		}
	}
	if r.Intn(10) != 0 {
		v4 := r.Intn(5)
		this.MetaTags = make([]*repr.Tag, v4)
		for i := 0; i < v4; i++ {
			this.MetaTags[i] = repr.NewPopulatedTag(r, easy)
		}
	}
	if r.Intn(10) != 0 {
		v5 := r.Intn(5)
		this.Data = make([]*RawDataPoint, v5)
		for i := 0; i < v5; i++ {
			this.Data[i] = NewPopulatedRawDataPoint(r, easy)
		}
	}
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

type randyObjects interface {
	Float32() float32
	Float64() float64
	Int63() int64
	Int31() int32
	Uint32() uint32
	Intn(n int) int
}

func randUTF8RuneObjects(r randyObjects) rune {
	ru := r.Intn(62)
	if ru < 10 {
		return rune(ru + 48)
	} else if ru < 36 {
		return rune(ru + 55)
	}
	return rune(ru + 61)
}
func randStringObjects(r randyObjects) string {
	v6 := r.Intn(100)
	tmps := make([]rune, v6)
	for i := 0; i < v6; i++ {
		tmps[i] = randUTF8RuneObjects(r)
	}
	return string(tmps)
}
func randUnrecognizedObjects(r randyObjects, maxFieldNumber int) (data []byte) {
	l := r.Intn(5)
	for i := 0; i < l; i++ {
		wire := r.Intn(4)
		if wire == 3 {
			wire = 5
		}
		fieldNumber := maxFieldNumber + r.Intn(100)
		data = randFieldObjects(data, r, fieldNumber, wire)
	}
	return data
}
func randFieldObjects(data []byte, r randyObjects, fieldNumber int, wire int) []byte {
	key := uint32(fieldNumber)<<3 | uint32(wire)
	switch wire {
	case 0:
		data = encodeVarintPopulateObjects(data, uint64(key))
		v7 := r.Int63()
		if r.Intn(2) == 0 {
			v7 *= -1
		}
		data = encodeVarintPopulateObjects(data, uint64(v7))
	case 1:
		data = encodeVarintPopulateObjects(data, uint64(key))
		data = append(data, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	case 2:
		data = encodeVarintPopulateObjects(data, uint64(key))
		ll := r.Intn(100)
		data = encodeVarintPopulateObjects(data, uint64(ll))
		for j := 0; j < ll; j++ {
			data = append(data, byte(r.Intn(256)))
		}
	default:
		data = encodeVarintPopulateObjects(data, uint64(key))
		data = append(data, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	}
	return data
}
func encodeVarintPopulateObjects(data []byte, v uint64) []byte {
	for v >= 1<<7 {
		data = append(data, uint8(uint64(v)&0x7f|0x80))
		v >>= 7
	}
	data = append(data, uint8(v))
	return data
}
func (m *DataPoint) Size() (n int) {
	var l int
	_ = l
	if m.Time != 0 {
		n += 1 + sovObjects(uint64(m.Time))
	}
	if m.Value != 0 {
		n += 9
	}
	return n
}

func (m *DataPoints) Size() (n int) {
	var l int
	_ = l
	if len(m.Points) > 0 {
		for _, e := range m.Points {
			l = e.Size()
			n += 1 + l + sovObjects(uint64(l))
		}
	}
	return n
}

func (m *RawDataPoint) Size() (n int) {
	var l int
	_ = l
	if m.Time != 0 {
		n += 1 + sovObjects(uint64(m.Time))
	}
	if m.Sum != 0 {
		n += 9
	}
	if m.Min != 0 {
		n += 9
	}
	if m.Max != 0 {
		n += 9
	}
	if m.Last != 0 {
		n += 9
	}
	if m.Count != 0 {
		n += 1 + sovObjects(uint64(m.Count))
	}
	return n
}

func (m *RenderItem) Size() (n int) {
	var l int
	_ = l
	l = len(m.Target)
	if l > 0 {
		n += 1 + l + sovObjects(uint64(l))
	}
	if len(m.DataPoints) > 0 {
		for _, e := range m.DataPoints {
			l = e.Size()
			n += 1 + l + sovObjects(uint64(l))
		}
	}
	return n
}

func (m *RawRenderItem) Size() (n int) {
	var l int
	_ = l
	l = len(m.Metric)
	if l > 0 {
		n += 1 + l + sovObjects(uint64(l))
	}
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovObjects(uint64(l))
	}
	if m.RealStart != 0 {
		n += 1 + sovObjects(uint64(m.RealStart))
	}
	if m.RealEnd != 0 {
		n += 1 + sovObjects(uint64(m.RealEnd))
	}
	if m.Start != 0 {
		n += 1 + sovObjects(uint64(m.Start))
	}
	if m.End != 0 {
		n += 1 + sovObjects(uint64(m.End))
	}
	if m.Step != 0 {
		n += 1 + sovObjects(uint64(m.Step))
	}
	if m.AggFunc != 0 {
		n += 1 + sovObjects(uint64(m.AggFunc))
	}
	if len(m.Tags) > 0 {
		for _, e := range m.Tags {
			l = e.Size()
			n += 1 + l + sovObjects(uint64(l))
		}
	}
	if len(m.MetaTags) > 0 {
		for _, e := range m.MetaTags {
			l = e.Size()
			n += 1 + l + sovObjects(uint64(l))
		}
	}
	if len(m.Data) > 0 {
		for _, e := range m.Data {
			l = e.Size()
			n += 1 + l + sovObjects(uint64(l))
		}
	}
	return n
}

func sovObjects(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozObjects(x uint64) (n int) {
	return sovObjects(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *DataPoint) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowObjects
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DataPoint: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DataPoint: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Time", wireType)
			}
			m.Time = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.Time |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var v uint64
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += 8
			v = uint64(data[iNdEx-8])
			v |= uint64(data[iNdEx-7]) << 8
			v |= uint64(data[iNdEx-6]) << 16
			v |= uint64(data[iNdEx-5]) << 24
			v |= uint64(data[iNdEx-4]) << 32
			v |= uint64(data[iNdEx-3]) << 40
			v |= uint64(data[iNdEx-2]) << 48
			v |= uint64(data[iNdEx-1]) << 56
			m.Value = float64(math.Float64frombits(v))
		default:
			iNdEx = preIndex
			skippy, err := skipObjects(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthObjects
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DataPoints) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowObjects
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DataPoints: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DataPoints: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Points", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthObjects
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Points = append(m.Points, &DataPoint{})
			if err := m.Points[len(m.Points)-1].Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipObjects(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthObjects
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *RawDataPoint) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowObjects
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RawDataPoint: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RawDataPoint: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Time", wireType)
			}
			m.Time = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.Time |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sum", wireType)
			}
			var v uint64
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += 8
			v = uint64(data[iNdEx-8])
			v |= uint64(data[iNdEx-7]) << 8
			v |= uint64(data[iNdEx-6]) << 16
			v |= uint64(data[iNdEx-5]) << 24
			v |= uint64(data[iNdEx-4]) << 32
			v |= uint64(data[iNdEx-3]) << 40
			v |= uint64(data[iNdEx-2]) << 48
			v |= uint64(data[iNdEx-1]) << 56
			m.Sum = float64(math.Float64frombits(v))
		case 4:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field Min", wireType)
			}
			var v uint64
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += 8
			v = uint64(data[iNdEx-8])
			v |= uint64(data[iNdEx-7]) << 8
			v |= uint64(data[iNdEx-6]) << 16
			v |= uint64(data[iNdEx-5]) << 24
			v |= uint64(data[iNdEx-4]) << 32
			v |= uint64(data[iNdEx-3]) << 40
			v |= uint64(data[iNdEx-2]) << 48
			v |= uint64(data[iNdEx-1]) << 56
			m.Min = float64(math.Float64frombits(v))
		case 5:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field Max", wireType)
			}
			var v uint64
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += 8
			v = uint64(data[iNdEx-8])
			v |= uint64(data[iNdEx-7]) << 8
			v |= uint64(data[iNdEx-6]) << 16
			v |= uint64(data[iNdEx-5]) << 24
			v |= uint64(data[iNdEx-4]) << 32
			v |= uint64(data[iNdEx-3]) << 40
			v |= uint64(data[iNdEx-2]) << 48
			v |= uint64(data[iNdEx-1]) << 56
			m.Max = float64(math.Float64frombits(v))
		case 6:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field Last", wireType)
			}
			var v uint64
			if (iNdEx + 8) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += 8
			v = uint64(data[iNdEx-8])
			v |= uint64(data[iNdEx-7]) << 8
			v |= uint64(data[iNdEx-6]) << 16
			v |= uint64(data[iNdEx-5]) << 24
			v |= uint64(data[iNdEx-4]) << 32
			v |= uint64(data[iNdEx-3]) << 40
			v |= uint64(data[iNdEx-2]) << 48
			v |= uint64(data[iNdEx-1]) << 56
			m.Last = float64(math.Float64frombits(v))
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Count", wireType)
			}
			m.Count = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.Count |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipObjects(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthObjects
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *RenderItem) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowObjects
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RenderItem: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RenderItem: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Target", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthObjects
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Target = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DataPoints", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthObjects
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DataPoints = append(m.DataPoints, &DataPoint{})
			if err := m.DataPoints[len(m.DataPoints)-1].Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipObjects(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthObjects
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *RawRenderItem) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowObjects
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RawRenderItem: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RawRenderItem: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Metric", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthObjects
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Metric = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthObjects
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RealStart", wireType)
			}
			m.RealStart = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.RealStart |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RealEnd", wireType)
			}
			m.RealEnd = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.RealEnd |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Start", wireType)
			}
			m.Start = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.Start |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field End", wireType)
			}
			m.End = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.End |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Step", wireType)
			}
			m.Step = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.Step |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AggFunc", wireType)
			}
			m.AggFunc = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.AggFunc |= (uint32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Tags", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthObjects
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Tags = append(m.Tags, &repr.Tag{})
			if err := m.Tags[len(m.Tags)-1].Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MetaTags", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthObjects
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MetaTags = append(m.MetaTags, &repr.Tag{})
			if err := m.MetaTags[len(m.MetaTags)-1].Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthObjects
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data, &RawDataPoint{})
			if err := m.Data[len(m.Data)-1].Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipObjects(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthObjects
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipObjects(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowObjects
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowObjects
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthObjects
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowObjects
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipObjects(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthObjects = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowObjects   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("objects.proto", fileDescriptorObjects) }

var fileDescriptorObjects = []byte{
	// 457 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x52, 0xcd, 0x8a, 0xd5, 0x30,
	0x18, 0x9d, 0xde, 0x9f, 0xde, 0xe9, 0x37, 0x56, 0x24, 0xa8, 0x84, 0x61, 0xa6, 0x48, 0x17, 0x32,
	0x0a, 0xb6, 0x30, 0x22, 0xb8, 0x16, 0x15, 0xdc, 0x49, 0x9c, 0x85, 0x3b, 0x49, 0xdb, 0x18, 0x2b,
	0xb7, 0x4d, 0x49, 0xd2, 0x71, 0xd6, 0x3e, 0x81, 0x6f, 0xe3, 0xd6, 0xe5, 0x2c, 0x7d, 0x04, 0x47,
	0x5f, 0xc2, 0xa5, 0xc9, 0x97, 0x4e, 0x1d, 0x05, 0x5d, 0xe4, 0x72, 0xce, 0xb9, 0x27, 0x5f, 0x72,
	0x4e, 0x03, 0xa9, 0xaa, 0xde, 0x8b, 0xda, 0x9a, 0x62, 0xd0, 0xca, 0x2a, 0xb2, 0xe9, 0x84, 0xd5,
	0x6d, 0x6d, 0xf6, 0x0f, 0x6b, 0xde, 0x88, 0xde, 0x96, 0x46, 0xe8, 0x53, 0xa1, 0x4b, 0x2d, 0x86,
	0xf0, 0x13, 0x7c, 0xfb, 0x0f, 0x64, 0x6b, 0xdf, 0x8d, 0x55, 0x51, 0xab, 0xae, 0x94, 0x4a, 0xaa,
	0x12, 0xe5, 0x6a, 0x7c, 0x8b, 0x0c, 0x09, 0xa2, 0x60, 0xcf, 0x1f, 0x41, 0xf2, 0x94, 0x5b, 0xfe,
	0x52, 0xb5, 0xbd, 0x25, 0x04, 0x56, 0xb6, 0xed, 0x04, 0x8d, 0xee, 0x44, 0x47, 0x29, 0x43, 0x4c,
	0x6e, 0xc2, 0xfa, 0x94, 0x6f, 0x47, 0x41, 0x17, 0x4e, 0x8c, 0x58, 0x20, 0xf9, 0x63, 0x80, 0x79,
	0x9b, 0x21, 0xf7, 0x21, 0x1e, 0x10, 0xb9, 0x9d, 0xcb, 0xa3, 0xbd, 0x63, 0x52, 0x4c, 0x97, 0x2d,
	0x66, 0x13, 0x9b, 0x1c, 0xf9, 0xc7, 0x08, 0xae, 0x31, 0xfe, 0xe1, 0xff, 0x87, 0xde, 0x80, 0xa5,
	0x19, 0x3b, 0xba, 0xc4, 0x23, 0x3d, 0xf4, 0x4a, 0xd7, 0xf6, 0x74, 0x15, 0x14, 0x07, 0x51, 0xe1,
	0x67, 0x74, 0x3d, 0x29, 0xfc, 0xcc, 0x4f, 0xda, 0x72, 0x63, 0x69, 0x8c, 0x12, 0x62, 0x7f, 0xfd,
	0x5a, 0x8d, 0xbd, 0xa5, 0x1b, 0x27, 0x2e, 0x59, 0x20, 0xf9, 0x6b, 0x00, 0x26, 0xfa, 0x46, 0xe8,
	0x17, 0x56, 0x74, 0xe4, 0x36, 0xc4, 0x96, 0x6b, 0x29, 0x2c, 0xde, 0x21, 0x61, 0x13, 0x23, 0xc7,
	0x00, 0xcd, 0x1c, 0xd2, 0xe5, 0xff, 0x57, 0xb4, 0x2b, 0xae, 0xfc, 0xf3, 0x02, 0x52, 0x17, 0xef,
	0xcf, 0xe9, 0x61, 0xcb, 0xe5, 0xf4, 0xc0, 0xc8, 0x75, 0x58, 0xb4, 0x0d, 0xb6, 0x9a, 0x30, 0x87,
	0xc8, 0x01, 0x24, 0x5a, 0xf0, 0xed, 0x2b, 0x77, 0xb8, 0xc5, 0xe4, 0x29, 0xfb, 0x2d, 0x10, 0x0a,
	0x1b, 0x4f, 0x9e, 0xf5, 0x0d, 0x76, 0x90, 0xb2, 0x4b, 0xea, 0x13, 0x1a, 0xdc, 0xb3, 0x46, 0x3d,
	0x10, 0xdf, 0x8e, 0xbb, 0x02, 0x56, 0x91, 0x32, 0x0f, 0x7d, 0x3b, 0xc6, 0x8a, 0x01, 0x8b, 0x70,
	0x3d, 0x7b, 0xec, 0xa7, 0x72, 0x29, 0x9f, 0x8f, 0x7d, 0x4d, 0x77, 0xc3, 0xd4, 0x89, 0x92, 0x43,
	0xf7, 0x55, 0xb8, 0x34, 0x34, 0xc1, 0xd4, 0x49, 0x81, 0x2f, 0xec, 0x84, 0x4b, 0x86, 0x32, 0xb9,
	0x0b, 0x89, 0x8b, 0xc1, 0xdf, 0xa0, 0x07, 0xfe, 0xf6, 0xec, 0xfa, 0xff, 0x4e, 0xbc, 0xef, 0x1e,
	0xac, 0x7c, 0x39, 0x74, 0x0f, 0x2d, 0xb7, 0xe6, 0xf2, 0xae, 0xbe, 0x00, 0x86, 0x96, 0x27, 0x07,
	0xe7, 0x17, 0xd9, 0xce, 0xcf, 0x8b, 0x2c, 0xfa, 0xf2, 0x3d, 0x8b, 0xce, 0xdd, 0xfa, 0xea, 0xd6,
	0x37, 0xb7, 0x3e, 0xfd, 0xc8, 0x76, 0xaa, 0x18, 0x9f, 0xeb, 0xc3, 0x5f, 0x01, 0x00, 0x00, 0xff,
	0xff, 0x93, 0x14, 0x2f, 0x64, 0x16, 0x03, 0x00, 0x00,
}
