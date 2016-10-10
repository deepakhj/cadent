package schemas

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	repr "cadent/server/repr"

	_ "github.com/gogo/protobuf/gogoproto"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *AnyMetric) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "Raw":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Raw = nil
			} else {
				if z.Raw == nil {
					z.Raw = new(RawMetric)
				}
				err = z.Raw.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Unprocessed":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Unprocessed = nil
			} else {
				if z.Unprocessed == nil {
					z.Unprocessed = new(UnProcessedMetric)
				}
				err = z.Unprocessed.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Single":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Single = nil
			} else {
				if z.Single == nil {
					z.Single = new(SingleMetric)
				}
				err = z.Single.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "Series":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Series = nil
			} else {
				if z.Series == nil {
					z.Series = new(SeriesMetric)
				}
				err = z.Series.DecodeMsg(dc)
				if err != nil {
					return
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
func (z *AnyMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 4
	// write "Raw"
	err = en.Append(0x84, 0xa3, 0x52, 0x61, 0x77)
	if err != nil {
		return err
	}
	if z.Raw == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Raw.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Unprocessed"
	err = en.Append(0xab, 0x55, 0x6e, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64)
	if err != nil {
		return err
	}
	if z.Unprocessed == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Unprocessed.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Single"
	err = en.Append(0xa6, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65)
	if err != nil {
		return err
	}
	if z.Single == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Single.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "Series"
	err = en.Append(0xa6, 0x53, 0x65, 0x72, 0x69, 0x65, 0x73)
	if err != nil {
		return err
	}
	if z.Series == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Series.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *AnyMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "Raw"
	o = append(o, 0x84, 0xa3, 0x52, 0x61, 0x77)
	if z.Raw == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Raw.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Unprocessed"
	o = append(o, 0xab, 0x55, 0x6e, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x64)
	if z.Unprocessed == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Unprocessed.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Single"
	o = append(o, 0xa6, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65)
	if z.Single == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Single.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "Series"
	o = append(o, 0xa6, 0x53, 0x65, 0x72, 0x69, 0x65, 0x73)
	if z.Series == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Series.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *AnyMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Raw":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Raw = nil
			} else {
				if z.Raw == nil {
					z.Raw = new(RawMetric)
				}
				bts, err = z.Raw.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Unprocessed":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Unprocessed = nil
			} else {
				if z.Unprocessed == nil {
					z.Unprocessed = new(UnProcessedMetric)
				}
				bts, err = z.Unprocessed.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Single":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Single = nil
			} else {
				if z.Single == nil {
					z.Single = new(SingleMetric)
				}
				bts, err = z.Single.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "Series":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Series = nil
			} else {
				if z.Series == nil {
					z.Series = new(SeriesMetric)
				}
				bts, err = z.Series.UnmarshalMsg(bts)
				if err != nil {
					return
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
func (z *AnyMetric) Msgsize() (s int) {
	s = 1 + 4
	if z.Raw == nil {
		s += msgp.NilSize
	} else {
		s += z.Raw.Msgsize()
	}
	s += 12
	if z.Unprocessed == nil {
		s += msgp.NilSize
	} else {
		s += z.Unprocessed.Msgsize()
	}
	s += 7
	if z.Single == nil {
		s += msgp.NilSize
	} else {
		s += z.Single.Msgsize()
	}
	s += 7
	if z.Series == nil {
		s += msgp.NilSize
	} else {
		s += z.Series.Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricName) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zajw uint32
	zajw, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zajw > 0 {
		zajw--
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
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Tags":
			var zwht uint32
			zwht, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zwht) {
				z.Tags = (z.Tags)[:zwht]
			} else {
				z.Tags = make([]*repr.Tag, zwht)
			}
			for zbai := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zbai] = nil
				} else {
					if z.Tags[zbai] == nil {
						z.Tags[zbai] = new(repr.Tag)
					}
					err = z.Tags[zbai].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zhct uint32
			zhct, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zhct) {
				z.MetaTags = (z.MetaTags)[:zhct]
			} else {
				z.MetaTags = make([]*repr.Tag, zhct)
			}
			for zcmr := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zcmr] = nil
				} else {
					if z.MetaTags[zcmr] == nil {
						z.MetaTags[zcmr] = new(repr.Tag)
					}
					err = z.MetaTags[zcmr].DecodeMsg(dc)
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
func (z *MetricName) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "Metric"
	err = en.Append(0x85, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
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
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "Uid"
	err = en.Append(0xa3, 0x55, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
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
	for zbai := range z.Tags {
		if z.Tags[zbai] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zbai].EncodeMsg(en)
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
	for zcmr := range z.MetaTags {
		if z.MetaTags[zcmr] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zcmr].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricName) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "Metric"
	o = append(o, 0x85, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Id"
	o = append(o, 0xa2, 0x49, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "Uid"
	o = append(o, 0xa3, 0x55, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zbai := range z.Tags {
		if z.Tags[zbai] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zbai].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zcmr := range z.MetaTags {
		if z.MetaTags[zcmr] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zcmr].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricName) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zcua uint32
	zcua, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zcua > 0 {
		zcua--
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
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zxhx uint32
			zxhx, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zxhx) {
				z.Tags = (z.Tags)[:zxhx]
			} else {
				z.Tags = make([]*repr.Tag, zxhx)
			}
			for zbai := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zbai] = nil
				} else {
					if z.Tags[zbai] == nil {
						z.Tags[zbai] = new(repr.Tag)
					}
					bts, err = z.Tags[zbai].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zlqf uint32
			zlqf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zlqf) {
				z.MetaTags = (z.MetaTags)[:zlqf]
			} else {
				z.MetaTags = make([]*repr.Tag, zlqf)
			}
			for zcmr := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zcmr] = nil
				} else {
					if z.MetaTags[zcmr] == nil {
						z.MetaTags[zcmr] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zcmr].UnmarshalMsg(bts)
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
func (z *MetricName) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Metric) + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.ArrayHeaderSize
	for zbai := range z.Tags {
		if z.Tags[zbai] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zbai].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zcmr := range z.MetaTags {
		if z.MetaTags[zcmr] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zcmr].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricType) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zdaf uint32
	zdaf, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zdaf > 0 {
		zdaf--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Type":
			z.Type, err = dc.ReadString()
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
func (z MetricType) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "Type"
	err = en.Append(0x81, 0xa4, 0x54, 0x79, 0x70, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Type)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z MetricType) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Type"
	o = append(o, 0x81, 0xa4, 0x54, 0x79, 0x70, 0x65)
	o = msgp.AppendString(o, z.Type)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricType) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zpks uint32
	zpks, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zpks > 0 {
		zpks--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Type":
			z.Type, bts, err = msgp.ReadStringBytes(bts)
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
func (z MetricType) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Type)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricValue) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zjfb uint32
	zjfb, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zjfb > 0 {
		zjfb--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, err = dc.ReadInt64()
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
		case "Sum":
			z.Sum, err = dc.ReadFloat64()
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
func (z *MetricValue) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "Time"
	err = en.Append(0x86, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
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
	// write "Sum"
	err = en.Append(0xa3, 0x53, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Sum)
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
func (z *MetricValue) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "Time"
	o = append(o, 0x86, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Min"
	o = append(o, 0xa3, 0x4d, 0x69, 0x6e)
	o = msgp.AppendFloat64(o, z.Min)
	// string "Max"
	o = append(o, 0xa3, 0x4d, 0x61, 0x78)
	o = msgp.AppendFloat64(o, z.Max)
	// string "Last"
	o = append(o, 0xa4, 0x4c, 0x61, 0x73, 0x74)
	o = msgp.AppendFloat64(o, z.Last)
	// string "Sum"
	o = append(o, 0xa3, 0x53, 0x75, 0x6d)
	o = msgp.AppendFloat64(o, z.Sum)
	// string "Count"
	o = append(o, 0xa5, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	o = msgp.AppendInt64(o, z.Count)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricValue) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zcxo uint32
	zcxo, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zcxo > 0 {
		zcxo--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
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
		case "Sum":
			z.Sum, bts, err = msgp.ReadFloat64Bytes(bts)
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
func (z *MetricValue) Msgsize() (s int) {
	s = 1 + 5 + msgp.Int64Size + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zxpk uint32
	zxpk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zxpk > 0 {
		zxpk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Value":
			z.Value, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Tags":
			var zdnj uint32
			zdnj, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zdnj) {
				z.Tags = (z.Tags)[:zdnj]
			} else {
				z.Tags = make([]*repr.Tag, zdnj)
			}
			for zeff := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zeff] = nil
				} else {
					if z.Tags[zeff] == nil {
						z.Tags[zeff] = new(repr.Tag)
					}
					err = z.Tags[zeff].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zobc uint32
			zobc, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zobc) {
				z.MetaTags = (z.MetaTags)[:zobc]
			} else {
				z.MetaTags = make([]*repr.Tag, zobc)
			}
			for zrsw := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zrsw] = nil
				} else {
					if z.MetaTags[zrsw] == nil {
						z.MetaTags[zrsw] = new(repr.Tag)
					}
					err = z.MetaTags[zrsw].DecodeMsg(dc)
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
func (z *RawMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "Time"
	err = en.Append(0x85, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
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
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zeff := range z.Tags {
		if z.Tags[zeff] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zeff].EncodeMsg(en)
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
	for zrsw := range z.MetaTags {
		if z.MetaTags[zrsw] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zrsw].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "Time"
	o = append(o, 0x85, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Value"
	o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendFloat64(o, z.Value)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zeff := range z.Tags {
		if z.Tags[zeff] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zeff].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zrsw := range z.MetaTags {
		if z.MetaTags[zrsw] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zrsw].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zsnv uint32
	zsnv, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zsnv > 0 {
		zsnv--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Value":
			z.Value, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zkgt uint32
			zkgt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zkgt) {
				z.Tags = (z.Tags)[:zkgt]
			} else {
				z.Tags = make([]*repr.Tag, zkgt)
			}
			for zeff := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zeff] = nil
				} else {
					if z.Tags[zeff] == nil {
						z.Tags[zeff] = new(repr.Tag)
					}
					bts, err = z.Tags[zeff].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zema uint32
			zema, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zema) {
				z.MetaTags = (z.MetaTags)[:zema]
			} else {
				z.MetaTags = make([]*repr.Tag, zema)
			}
			for zrsw := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zrsw] = nil
				} else {
					if z.MetaTags[zrsw] == nil {
						z.MetaTags[zrsw] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zrsw].UnmarshalMsg(bts)
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
func (z *RawMetric) Msgsize() (s int) {
	s = 1 + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 6 + msgp.Float64Size + 5 + msgp.ArrayHeaderSize
	for zeff := range z.Tags {
		if z.Tags[zeff] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zeff].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zrsw := range z.MetaTags {
		if z.MetaTags[zrsw] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zrsw].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SeriesMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zqyh uint32
	zqyh, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zqyh > 0 {
		zqyh--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Encoding":
			z.Encoding, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Data":
			z.Data, err = dc.ReadBytes(z.Data)
			if err != nil {
				return
			}
		case "Resolution":
			z.Resolution, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Ttl":
			z.Ttl, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Tags":
			var zyzr uint32
			zyzr, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zyzr) {
				z.Tags = (z.Tags)[:zyzr]
			} else {
				z.Tags = make([]*repr.Tag, zyzr)
			}
			for zpez := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zpez] = nil
				} else {
					if z.Tags[zpez] == nil {
						z.Tags[zpez] = new(repr.Tag)
					}
					err = z.Tags[zpez].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zywj uint32
			zywj, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zywj) {
				z.MetaTags = (z.MetaTags)[:zywj]
			} else {
				z.MetaTags = make([]*repr.Tag, zywj)
			}
			for zqke := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zqke] = nil
				} else {
					if z.MetaTags[zqke] == nil {
						z.MetaTags[zqke] = new(repr.Tag)
					}
					err = z.MetaTags[zqke].DecodeMsg(dc)
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
func (z *SeriesMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 10
	// write "Id"
	err = en.Append(0x8a, 0xa2, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "Uid"
	err = en.Append(0xa3, 0x55, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
	}
	// write "Time"
	err = en.Append(0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
	if err != nil {
		return
	}
	// write "Encoding"
	err = en.Append(0xa8, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Encoding)
	if err != nil {
		return
	}
	// write "Data"
	err = en.Append(0xa4, 0x44, 0x61, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteBytes(z.Data)
	if err != nil {
		return
	}
	// write "Resolution"
	err = en.Append(0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Resolution)
	if err != nil {
		return
	}
	// write "Ttl"
	err = en.Append(0xa3, 0x54, 0x74, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Ttl)
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
	for zpez := range z.Tags {
		if z.Tags[zpez] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zpez].EncodeMsg(en)
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
	for zqke := range z.MetaTags {
		if z.MetaTags[zqke] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zqke].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SeriesMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 10
	// string "Id"
	o = append(o, 0x8a, 0xa2, 0x49, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "Uid"
	o = append(o, 0xa3, 0x55, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "Time"
	o = append(o, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Encoding"
	o = append(o, 0xa8, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67)
	o = msgp.AppendString(o, z.Encoding)
	// string "Data"
	o = append(o, 0xa4, 0x44, 0x61, 0x74, 0x61)
	o = msgp.AppendBytes(o, z.Data)
	// string "Resolution"
	o = append(o, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "Ttl"
	o = append(o, 0xa3, 0x54, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.Ttl)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zpez := range z.Tags {
		if z.Tags[zpez] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zpez].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zqke := range z.MetaTags {
		if z.MetaTags[zqke] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zqke].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SeriesMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zjpj uint32
	zjpj, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zjpj > 0 {
		zjpj--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Encoding":
			z.Encoding, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Data":
			z.Data, bts, err = msgp.ReadBytesBytes(bts, z.Data)
			if err != nil {
				return
			}
		case "Resolution":
			z.Resolution, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Ttl":
			z.Ttl, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zzpf uint32
			zzpf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zzpf) {
				z.Tags = (z.Tags)[:zzpf]
			} else {
				z.Tags = make([]*repr.Tag, zzpf)
			}
			for zpez := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zpez] = nil
				} else {
					if z.Tags[zpez] == nil {
						z.Tags[zpez] = new(repr.Tag)
					}
					bts, err = z.Tags[zpez].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zrfe uint32
			zrfe, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zrfe) {
				z.MetaTags = (z.MetaTags)[:zrfe]
			} else {
				z.MetaTags = make([]*repr.Tag, zrfe)
			}
			for zqke := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zqke] = nil
				} else {
					if z.MetaTags[zqke] == nil {
						z.MetaTags[zqke] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zqke].UnmarshalMsg(bts)
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
func (z *SeriesMetric) Msgsize() (s int) {
	s = 1 + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 9 + msgp.StringPrefixSize + len(z.Encoding) + 5 + msgp.BytesPrefixSize + len(z.Data) + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zpez := range z.Tags {
		if z.Tags[zpez] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zpez].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zqke := range z.MetaTags {
		if z.MetaTags[zqke] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zqke].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SingleMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zeth uint32
	zeth, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zeth > 0 {
		zeth--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
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
		case "Sum":
			z.Sum, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Count":
			z.Count, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Resolution":
			z.Resolution, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Ttl":
			z.Ttl, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "Tags":
			var zsbz uint32
			zsbz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zsbz) {
				z.Tags = (z.Tags)[:zsbz]
			} else {
				z.Tags = make([]*repr.Tag, zsbz)
			}
			for zgmo := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zgmo] = nil
				} else {
					if z.Tags[zgmo] == nil {
						z.Tags[zgmo] = new(repr.Tag)
					}
					err = z.Tags[zgmo].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zrjx uint32
			zrjx, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zrjx) {
				z.MetaTags = (z.MetaTags)[:zrjx]
			} else {
				z.MetaTags = make([]*repr.Tag, zrjx)
			}
			for ztaf := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[ztaf] = nil
				} else {
					if z.MetaTags[ztaf] == nil {
						z.MetaTags[ztaf] = new(repr.Tag)
					}
					err = z.MetaTags[ztaf].DecodeMsg(dc)
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
func (z *SingleMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 13
	// write "Id"
	err = en.Append(0x8d, 0xa2, 0x49, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "Uid"
	err = en.Append(0xa3, 0x55, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
	}
	// write "Time"
	err = en.Append(0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
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
	// write "Sum"
	err = en.Append(0xa3, 0x53, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Sum)
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
	// write "Resolution"
	err = en.Append(0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Resolution)
	if err != nil {
		return
	}
	// write "Ttl"
	err = en.Append(0xa3, 0x54, 0x74, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Ttl)
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
	for zgmo := range z.Tags {
		if z.Tags[zgmo] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zgmo].EncodeMsg(en)
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
	for ztaf := range z.MetaTags {
		if z.MetaTags[ztaf] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[ztaf].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SingleMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 13
	// string "Id"
	o = append(o, 0x8d, 0xa2, 0x49, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "Uid"
	o = append(o, 0xa3, 0x55, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "Time"
	o = append(o, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Min"
	o = append(o, 0xa3, 0x4d, 0x69, 0x6e)
	o = msgp.AppendFloat64(o, z.Min)
	// string "Max"
	o = append(o, 0xa3, 0x4d, 0x61, 0x78)
	o = msgp.AppendFloat64(o, z.Max)
	// string "Last"
	o = append(o, 0xa4, 0x4c, 0x61, 0x73, 0x74)
	o = msgp.AppendFloat64(o, z.Last)
	// string "Sum"
	o = append(o, 0xa3, 0x53, 0x75, 0x6d)
	o = msgp.AppendFloat64(o, z.Sum)
	// string "Count"
	o = append(o, 0xa5, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	o = msgp.AppendInt64(o, z.Count)
	// string "Resolution"
	o = append(o, 0xaa, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "Ttl"
	o = append(o, 0xa3, 0x54, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.Ttl)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zgmo := range z.Tags {
		if z.Tags[zgmo] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zgmo].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for ztaf := range z.MetaTags {
		if z.MetaTags[ztaf] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[ztaf].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SingleMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zawn uint32
	zawn, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zawn > 0 {
		zawn--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Id":
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "Uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
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
		case "Sum":
			z.Sum, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Count":
			z.Count, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Resolution":
			z.Resolution, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Ttl":
			z.Ttl, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var zwel uint32
			zwel, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zwel) {
				z.Tags = (z.Tags)[:zwel]
			} else {
				z.Tags = make([]*repr.Tag, zwel)
			}
			for zgmo := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zgmo] = nil
				} else {
					if z.Tags[zgmo] == nil {
						z.Tags[zgmo] = new(repr.Tag)
					}
					bts, err = z.Tags[zgmo].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zrbe uint32
			zrbe, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zrbe) {
				z.MetaTags = (z.MetaTags)[:zrbe]
			} else {
				z.MetaTags = make([]*repr.Tag, zrbe)
			}
			for ztaf := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[ztaf] = nil
				} else {
					if z.MetaTags[ztaf] == nil {
						z.MetaTags[ztaf] = new(repr.Tag)
					}
					bts, err = z.MetaTags[ztaf].UnmarshalMsg(bts)
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
func (z *SingleMetric) Msgsize() (s int) {
	s = 1 + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zgmo := range z.Tags {
		if z.Tags[zgmo] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zgmo].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for ztaf := range z.MetaTags {
		if z.MetaTags[ztaf] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[ztaf].Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UnProcessedMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zelx uint32
	zelx, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zelx > 0 {
		zelx--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, err = dc.ReadString()
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
		case "Sum":
			z.Sum, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "Count":
			z.Count, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "Tags":
			var zbal uint32
			zbal, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zbal) {
				z.Tags = (z.Tags)[:zbal]
			} else {
				z.Tags = make([]*repr.Tag, zbal)
			}
			for zmfd := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zmfd] = nil
				} else {
					if z.Tags[zmfd] == nil {
						z.Tags[zmfd] = new(repr.Tag)
					}
					err = z.Tags[zmfd].DecodeMsg(dc)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var zjqz uint32
			zjqz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zjqz) {
				z.MetaTags = (z.MetaTags)[:zjqz]
			} else {
				z.MetaTags = make([]*repr.Tag, zjqz)
			}
			for zzdc := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zzdc] = nil
				} else {
					if z.MetaTags[zzdc] == nil {
						z.MetaTags[zzdc] = new(repr.Tag)
					}
					err = z.MetaTags[zzdc].DecodeMsg(dc)
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
func (z *UnProcessedMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 9
	// write "Time"
	err = en.Append(0x89, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "Metric"
	err = en.Append(0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
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
	// write "Sum"
	err = en.Append(0xa3, 0x53, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Sum)
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
	// write "Tags"
	err = en.Append(0xa4, 0x54, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zmfd := range z.Tags {
		if z.Tags[zmfd] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Tags[zmfd].EncodeMsg(en)
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
	for zzdc := range z.MetaTags {
		if z.MetaTags[zzdc] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.MetaTags[zzdc].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *UnProcessedMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "Time"
	o = append(o, 0x89, 0xa4, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "Metric"
	o = append(o, 0xa6, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "Min"
	o = append(o, 0xa3, 0x4d, 0x69, 0x6e)
	o = msgp.AppendFloat64(o, z.Min)
	// string "Max"
	o = append(o, 0xa3, 0x4d, 0x61, 0x78)
	o = msgp.AppendFloat64(o, z.Max)
	// string "Last"
	o = append(o, 0xa4, 0x4c, 0x61, 0x73, 0x74)
	o = msgp.AppendFloat64(o, z.Last)
	// string "Sum"
	o = append(o, 0xa3, 0x53, 0x75, 0x6d)
	o = msgp.AppendFloat64(o, z.Sum)
	// string "Count"
	o = append(o, 0xa5, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	o = msgp.AppendInt64(o, z.Count)
	// string "Tags"
	o = append(o, 0xa4, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zmfd := range z.Tags {
		if z.Tags[zmfd] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Tags[zmfd].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zzdc := range z.MetaTags {
		if z.MetaTags[zzdc] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.MetaTags[zzdc].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UnProcessedMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zkct uint32
	zkct, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zkct > 0 {
		zkct--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
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
		case "Sum":
			z.Sum, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "Count":
			z.Count, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "Tags":
			var ztmt uint32
			ztmt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(ztmt) {
				z.Tags = (z.Tags)[:ztmt]
			} else {
				z.Tags = make([]*repr.Tag, ztmt)
			}
			for zmfd := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zmfd] = nil
				} else {
					if z.Tags[zmfd] == nil {
						z.Tags[zmfd] = new(repr.Tag)
					}
					bts, err = z.Tags[zmfd].UnmarshalMsg(bts)
					if err != nil {
						return
					}
				}
			}
		case "MetaTags":
			var ztco uint32
			ztco, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(ztco) {
				z.MetaTags = (z.MetaTags)[:ztco]
			} else {
				z.MetaTags = make([]*repr.Tag, ztco)
			}
			for zzdc := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zzdc] = nil
				} else {
					if z.MetaTags[zzdc] == nil {
						z.MetaTags[zzdc] = new(repr.Tag)
					}
					bts, err = z.MetaTags[zzdc].UnmarshalMsg(bts)
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
func (z *UnProcessedMetric) Msgsize() (s int) {
	s = 1 + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size + 5 + msgp.ArrayHeaderSize
	for zmfd := range z.Tags {
		if z.Tags[zmfd] == nil {
			s += msgp.NilSize
		} else {
			s += z.Tags[zmfd].Msgsize()
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zzdc := range z.MetaTags {
		if z.MetaTags[zzdc] == nil {
			s += msgp.NilSize
		} else {
			s += z.MetaTags[zzdc].Msgsize()
		}
	}
	return
}
