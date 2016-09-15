package schemas

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *KafkaEncodingType) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var zxvk uint8
		zxvk, err = dc.ReadUint8()
		(*z) = KafkaEncodingType(zxvk)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z KafkaEncodingType) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteUint8(uint8(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z KafkaEncodingType) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendUint8(o, uint8(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *KafkaEncodingType) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zbzg uint8
		zbzg, bts, err = msgp.ReadUint8Bytes(bts)
		(*z) = KafkaEncodingType(zbzg)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z KafkaEncodingType) Msgsize() (s int) {
	s = msgp.Uint8Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *KafkaSeriesMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zhct uint32
	zhct, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zhct > 0 {
		zhct--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "type":
			z.Type, err = dc.ReadString()
			if err != nil {
				return
			}
		case "id":
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "encoding":
			z.Encoding, err = dc.ReadString()
			if err != nil {
				return
			}
		case "data":
			z.Data, err = dc.ReadBytes(z.Data)
			if err != nil {
				return
			}
		case "resolution":
			z.Resolution, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "ttl":
			z.TTL, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "tags":
			var zcua uint32
			zcua, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zcua) {
				z.Tags = (z.Tags)[:zcua]
			} else {
				z.Tags = make([][]string, zcua)
			}
			for zbai := range z.Tags {
				var zxhx uint32
				zxhx, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.Tags[zbai]) >= int(zxhx) {
					z.Tags[zbai] = (z.Tags[zbai])[:zxhx]
				} else {
					z.Tags[zbai] = make([]string, zxhx)
				}
				for zcmr := range z.Tags[zbai] {
					z.Tags[zbai][zcmr], err = dc.ReadString()
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zlqf uint32
			zlqf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zlqf) {
				z.MetaTags = (z.MetaTags)[:zlqf]
			} else {
				z.MetaTags = make([][]string, zlqf)
			}
			for zajw := range z.MetaTags {
				var zdaf uint32
				zdaf, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.MetaTags[zajw]) >= int(zdaf) {
					z.MetaTags[zajw] = (z.MetaTags[zajw])[:zdaf]
				} else {
					z.MetaTags[zajw] = make([]string, zdaf)
				}
				for zwht := range z.MetaTags[zajw] {
					z.MetaTags[zajw][zwht], err = dc.ReadString()
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
func (z *KafkaSeriesMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 11
	// write "type"
	err = en.Append(0x8b, 0xa4, 0x74, 0x79, 0x70, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Type)
	if err != nil {
		return
	}
	// write "id"
	err = en.Append(0xa2, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "uid"
	err = en.Append(0xa3, 0x75, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
	}
	// write "time"
	err = en.Append(0xa4, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "metric"
	err = en.Append(0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
	if err != nil {
		return
	}
	// write "encoding"
	err = en.Append(0xa8, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Encoding)
	if err != nil {
		return
	}
	// write "data"
	err = en.Append(0xa4, 0x64, 0x61, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteBytes(z.Data)
	if err != nil {
		return
	}
	// write "resolution"
	err = en.Append(0xaa, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Resolution)
	if err != nil {
		return
	}
	// write "ttl"
	err = en.Append(0xa3, 0x74, 0x74, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.TTL)
	if err != nil {
		return
	}
	// write "tags"
	err = en.Append(0xa4, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zbai := range z.Tags {
		err = en.WriteArrayHeader(uint32(len(z.Tags[zbai])))
		if err != nil {
			return
		}
		for zcmr := range z.Tags[zbai] {
			err = en.WriteString(z.Tags[zbai][zcmr])
			if err != nil {
				return
			}
		}
	}
	// write "metatags"
	err = en.Append(0xa8, 0x6d, 0x65, 0x74, 0x61, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zajw := range z.MetaTags {
		err = en.WriteArrayHeader(uint32(len(z.MetaTags[zajw])))
		if err != nil {
			return
		}
		for zwht := range z.MetaTags[zajw] {
			err = en.WriteString(z.MetaTags[zajw][zwht])
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *KafkaSeriesMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 11
	// string "type"
	o = append(o, 0x8b, 0xa4, 0x74, 0x79, 0x70, 0x65)
	o = msgp.AppendString(o, z.Type)
	// string "id"
	o = append(o, 0xa2, 0x69, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "uid"
	o = append(o, 0xa3, 0x75, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "time"
	o = append(o, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "metric"
	o = append(o, 0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "encoding"
	o = append(o, 0xa8, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67)
	o = msgp.AppendString(o, z.Encoding)
	// string "data"
	o = append(o, 0xa4, 0x64, 0x61, 0x74, 0x61)
	o = msgp.AppendBytes(o, z.Data)
	// string "resolution"
	o = append(o, 0xaa, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "ttl"
	o = append(o, 0xa3, 0x74, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.TTL)
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zbai := range z.Tags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zbai])))
		for zcmr := range z.Tags[zbai] {
			o = msgp.AppendString(o, z.Tags[zbai][zcmr])
		}
	}
	// string "metatags"
	o = append(o, 0xa8, 0x6d, 0x65, 0x74, 0x61, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zajw := range z.MetaTags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zajw])))
		for zwht := range z.MetaTags[zajw] {
			o = msgp.AppendString(o, z.MetaTags[zajw][zwht])
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *KafkaSeriesMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "type":
			z.Type, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "id":
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "encoding":
			z.Encoding, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "data":
			z.Data, bts, err = msgp.ReadBytesBytes(bts, z.Data)
			if err != nil {
				return
			}
		case "resolution":
			z.Resolution, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "ttl":
			z.TTL, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "tags":
			var zjfb uint32
			zjfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zjfb) {
				z.Tags = (z.Tags)[:zjfb]
			} else {
				z.Tags = make([][]string, zjfb)
			}
			for zbai := range z.Tags {
				var zcxo uint32
				zcxo, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.Tags[zbai]) >= int(zcxo) {
					z.Tags[zbai] = (z.Tags[zbai])[:zcxo]
				} else {
					z.Tags[zbai] = make([]string, zcxo)
				}
				for zcmr := range z.Tags[zbai] {
					z.Tags[zbai][zcmr], bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zeff uint32
			zeff, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zeff) {
				z.MetaTags = (z.MetaTags)[:zeff]
			} else {
				z.MetaTags = make([][]string, zeff)
			}
			for zajw := range z.MetaTags {
				var zrsw uint32
				zrsw, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.MetaTags[zajw]) >= int(zrsw) {
					z.MetaTags[zajw] = (z.MetaTags[zajw])[:zrsw]
				} else {
					z.MetaTags[zajw] = make([]string, zrsw)
				}
				for zwht := range z.MetaTags[zajw] {
					z.MetaTags[zajw][zwht], bts, err = msgp.ReadStringBytes(bts)
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
func (z *KafkaSeriesMetric) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Type) + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 9 + msgp.StringPrefixSize + len(z.Encoding) + 5 + msgp.BytesPrefixSize + len(z.Data) + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zbai := range z.Tags {
		s += msgp.ArrayHeaderSize
		for zcmr := range z.Tags[zbai] {
			s += msgp.StringPrefixSize + len(z.Tags[zbai][zcmr])
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zajw := range z.MetaTags {
		s += msgp.ArrayHeaderSize
		for zwht := range z.MetaTags[zajw] {
			s += msgp.StringPrefixSize + len(z.MetaTags[zajw][zwht])
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *KafkaSingleMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zkgt uint32
	zkgt, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zkgt > 0 {
		zkgt--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "type":
			z.Type, err = dc.ReadString()
			if err != nil {
				return
			}
		case "id":
			z.Id, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "uid":
			z.Uid, err = dc.ReadString()
			if err != nil {
				return
			}
		case "time":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "metric":
			z.Metric, err = dc.ReadString()
			if err != nil {
				return
			}
		case "min":
			z.Min, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "max":
			z.Max, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "last":
			z.Last, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "sum":
			z.Sum, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "count":
			z.Count, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "resolution":
			z.Resolution, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "ttl":
			z.TTL, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "tags":
			var zema uint32
			zema, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zema) {
				z.Tags = (z.Tags)[:zema]
			} else {
				z.Tags = make([][]string, zema)
			}
			for zxpk := range z.Tags {
				var zpez uint32
				zpez, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.Tags[zxpk]) >= int(zpez) {
					z.Tags[zxpk] = (z.Tags[zxpk])[:zpez]
				} else {
					z.Tags[zxpk] = make([]string, zpez)
				}
				for zdnj := range z.Tags[zxpk] {
					z.Tags[zxpk][zdnj], err = dc.ReadString()
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zqke uint32
			zqke, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zqke) {
				z.MetaTags = (z.MetaTags)[:zqke]
			} else {
				z.MetaTags = make([][]string, zqke)
			}
			for zobc := range z.MetaTags {
				var zqyh uint32
				zqyh, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.MetaTags[zobc]) >= int(zqyh) {
					z.MetaTags[zobc] = (z.MetaTags[zobc])[:zqyh]
				} else {
					z.MetaTags[zobc] = make([]string, zqyh)
				}
				for zsnv := range z.MetaTags[zobc] {
					z.MetaTags[zobc][zsnv], err = dc.ReadString()
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
func (z *KafkaSingleMetric) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 14
	// write "type"
	err = en.Append(0x8e, 0xa4, 0x74, 0x79, 0x70, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Type)
	if err != nil {
		return
	}
	// write "id"
	err = en.Append(0xa2, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteUint64(z.Id)
	if err != nil {
		return
	}
	// write "uid"
	err = en.Append(0xa3, 0x75, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Uid)
	if err != nil {
		return
	}
	// write "time"
	err = en.Append(0xa4, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "metric"
	err = en.Append(0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
	if err != nil {
		return
	}
	// write "min"
	err = en.Append(0xa3, 0x6d, 0x69, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Min)
	if err != nil {
		return
	}
	// write "max"
	err = en.Append(0xa3, 0x6d, 0x61, 0x78)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Max)
	if err != nil {
		return
	}
	// write "last"
	err = en.Append(0xa4, 0x6c, 0x61, 0x73, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Last)
	if err != nil {
		return
	}
	// write "sum"
	err = en.Append(0xa3, 0x73, 0x75, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Sum)
	if err != nil {
		return
	}
	// write "count"
	err = en.Append(0xa5, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Count)
	if err != nil {
		return
	}
	// write "resolution"
	err = en.Append(0xaa, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.Resolution)
	if err != nil {
		return
	}
	// write "ttl"
	err = en.Append(0xa3, 0x74, 0x74, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteUint32(z.TTL)
	if err != nil {
		return
	}
	// write "tags"
	err = en.Append(0xa4, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zxpk := range z.Tags {
		err = en.WriteArrayHeader(uint32(len(z.Tags[zxpk])))
		if err != nil {
			return
		}
		for zdnj := range z.Tags[zxpk] {
			err = en.WriteString(z.Tags[zxpk][zdnj])
			if err != nil {
				return
			}
		}
	}
	// write "metatags"
	err = en.Append(0xa8, 0x6d, 0x65, 0x74, 0x61, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zobc := range z.MetaTags {
		err = en.WriteArrayHeader(uint32(len(z.MetaTags[zobc])))
		if err != nil {
			return
		}
		for zsnv := range z.MetaTags[zobc] {
			err = en.WriteString(z.MetaTags[zobc][zsnv])
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *KafkaSingleMetric) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 14
	// string "type"
	o = append(o, 0x8e, 0xa4, 0x74, 0x79, 0x70, 0x65)
	o = msgp.AppendString(o, z.Type)
	// string "id"
	o = append(o, 0xa2, 0x69, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "uid"
	o = append(o, 0xa3, 0x75, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "time"
	o = append(o, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "metric"
	o = append(o, 0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "min"
	o = append(o, 0xa3, 0x6d, 0x69, 0x6e)
	o = msgp.AppendFloat64(o, z.Min)
	// string "max"
	o = append(o, 0xa3, 0x6d, 0x61, 0x78)
	o = msgp.AppendFloat64(o, z.Max)
	// string "last"
	o = append(o, 0xa4, 0x6c, 0x61, 0x73, 0x74)
	o = msgp.AppendFloat64(o, z.Last)
	// string "sum"
	o = append(o, 0xa3, 0x73, 0x75, 0x6d)
	o = msgp.AppendFloat64(o, z.Sum)
	// string "count"
	o = append(o, 0xa5, 0x63, 0x6f, 0x75, 0x6e, 0x74)
	o = msgp.AppendInt64(o, z.Count)
	// string "resolution"
	o = append(o, 0xaa, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendUint32(o, z.Resolution)
	// string "ttl"
	o = append(o, 0xa3, 0x74, 0x74, 0x6c)
	o = msgp.AppendUint32(o, z.TTL)
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zxpk := range z.Tags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zxpk])))
		for zdnj := range z.Tags[zxpk] {
			o = msgp.AppendString(o, z.Tags[zxpk][zdnj])
		}
	}
	// string "metatags"
	o = append(o, 0xa8, 0x6d, 0x65, 0x74, 0x61, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zobc := range z.MetaTags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zobc])))
		for zsnv := range z.MetaTags[zobc] {
			o = msgp.AppendString(o, z.MetaTags[zobc][zsnv])
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *KafkaSingleMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zyzr uint32
	zyzr, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zyzr > 0 {
		zyzr--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "type":
			z.Type, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "id":
			z.Id, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "uid":
			z.Uid, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "min":
			z.Min, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "max":
			z.Max, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "last":
			z.Last, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "sum":
			z.Sum, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "count":
			z.Count, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "resolution":
			z.Resolution, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "ttl":
			z.TTL, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "tags":
			var zywj uint32
			zywj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zywj) {
				z.Tags = (z.Tags)[:zywj]
			} else {
				z.Tags = make([][]string, zywj)
			}
			for zxpk := range z.Tags {
				var zjpj uint32
				zjpj, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.Tags[zxpk]) >= int(zjpj) {
					z.Tags[zxpk] = (z.Tags[zxpk])[:zjpj]
				} else {
					z.Tags[zxpk] = make([]string, zjpj)
				}
				for zdnj := range z.Tags[zxpk] {
					z.Tags[zxpk][zdnj], bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zzpf uint32
			zzpf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zzpf) {
				z.MetaTags = (z.MetaTags)[:zzpf]
			} else {
				z.MetaTags = make([][]string, zzpf)
			}
			for zobc := range z.MetaTags {
				var zrfe uint32
				zrfe, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.MetaTags[zobc]) >= int(zrfe) {
					z.MetaTags[zobc] = (z.MetaTags[zobc])[:zrfe]
				} else {
					z.MetaTags[zobc] = make([]string, zrfe)
				}
				for zsnv := range z.MetaTags[zobc] {
					z.MetaTags[zobc][zsnv], bts, err = msgp.ReadStringBytes(bts)
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
func (z *KafkaSingleMetric) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Type) + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zxpk := range z.Tags {
		s += msgp.ArrayHeaderSize
		for zdnj := range z.Tags[zxpk] {
			s += msgp.StringPrefixSize + len(z.Tags[zxpk][zdnj])
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zobc := range z.MetaTags {
		s += msgp.ArrayHeaderSize
		for zsnv := range z.MetaTags[zobc] {
			s += msgp.StringPrefixSize + len(z.MetaTags[zobc][zsnv])
		}
	}
	return
}
