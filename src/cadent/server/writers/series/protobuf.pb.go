// Code generated by protoc-gen-gogo.
// source: protobuf.proto
// DO NOT EDIT!

/*
	Package series is a generated protocol buffer package.

	It is generated from these files:
		protobuf.proto

	It has these top-level messages:
		ProtStatFull
		ProtStatSmall
		ProtStat
		ProtStats
*/
package series

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import github_com_golang_protobuf_proto "github.com/golang/protobuf/proto"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

type ProtStatFull struct {
	Time             *int64   `protobuf:"varint,1,req,name=time" json:"time,omitempty"`
	Count            *int64   `protobuf:"varint,7,req,name=count" json:"count,omitempty"`
	Min              *float64 `protobuf:"fixed64,2,req,name=min" json:"min,omitempty"`
	Max              *float64 `protobuf:"fixed64,3,req,name=max" json:"max,omitempty"`
	Sum              *float64 `protobuf:"fixed64,4,req,name=sum" json:"sum,omitempty"`
	First            *float64 `protobuf:"fixed64,5,req,name=first" json:"first,omitempty"`
	Last             *float64 `protobuf:"fixed64,6,req,name=last" json:"last,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *ProtStatFull) Reset()                    { *m = ProtStatFull{} }
func (m *ProtStatFull) String() string            { return proto.CompactTextString(m) }
func (*ProtStatFull) ProtoMessage()               {}
func (*ProtStatFull) Descriptor() ([]byte, []int) { return fileDescriptorProtobuf, []int{0} }

func (m *ProtStatFull) GetTime() int64 {
	if m != nil && m.Time != nil {
		return *m.Time
	}
	return 0
}

func (m *ProtStatFull) GetCount() int64 {
	if m != nil && m.Count != nil {
		return *m.Count
	}
	return 0
}

func (m *ProtStatFull) GetMin() float64 {
	if m != nil && m.Min != nil {
		return *m.Min
	}
	return 0
}

func (m *ProtStatFull) GetMax() float64 {
	if m != nil && m.Max != nil {
		return *m.Max
	}
	return 0
}

func (m *ProtStatFull) GetSum() float64 {
	if m != nil && m.Sum != nil {
		return *m.Sum
	}
	return 0
}

func (m *ProtStatFull) GetFirst() float64 {
	if m != nil && m.First != nil {
		return *m.First
	}
	return 0
}

func (m *ProtStatFull) GetLast() float64 {
	if m != nil && m.Last != nil {
		return *m.Last
	}
	return 0
}

// object for just int32 time and one value
type ProtStatSmall struct {
	Time             *int64   `protobuf:"varint,1,req,name=time" json:"time,omitempty"`
	Val              *float64 `protobuf:"fixed64,2,req,name=val" json:"val,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *ProtStatSmall) Reset()                    { *m = ProtStatSmall{} }
func (m *ProtStatSmall) String() string            { return proto.CompactTextString(m) }
func (*ProtStatSmall) ProtoMessage()               {}
func (*ProtStatSmall) Descriptor() ([]byte, []int) { return fileDescriptorProtobuf, []int{1} }

func (m *ProtStatSmall) GetTime() int64 {
	if m != nil && m.Time != nil {
		return *m.Time
	}
	return 0
}

func (m *ProtStatSmall) GetVal() float64 {
	if m != nil && m.Val != nil {
		return *m.Val
	}
	return 0
}

type ProtStat struct {
	StatType         *bool          `protobuf:"varint,1,req,name=stat_type" json:"stat_type,omitempty"`
	Stat             *ProtStatFull  `protobuf:"bytes,2,opt,name=stat" json:"stat,omitempty"`
	SmallStat        *ProtStatSmall `protobuf:"bytes,3,opt,name=small_stat" json:"small_stat,omitempty"`
	XXX_unrecognized []byte         `json:"-"`
}

func (m *ProtStat) Reset()                    { *m = ProtStat{} }
func (m *ProtStat) String() string            { return proto.CompactTextString(m) }
func (*ProtStat) ProtoMessage()               {}
func (*ProtStat) Descriptor() ([]byte, []int) { return fileDescriptorProtobuf, []int{2} }

func (m *ProtStat) GetStatType() bool {
	if m != nil && m.StatType != nil {
		return *m.StatType
	}
	return false
}

func (m *ProtStat) GetStat() *ProtStatFull {
	if m != nil {
		return m.Stat
	}
	return nil
}

func (m *ProtStat) GetSmallStat() *ProtStatSmall {
	if m != nil {
		return m.SmallStat
	}
	return nil
}

type ProtStats struct {
	Stats            []*ProtStat `protobuf:"bytes,2,rep,name=stats" json:"stats,omitempty"`
	XXX_unrecognized []byte      `json:"-"`
}

func (m *ProtStats) Reset()                    { *m = ProtStats{} }
func (m *ProtStats) String() string            { return proto.CompactTextString(m) }
func (*ProtStats) ProtoMessage()               {}
func (*ProtStats) Descriptor() ([]byte, []int) { return fileDescriptorProtobuf, []int{3} }

func (m *ProtStats) GetStats() []*ProtStat {
	if m != nil {
		return m.Stats
	}
	return nil
}

func init() {
	proto.RegisterType((*ProtStatFull)(nil), "series.ProtStatFull")
	proto.RegisterType((*ProtStatSmall)(nil), "series.ProtStatSmall")
	proto.RegisterType((*ProtStat)(nil), "series.ProtStat")
	proto.RegisterType((*ProtStats)(nil), "series.ProtStats")
}
func (m *ProtStatFull) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *ProtStatFull) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Time == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x8
		i++
		i = encodeVarintProtobuf(data, i, uint64(*m.Time))
	}
	if m.Min == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x11
		i++
		i = encodeFixed64Protobuf(data, i, uint64(math.Float64bits(float64(*m.Min))))
	}
	if m.Max == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x19
		i++
		i = encodeFixed64Protobuf(data, i, uint64(math.Float64bits(float64(*m.Max))))
	}
	if m.Sum == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x21
		i++
		i = encodeFixed64Protobuf(data, i, uint64(math.Float64bits(float64(*m.Sum))))
	}
	if m.First == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x29
		i++
		i = encodeFixed64Protobuf(data, i, uint64(math.Float64bits(float64(*m.First))))
	}
	if m.Last == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x31
		i++
		i = encodeFixed64Protobuf(data, i, uint64(math.Float64bits(float64(*m.Last))))
	}
	if m.Count == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x38
		i++
		i = encodeVarintProtobuf(data, i, uint64(*m.Count))
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func (m *ProtStatSmall) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *ProtStatSmall) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Time == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x8
		i++
		i = encodeVarintProtobuf(data, i, uint64(*m.Time))
	}
	if m.Val == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x11
		i++
		i = encodeFixed64Protobuf(data, i, uint64(math.Float64bits(float64(*m.Val))))
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func (m *ProtStat) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *ProtStat) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.StatType == nil {
		return 0, new(github_com_golang_protobuf_proto.RequiredNotSetError)
	} else {
		data[i] = 0x8
		i++
		if *m.StatType {
			data[i] = 1
		} else {
			data[i] = 0
		}
		i++
	}
	if m.Stat != nil {
		data[i] = 0x12
		i++
		i = encodeVarintProtobuf(data, i, uint64(m.Stat.Size()))
		n1, err := m.Stat.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n1
	}
	if m.SmallStat != nil {
		data[i] = 0x1a
		i++
		i = encodeVarintProtobuf(data, i, uint64(m.SmallStat.Size()))
		n2, err := m.SmallStat.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n2
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func (m *ProtStats) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *ProtStats) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Stats) > 0 {
		for _, msg := range m.Stats {
			data[i] = 0x12
			i++
			i = encodeVarintProtobuf(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func encodeFixed64Protobuf(data []byte, offset int, v uint64) int {
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
func encodeFixed32Protobuf(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintProtobuf(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *ProtStatFull) Size() (n int) {
	var l int
	_ = l
	if m.Time != nil {
		n += 1 + sovProtobuf(uint64(*m.Time))
	}
	if m.Min != nil {
		n += 9
	}
	if m.Max != nil {
		n += 9
	}
	if m.Sum != nil {
		n += 9
	}
	if m.First != nil {
		n += 9
	}
	if m.Last != nil {
		n += 9
	}
	if m.Count != nil {
		n += 1 + sovProtobuf(uint64(*m.Count))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ProtStatSmall) Size() (n int) {
	var l int
	_ = l
	if m.Time != nil {
		n += 1 + sovProtobuf(uint64(*m.Time))
	}
	if m.Val != nil {
		n += 9
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ProtStat) Size() (n int) {
	var l int
	_ = l
	if m.StatType != nil {
		n += 2
	}
	if m.Stat != nil {
		l = m.Stat.Size()
		n += 1 + l + sovProtobuf(uint64(l))
	}
	if m.SmallStat != nil {
		l = m.SmallStat.Size()
		n += 1 + l + sovProtobuf(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ProtStats) Size() (n int) {
	var l int
	_ = l
	if len(m.Stats) > 0 {
		for _, e := range m.Stats {
			l = e.Size()
			n += 1 + l + sovProtobuf(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovProtobuf(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozProtobuf(x uint64) (n int) {
	return sovProtobuf(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ProtStatFull) Unmarshal(data []byte) error {
	var hasFields [1]uint64
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtobuf
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
			return fmt.Errorf("proto: ProtStatFull: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ProtStatFull: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Time", wireType)
			}
			var v int64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtobuf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Time = &v
			hasFields[0] |= uint64(0x00000001)
		case 2:
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
			v2 := float64(math.Float64frombits(v))
			m.Min = &v2
			hasFields[0] |= uint64(0x00000002)
		case 3:
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
			v2 := float64(math.Float64frombits(v))
			m.Max = &v2
			hasFields[0] |= uint64(0x00000004)
		case 4:
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
			v2 := float64(math.Float64frombits(v))
			m.Sum = &v2
			hasFields[0] |= uint64(0x00000008)
		case 5:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field First", wireType)
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
			v2 := float64(math.Float64frombits(v))
			m.First = &v2
			hasFields[0] |= uint64(0x00000010)
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
			v2 := float64(math.Float64frombits(v))
			m.Last = &v2
			hasFields[0] |= uint64(0x00000020)
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Count", wireType)
			}
			var v int64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtobuf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Count = &v
			hasFields[0] |= uint64(0x00000040)
		default:
			iNdEx = preIndex
			skippy, err := skipProtobuf(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtobuf
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}
	if hasFields[0]&uint64(0x00000001) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}
	if hasFields[0]&uint64(0x00000002) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}
	if hasFields[0]&uint64(0x00000004) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}
	if hasFields[0]&uint64(0x00000008) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}
	if hasFields[0]&uint64(0x00000010) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}
	if hasFields[0]&uint64(0x00000020) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}
	if hasFields[0]&uint64(0x00000040) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ProtStatSmall) Unmarshal(data []byte) error {
	var hasFields [1]uint64
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtobuf
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
			return fmt.Errorf("proto: ProtStatSmall: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ProtStatSmall: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Time", wireType)
			}
			var v int64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtobuf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Time = &v
			hasFields[0] |= uint64(0x00000001)
		case 2:
			if wireType != 1 {
				return fmt.Errorf("proto: wrong wireType = %d for field Val", wireType)
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
			v2 := float64(math.Float64frombits(v))
			m.Val = &v2
			hasFields[0] |= uint64(0x00000002)
		default:
			iNdEx = preIndex
			skippy, err := skipProtobuf(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtobuf
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}
	if hasFields[0]&uint64(0x00000001) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}
	if hasFields[0]&uint64(0x00000002) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ProtStat) Unmarshal(data []byte) error {
	var hasFields [1]uint64
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtobuf
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
			return fmt.Errorf("proto: ProtStat: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ProtStat: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field StatType", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtobuf
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			b := bool(v != 0)
			m.StatType = &b
			hasFields[0] |= uint64(0x00000001)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Stat", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtobuf
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
				return ErrInvalidLengthProtobuf
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Stat == nil {
				m.Stat = &ProtStatFull{}
			}
			if err := m.Stat.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SmallStat", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtobuf
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
				return ErrInvalidLengthProtobuf
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.SmallStat == nil {
				m.SmallStat = &ProtStatSmall{}
			}
			if err := m.SmallStat.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProtobuf(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtobuf
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}
	if hasFields[0]&uint64(0x00000001) == 0 {
		return new(github_com_golang_protobuf_proto.RequiredNotSetError)
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ProtStats) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtobuf
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
			return fmt.Errorf("proto: ProtStats: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ProtStats: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Stats", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtobuf
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
				return ErrInvalidLengthProtobuf
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Stats = append(m.Stats, &ProtStat{})
			if err := m.Stats[len(m.Stats)-1].Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProtobuf(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtobuf
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipProtobuf(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowProtobuf
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
					return 0, ErrIntOverflowProtobuf
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
					return 0, ErrIntOverflowProtobuf
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
				return 0, ErrInvalidLengthProtobuf
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowProtobuf
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
				next, err := skipProtobuf(data[start:])
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
	ErrInvalidLengthProtobuf = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowProtobuf   = fmt.Errorf("proto: integer overflow")
)

var fileDescriptorProtobuf = []byte{
	// 240 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0x2b, 0x28, 0xca, 0x2f,
	0xc9, 0x4f, 0x2a, 0x4d, 0xd3, 0x03, 0x33, 0x84, 0xd8, 0x8a, 0x53, 0x8b, 0x32, 0x53, 0x8b, 0x95,
	0xf2, 0xb9, 0x78, 0x02, 0x80, 0x02, 0xc1, 0x25, 0x89, 0x25, 0x6e, 0xa5, 0x39, 0x39, 0x42, 0x3c,
	0x5c, 0x2c, 0x25, 0x99, 0xb9, 0xa9, 0x12, 0x8c, 0x0a, 0x4c, 0x1a, 0xcc, 0x42, 0xdc, 0x5c, 0xcc,
	0xb9, 0x99, 0x79, 0x12, 0x4c, 0x40, 0x0e, 0x23, 0x98, 0x93, 0x58, 0x21, 0xc1, 0x0c, 0xe3, 0x14,
	0x97, 0xe6, 0x4a, 0xb0, 0x80, 0x39, 0xbc, 0x5c, 0xac, 0x69, 0x99, 0x45, 0xc5, 0x25, 0x12, 0xac,
	0x60, 0x2e, 0xd0, 0x8c, 0x9c, 0x44, 0x20, 0x8f, 0x0d, 0x26, 0x99, 0x9c, 0x5f, 0x9a, 0x57, 0x22,
	0xc1, 0x0e, 0x32, 0x52, 0x49, 0x8b, 0x8b, 0x17, 0x66, 0x61, 0x70, 0x6e, 0x22, 0x36, 0x1b, 0xcb,
	0x12, 0x73, 0x20, 0x36, 0x2a, 0xe5, 0x70, 0x71, 0xc0, 0xd4, 0x0a, 0x09, 0x72, 0x71, 0x16, 0x03,
	0xe9, 0xf8, 0x92, 0xca, 0x02, 0x88, 0x5a, 0x0e, 0x21, 0x25, 0x2e, 0x16, 0x90, 0x10, 0x50, 0x31,
	0xa3, 0x06, 0xb7, 0x91, 0x88, 0x1e, 0xc4, 0x4b, 0x7a, 0x28, 0xfe, 0xd1, 0xe4, 0xe2, 0x2a, 0x06,
	0x59, 0x13, 0x0f, 0x56, 0xc9, 0x0c, 0x56, 0x29, 0x8a, 0xae, 0x12, 0xec, 0x10, 0x25, 0x1d, 0x2e,
	0x4e, 0x98, 0x40, 0xb1, 0x90, 0x3c, 0x17, 0x2b, 0x48, 0x47, 0x31, 0xd0, 0x70, 0x66, 0xa0, 0x16,
	0x01, 0x74, 0x2d, 0x4e, 0x02, 0x27, 0x1e, 0xc9, 0x31, 0x5e, 0x00, 0xe2, 0x07, 0x40, 0x3c, 0xe3,
	0xb1, 0x1c, 0x03, 0x20, 0x00, 0x00, 0xff, 0xff, 0xf2, 0x1f, 0xbb, 0x6c, 0x63, 0x01, 0x00, 0x00,
}
