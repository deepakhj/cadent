package metrics

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	repr "cadent/server/repr"

	_ "github.com/gogo/protobuf/gogoproto"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *DataPoint) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zxvk uint32
	zxvk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zxvk > 0 {
		zxvk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Value":
			z.Value, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z DataPoint) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Time"
	err = en.Append(0x82, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Time)
	if err != nil {
		return
	}
	// write "Value"
	err = en.Append(0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Value)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z DataPoint) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Time"
	o = append(o, 0x82, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendUint32(o, z.Time)
	// string "Value"
	o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendFloat64(o, z.Value)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DataPoint) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zbzg uint32
	zbzg, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zbzg > 0 {
		zbzg--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Value":
			z.Value, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z DataPoint) Msgsize() (s int) {
	s = 1 + 5 + msgp.Uint32Size + 6 + msgp.Float64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *DataPoints) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zcmr uint32
	zcmr, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zcmr > 0 {
		zcmr--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Points":
			var zajw uint32
			zajw, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Points) >= int(zajw) {
				z.Points = (z.Points)[:zajw]
			} else {
				z.Points = make([]*DataPoint, zajw)
			}
			for zbai := range z.Points {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Points[zbai] = nil
				} else {
					if z.Points[zbai] == nil {
						z.Points[zbai] = new(DataPoint)
					}
					var zwht uint32
					zwht, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zwht > 0 {
						zwht--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Time":
							z.Points[zbai].Time, err = dc.ReadUint32()
							if err != nil {
								return
							}
						case "Value":
							z.Points[zbai].Value, err = dc.ReadFloat64()
							if err != nil {
								return
							}
						default:
							err = dc.Skip()
							if err != nil {
								return
							}
						}
					}
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *DataPoints) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "Points"
	err = en.Append(0x81, 0xa6, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Points)))
	if err != nil {
		return
	}
	for zbai := range z.Points {
		if z.Points[zbai] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "Time"
			err = en.Append(0x82, 0xa4, 0x54, 0x69, 0x6d, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteUint32(z.Points[zbai].Time)
			if err != nil {
				return
			}
			// write "Value"
			err = en.Append(0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteFloat64(z.Points[zbai].Value)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *DataPoints) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Points"
	o = append(o, 0x81, 0xa6, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Points)))
	for zbai := range z.Points {
		if z.Points[zbai] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "Time"
			o = append(o, 0x82, 0xa4, 0x54, 0x69, 0x6d, 0x65)
			o = msgp.AppendUint32(o, z.Points[zbai].Time)
			// string "Value"
			o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
			o = msgp.AppendFloat64(o, z.Points[zbai].Value)
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DataPoints) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zhct uint32
	zhct, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zhct > 0 {
		zhct--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Points":
			var zcua uint32
			zcua, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Points) >= int(zcua) {
				z.Points = (z.Points)[:zcua]
			} else {
				z.Points = make([]*DataPoint, zcua)
			}
			for zbai := range z.Points {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Points[zbai] = nil
				} else {
					if z.Points[zbai] == nil {
						z.Points[zbai] = new(DataPoint)
					}
					var zxhx uint32
					zxhx, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zxhx > 0 {
						zxhx--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Time":
							z.Points[zbai].Time, bts, err = msgp.ReadUint32Bytes(bts)
							if err != nil {
								return
							}
						case "Value":
							z.Points[zbai].Value, bts, err = msgp.ReadFloat64Bytes(bts)
							if err != nil {
								return
							}
						default:
							bts, err = msgp.Skip(bts)
							if err != nil {
								return
							}
						}
					}
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *DataPoints) Msgsize() (s int) {
	s = 1 + 7 + msgp.ArrayHeaderSize
	for zbai := range z.Points {
		if z.Points[zbai] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 5 + msgp.Uint32Size + 6 + msgp.Float64Size
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawDataPoint) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zlqf uint32
	zlqf, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zlqf > 0 {
		zlqf--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Sum":
			z.Sum, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Min":
			z.Min, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Max":
			z.Max, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Last":
			z.Last, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Count":
			z.Count, err = dc.ReadInt64()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RawDataPoint) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "Time"
	err = en.Append(0x86, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Time)
	if err != nil {
		return
	}
	// write "Sum"
	err = en.Append(0xa3, 0x53, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Sum)
	if err != nil {
		return
	}
	// write "Min"
	err = en.Append(0xa3, 0x4d, 0x69, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Min)
	if err != nil {
		return
	}
	// write "Max"
	err = en.Append(0xa3, 0x4d, 0x61, 0x78)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Max)
	if err != nil {
		return
	}
	// write "Last"
	err = en.Append(0xa4, 0x4c, 0x61, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Last)
	if err != nil {
		return
	}
	// write "Count"
	err = en.Append(0xa5, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Count)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawDataPoint) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "Time"
	o = append(o, 0x86, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendUint32(o, z.Time)
	// string "Sum"
	o = append(o, 0xa3, 0x53, 0x75, 0x6d)
	o = msgp.AppendFloat64(o, z.Sum)
	// string "Min"
	o = append(o, 0xa3, 0x4d, 0x69, 0x6e)
	o = msgp.AppendFloat64(o, z.Min)
	// string "Max"
	o = append(o, 0xa3, 0x4d, 0x61, 0x78)
	o = msgp.AppendFloat64(o, z.Max)
	// string "Last"
	o = append(o, 0xa4, 0x4c, 0x61, 0x73, 0x74)
	o = msgp.AppendFloat64(o, z.Last)
	// string "Count"
	o = append(o, 0xa5, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	o = msgp.AppendInt64(o, z.Count)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawDataPoint) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zdaf uint32
	zdaf, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zdaf > 0 {
		zdaf--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Sum":
			z.Sum, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Min":
			z.Min, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Max":
			z.Max, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Last":
			z.Last, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Count":
			z.Count, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RawDataPoint) Msgsize() (s int) {
	s = 1 + 5 + msgp.Uint32Size + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 6 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawRenderItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zeff uint32
	zeff, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zeff > 0 {
		zeff--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Id":
			z.Id, err = dc.ReadString()
			if err != nil {
				return
			}
		case "RealStart":
			z.RealStart, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "RealEnd":
			z.RealEnd, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Start":
			z.Start, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "End":
			z.End, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Step":
			z.Step, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "AggFunc":
			z.AggFunc, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Tags":
			var zrsw uint32
			zrsw, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zrsw) {
				z.Tags = (z.Tags)[:zrsw]
			} else {
				z.Tags = make([]*repr.Tag, zrsw)
			}
			for zpks := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zpks] = nil
				} else {
					if z.Tags[zpks] == nil {
						z.Tags[zpks] = new(repr.Tag)
					}
					err = z.Tags[zpks].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zxpk uint32
			zxpk, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zxpk) {
				z.MetaTags = (z.MetaTags)[:zxpk]
			} else {
				z.MetaTags = make([]*repr.Tag, zxpk)
			}
			for zjfb := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zjfb] = nil
				} else {
					if z.MetaTags[zjfb] == nil {
						z.MetaTags[zjfb] = new(repr.Tag)
					}
					err = z.MetaTags[zjfb].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "Data":
			var zdnj uint32
			zdnj, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Data) >= int(zdnj) {
				z.Data = (z.Data)[:zdnj]
			} else {
				z.Data = make([]*RawDataPoint, zdnj)
			}
			for zcxo := range z.Data {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Data[zcxo] = nil
				} else {
					if z.Data[zcxo] == nil {
						z.Data[zcxo] = new(RawDataPoint)
					}
					err = z.Data[zcxo].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RawRenderItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 11
	// write "Metric"
	err = en.Append(0x8b, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
	if err != nil {
		return
	}
	// write "Id"
	err = en.Append(0xa2, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Id)
	if err != nil {
		return
	}
	// write "RealStart"
	err = en.Append(0xa9, 0x52, 0x65, 0x61, 0x6c, 0x53, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.RealStart)
	if err != nil {
		return
	}
	// write "RealEnd"
	err = en.Append(0xa7, 0x52, 0x65, 0x61, 0x6c, 0x45, 0x6e, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.RealEnd)
	if err != nil {
		return
	}
	// write "Start"
	err = en.Append(0xa5, 0x53, 0x74, 0x61, 0x72, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Start)
	if err != nil {
		return
	}
	// write "End"
	err = en.Append(0xa3, 0x45, 0x6e, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.End)
	if err != nil {
		return
	}
	// write "Step"
	err = en.Append(0xa4, 0x53, 0x74, 0x65, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Step)
	if err != nil {
		return
	}
	// write "AggFunc"
	err = en.Append(0xa7, 0x41, 0x67, 0x67, 0x46, 0x75, 0x6e, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.AggFunc)
	if err != nil {
		return
	}
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zpks := range z.Tags {
		if z.Tags[zpks] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zpks].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "MetaTags"
	err = en.Append(0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zjfb := range z.MetaTags {
		if z.MetaTags[zjfb] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zjfb].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	// write "Data"
	err = en.Append(0xa4, 0x44, 0x61, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Data)))
	if err != nil {
		return
	}
	for zcxo := range z.Data {
		if z.Data[zcxo] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Data[zcxo].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawRenderItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 11
	// string "Metric"
	o = append(o, 0x8b, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Id"
	o = append(o, 0xa2, 0x49, 0x64)
	o = msgp.AppendString(o, z.Id)
	// string "RealStart"
	o = append(o, 0xa9, 0x52, 0x65, 0x61, 0x6c, 0x53, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendUint32(o, z.RealStart)
	// string "RealEnd"
	o = append(o, 0xa7, 0x52, 0x65, 0x61, 0x6c, 0x45, 0x6e, 0x64)
	o = msgp.AppendUint32(o, z.RealEnd)
	// string "Start"
	o = append(o, 0xa5, 0x53, 0x74, 0x61, 0x72, 0x74)
	o = msgp.AppendUint32(o, z.Start)
	// string "End"
	o = append(o, 0xa3, 0x45, 0x6e, 0x64)
	o = msgp.AppendUint32(o, z.End)
	// string "Step"
	o = append(o, 0xa4, 0x53, 0x74, 0x65, 0x70)
	o = msgp.AppendUint32(o, z.Step)
	// string "AggFunc"
	o = append(o, 0xa7, 0x41, 0x67, 0x67, 0x46, 0x75, 0x6e, 0x63)
	o = msgp.AppendUint32(o, z.AggFunc)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zpks := range z.Tags {
		if z.Tags[zpks] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zpks].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zjfb := range z.MetaTags {
		if z.MetaTags[zjfb] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zjfb].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "Data"
	o = append(o, 0xa4, 0x44, 0x61, 0x74, 0x61)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Data)))
	for zcxo := range z.Data {
		if z.Data[zcxo] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Data[zcxo].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawRenderItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zobc uint32
	zobc, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zobc > 0 {
		zobc--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Id":
			z.Id, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "RealStart":
			z.RealStart, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "RealEnd":
			z.RealEnd, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Start":
			z.Start, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "End":
			z.End, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Step":
			z.Step, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "AggFunc":
			z.AggFunc, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zsnv uint32
			zsnv, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zsnv) {
				z.Tags = (z.Tags)[:zsnv]
			} else {
				z.Tags = make([]*repr.Tag, zsnv)
			}
			for zpks := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zpks] = nil
				} else {
					if z.Tags[zpks] == nil {
						z.Tags[zpks] = new(repr.Tag)
					}
					bts, err = z.Tags[zpks].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zkgt uint32
			zkgt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zkgt) {
				z.MetaTags = (z.MetaTags)[:zkgt]
			} else {
				z.MetaTags = make([]*repr.Tag, zkgt)
			}
			for zjfb := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zjfb] = nil
				} else {
					if z.MetaTags[zjfb] == nil {
						z.MetaTags[zjfb] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zjfb].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "Data":
			var zema uint32
			zema, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Data) >= int(zema) {
				z.Data = (z.Data)[:zema]
			} else {
				z.Data = make([]*RawDataPoint, zema)
			}
			for zcxo := range z.Data {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Data[zcxo] = nil
				} else {
					if z.Data[zcxo] == nil {
						z.Data[zcxo] = new(RawDataPoint)
					}
					bts, err = z.Data[zcxo].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RawRenderItem) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Metric) + 3 + msgp.StringPrefixSize + len(z.Id) + 10 + msgp.Uint32Size + 8 + msgp.Uint32Size + 6 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.Uint32Size + 8 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zpks := range z.Tags {
		if z.Tags[zpks] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zpks].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zjfb := range z.MetaTags {
		if z.MetaTags[zjfb] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zjfb].Msgsize()
		}
	}
	s += 5 + msgp.ArrayHeaderSize
	for zcxo := range z.Data {
		if z.Data[zcxo] == nil {
			s += msgp.NilSize
		} else {
			s += z.Data[zcxo].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RenderItem) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zqke uint32
	zqke, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zqke > 0 {
		zqke--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Target":
			z.Target, err = dc.ReadString()
			if err != nil {
				return
			}
		case "DataPoints":
			var zqyh uint32
			zqyh, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.DataPoints) >= int(zqyh) {
				z.DataPoints = (z.DataPoints)[:zqyh]
			} else {
				z.DataPoints = make([]*DataPoint, zqyh)
			}
			for zpez := range z.DataPoints {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.DataPoints[zpez] = nil
				} else {
					if z.DataPoints[zpez] == nil {
						z.DataPoints[zpez] = new(DataPoint)
					}
					var zyzr uint32
					zyzr, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zyzr > 0 {
						zyzr--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Time":
							z.DataPoints[zpez].Time, err = dc.ReadUint32()
							if err != nil {
								return
							}
						case "Value":
							z.DataPoints[zpez].Value, err = dc.ReadFloat64()
							if err != nil {
								return
							}
						default:
							err = dc.Skip()
							if err != nil {
								return
							}
						}
					}
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RenderItem) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Target"
	err = en.Append(0x82, 0xa6, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Target)
	if err != nil {
		return
	}
	// write "DataPoints"
	err = en.Append(0xaa, 0x44, 0x61, 0x74, 0x61, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.DataPoints)))
	if err != nil {
		return
	}
	for zpez := range z.DataPoints {
		if z.DataPoints[zpez] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 2
			// write "Time"
			err = en.Append(0x82, 0xa4, 0x54, 0x69, 0x6d, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteUint32(z.DataPoints[zpez].Time)
			if err != nil {
				return
			}
			// write "Value"
			err = en.Append(0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
			if err != nil {
				return err
			}
			err = en.WriteFloat64(z.DataPoints[zpez].Value)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RenderItem) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Target"
	o = append(o, 0x82, 0xa6, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74)
	o = msgp.AppendString(o, z.Target)
	// string "DataPoints"
	o = append(o, 0xaa, 0x44, 0x61, 0x74, 0x61, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.DataPoints)))
	for zpez := range z.DataPoints {
		if z.DataPoints[zpez] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 2
			// string "Time"
			o = append(o, 0x82, 0xa4, 0x54, 0x69, 0x6d, 0x65)
			o = msgp.AppendUint32(o, z.DataPoints[zpez].Time)
			// string "Value"
			o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
			o = msgp.AppendFloat64(o, z.DataPoints[zpez].Value)
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RenderItem) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zywj uint32
	zywj, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zywj > 0 {
		zywj--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Target":
			z.Target, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "DataPoints":
			var zjpj uint32
			zjpj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.DataPoints) >= int(zjpj) {
				z.DataPoints = (z.DataPoints)[:zjpj]
			} else {
				z.DataPoints = make([]*DataPoint, zjpj)
			}
			for zpez := range z.DataPoints {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.DataPoints[zpez] = nil
				} else {
					if z.DataPoints[zpez] == nil {
						z.DataPoints[zpez] = new(DataPoint)
					}
					var zzpf uint32
					zzpf, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zzpf > 0 {
						zzpf--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Time":
							z.DataPoints[zpez].Time, bts, err = msgp.ReadUint32Bytes(bts)
							if err != nil {
								return
							}
						case "Value":
							z.DataPoints[zpez].Value, bts, err = msgp.ReadFloat64Bytes(bts)
							if err != nil {
								return
							}
						default:
							bts, err = msgp.Skip(bts)
							if err != nil {
								return
							}
						}
					}
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RenderItem) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Target) + 11 + msgp.ArrayHeaderSize
	for zpez := range z.DataPoints {
		if z.DataPoints[zpez] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 5 + msgp.Uint32Size + 6 + msgp.Float64Size
		}
	}
	return
}
