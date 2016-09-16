package schemas

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *SeriesMetric) DecodeMsg(dc *msgp.Reader) (err error) {
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
			var zwht uint32
			zwht, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zwht) {
				z.Tags = (z.Tags)[:zwht]
			} else {
				z.Tags = make([][]string, zwht)
			}
			for zxvk := range z.Tags {
				var zhct uint32
				zhct, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.Tags[zxvk]) >= int(zhct) {
					z.Tags[zxvk] = (z.Tags[zxvk])[:zhct]
				} else {
					z.Tags[zxvk] = make([]string, zhct)
				}
				for zbzg := range z.Tags[zxvk] {
					z.Tags[zxvk][zbzg], err = dc.ReadString()
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zcua uint32
			zcua, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zcua) {
				z.MetaTags = (z.MetaTags)[:zcua]
			} else {
				z.MetaTags = make([][]string, zcua)
			}
			for zbai := range z.MetaTags {
				var zxhx uint32
				zxhx, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.MetaTags[zbai]) >= int(zxhx) {
					z.MetaTags[zbai] = (z.MetaTags[zbai])[:zxhx]
				} else {
					z.MetaTags[zbai] = make([]string, zxhx)
				}
				for zcmr := range z.MetaTags[zbai] {
					z.MetaTags[zbai][zcmr], err = dc.ReadString()
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
	for zxvk := range z.Tags {
		err = en.WriteArrayHeader(uint32(len(z.Tags[zxvk])))
		if err != nil {
			return
		}
		for zbzg := range z.Tags[zxvk] {
			err = en.WriteString(z.Tags[zxvk][zbzg])
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
	for zbai := range z.MetaTags {
		err = en.WriteArrayHeader(uint32(len(z.MetaTags[zbai])))
		if err != nil {
			return
		}
		for zcmr := range z.MetaTags[zbai] {
			err = en.WriteString(z.MetaTags[zbai][zcmr])
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
	for zxvk := range z.Tags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zxvk])))
		for zbzg := range z.Tags[zxvk] {
			o = msgp.AppendString(o, z.Tags[zxvk][zbzg])
		}
	}
	// string "metatags"
	o = append(o, 0xa8, 0x6d, 0x65, 0x74, 0x61, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zbai := range z.MetaTags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zbai])))
		for zcmr := range z.MetaTags[zbai] {
			o = msgp.AppendString(o, z.MetaTags[zbai][zcmr])
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SeriesMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zlqf uint32
	zlqf, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zlqf > 0 {
		zlqf--
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
			var zdaf uint32
			zdaf, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zdaf) {
				z.Tags = (z.Tags)[:zdaf]
			} else {
				z.Tags = make([][]string, zdaf)
			}
			for zxvk := range z.Tags {
				var zpks uint32
				zpks, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.Tags[zxvk]) >= int(zpks) {
					z.Tags[zxvk] = (z.Tags[zxvk])[:zpks]
				} else {
					z.Tags[zxvk] = make([]string, zpks)
				}
				for zbzg := range z.Tags[zxvk] {
					z.Tags[zxvk][zbzg], bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zjfb uint32
			zjfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zjfb) {
				z.MetaTags = (z.MetaTags)[:zjfb]
			} else {
				z.MetaTags = make([][]string, zjfb)
			}
			for zbai := range z.MetaTags {
				var zcxo uint32
				zcxo, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.MetaTags[zbai]) >= int(zcxo) {
					z.MetaTags[zbai] = (z.MetaTags[zbai])[:zcxo]
				} else {
					z.MetaTags[zbai] = make([]string, zcxo)
				}
				for zcmr := range z.MetaTags[zbai] {
					z.MetaTags[zbai][zcmr], bts, err = msgp.ReadStringBytes(bts)
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
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Type) + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 9 + msgp.StringPrefixSize + len(z.Encoding) + 5 + msgp.BytesPrefixSize + len(z.Data) + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zxvk := range z.Tags {
		s += msgp.ArrayHeaderSize
		for zbzg := range z.Tags[zxvk] {
			s += msgp.StringPrefixSize + len(z.Tags[zxvk][zbzg])
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zbai := range z.MetaTags {
		s += msgp.ArrayHeaderSize
		for zcmr := range z.MetaTags[zbai] {
			s += msgp.StringPrefixSize + len(z.MetaTags[zbai][zcmr])
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SingleMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zobc uint32
	zobc, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zobc > 0 {
		zobc--
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
			var zsnv uint32
			zsnv, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zsnv) {
				z.Tags = (z.Tags)[:zsnv]
			} else {
				z.Tags = make([][]string, zsnv)
			}
			for zeff := range z.Tags {
				var zkgt uint32
				zkgt, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.Tags[zeff]) >= int(zkgt) {
					z.Tags[zeff] = (z.Tags[zeff])[:zkgt]
				} else {
					z.Tags[zeff] = make([]string, zkgt)
				}
				for zrsw := range z.Tags[zeff] {
					z.Tags[zeff][zrsw], err = dc.ReadString()
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zema uint32
			zema, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zema) {
				z.MetaTags = (z.MetaTags)[:zema]
			} else {
				z.MetaTags = make([][]string, zema)
			}
			for zxpk := range z.MetaTags {
				var zpez uint32
				zpez, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.MetaTags[zxpk]) >= int(zpez) {
					z.MetaTags[zxpk] = (z.MetaTags[zxpk])[:zpez]
				} else {
					z.MetaTags[zxpk] = make([]string, zpez)
				}
				for zdnj := range z.MetaTags[zxpk] {
					z.MetaTags[zxpk][zdnj], err = dc.ReadString()
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
	for zeff := range z.Tags {
		err = en.WriteArrayHeader(uint32(len(z.Tags[zeff])))
		if err != nil {
			return
		}
		for zrsw := range z.Tags[zeff] {
			err = en.WriteString(z.Tags[zeff][zrsw])
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
	for zxpk := range z.MetaTags {
		err = en.WriteArrayHeader(uint32(len(z.MetaTags[zxpk])))
		if err != nil {
			return
		}
		for zdnj := range z.MetaTags[zxpk] {
			err = en.WriteString(z.MetaTags[zxpk][zdnj])
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
	for zeff := range z.Tags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zeff])))
		for zrsw := range z.Tags[zeff] {
			o = msgp.AppendString(o, z.Tags[zeff][zrsw])
		}
	}
	// string "metatags"
	o = append(o, 0xa8, 0x6d, 0x65, 0x74, 0x61, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zxpk := range z.MetaTags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zxpk])))
		for zdnj := range z.MetaTags[zxpk] {
			o = msgp.AppendString(o, z.MetaTags[zxpk][zdnj])
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SingleMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zqke uint32
	zqke, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zqke > 0 {
		zqke--
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
			var zqyh uint32
			zqyh, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zqyh) {
				z.Tags = (z.Tags)[:zqyh]
			} else {
				z.Tags = make([][]string, zqyh)
			}
			for zeff := range z.Tags {
				var zyzr uint32
				zyzr, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.Tags[zeff]) >= int(zyzr) {
					z.Tags[zeff] = (z.Tags[zeff])[:zyzr]
				} else {
					z.Tags[zeff] = make([]string, zyzr)
				}
				for zrsw := range z.Tags[zeff] {
					z.Tags[zeff][zrsw], bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zywj uint32
			zywj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zywj) {
				z.MetaTags = (z.MetaTags)[:zywj]
			} else {
				z.MetaTags = make([][]string, zywj)
			}
			for zxpk := range z.MetaTags {
				var zjpj uint32
				zjpj, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.MetaTags[zxpk]) >= int(zjpj) {
					z.MetaTags[zxpk] = (z.MetaTags[zxpk])[:zjpj]
				} else {
					z.MetaTags[zxpk] = make([]string, zjpj)
				}
				for zdnj := range z.MetaTags[zxpk] {
					z.MetaTags[zxpk][zdnj], bts, err = msgp.ReadStringBytes(bts)
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
	s = 1 + 5 + msgp.StringPrefixSize + len(z.Type) + 3 + msgp.Uint64Size + 4 + msgp.StringPrefixSize + len(z.Uid) + 5 + msgp.Int64Size + 7 + msgp.StringPrefixSize + len(z.Metric) + 4 + msgp.Float64Size + 4 + msgp.Float64Size + 5 + msgp.Float64Size + 4 + msgp.Float64Size + 6 + msgp.Int64Size + 11 + msgp.Uint32Size + 4 + msgp.Uint32Size + 5 + msgp.ArrayHeaderSize
	for zeff := range z.Tags {
		s += msgp.ArrayHeaderSize
		for zrsw := range z.Tags[zeff] {
			s += msgp.StringPrefixSize + len(z.Tags[zeff][zrsw])
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zxpk := range z.MetaTags {
		s += msgp.ArrayHeaderSize
		for zdnj := range z.MetaTags[zxpk] {
			s += msgp.StringPrefixSize + len(z.MetaTags[zxpk][zdnj])
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UnProcessedMetric) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "tags":
			var zsbz uint32
			zsbz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zsbz) {
				z.Tags = (z.Tags)[:zsbz]
			} else {
				z.Tags = make([][]string, zsbz)
			}
			for zzpf := range z.Tags {
				var zrjx uint32
				zrjx, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.Tags[zzpf]) >= int(zrjx) {
					z.Tags[zzpf] = (z.Tags[zzpf])[:zrjx]
				} else {
					z.Tags[zzpf] = make([]string, zrjx)
				}
				for zrfe := range z.Tags[zzpf] {
					z.Tags[zzpf][zrfe], err = dc.ReadString()
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zawn uint32
			zawn, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zawn) {
				z.MetaTags = (z.MetaTags)[:zawn]
			} else {
				z.MetaTags = make([][]string, zawn)
			}
			for zgmo := range z.MetaTags {
				var zwel uint32
				zwel, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(z.MetaTags[zgmo]) >= int(zwel) {
					z.MetaTags[zgmo] = (z.MetaTags[zgmo])[:zwel]
				} else {
					z.MetaTags[zgmo] = make([]string, zwel)
				}
				for ztaf := range z.MetaTags[zgmo] {
					z.MetaTags[zgmo][ztaf], err = dc.ReadString()
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
	// write "time"
	err = en.Append(0x89, 0xa4, 0x74, 0x69, 0x6d, 0x65)
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
	// write "tags"
	err = en.Append(0xa4, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tags)))
	if err != nil {
		return
	}
	for zzpf := range z.Tags {
		err = en.WriteArrayHeader(uint32(len(z.Tags[zzpf])))
		if err != nil {
			return
		}
		for zrfe := range z.Tags[zzpf] {
			err = en.WriteString(z.Tags[zzpf][zrfe])
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
	for zgmo := range z.MetaTags {
		err = en.WriteArrayHeader(uint32(len(z.MetaTags[zgmo])))
		if err != nil {
			return
		}
		for ztaf := range z.MetaTags[zgmo] {
			err = en.WriteString(z.MetaTags[zgmo][ztaf])
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
	// string "time"
	o = append(o, 0x89, 0xa4, 0x74, 0x69, 0x6d, 0x65)
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
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zzpf := range z.Tags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zzpf])))
		for zrfe := range z.Tags[zzpf] {
			o = msgp.AppendString(o, z.Tags[zzpf][zrfe])
		}
	}
	// string "metatags"
	o = append(o, 0xa8, 0x6d, 0x65, 0x74, 0x61, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zgmo := range z.MetaTags {
		o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zgmo])))
		for ztaf := range z.MetaTags[zgmo] {
			o = msgp.AppendString(o, z.MetaTags[zgmo][ztaf])
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UnProcessedMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zrbe uint32
	zrbe, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zrbe > 0 {
		zrbe--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
		case "tags":
			var zmfd uint32
			zmfd, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zmfd) {
				z.Tags = (z.Tags)[:zmfd]
			} else {
				z.Tags = make([][]string, zmfd)
			}
			for zzpf := range z.Tags {
				var zzdc uint32
				zzdc, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.Tags[zzpf]) >= int(zzdc) {
					z.Tags[zzpf] = (z.Tags[zzpf])[:zzdc]
				} else {
					z.Tags[zzpf] = make([]string, zzdc)
				}
				for zrfe := range z.Tags[zzpf] {
					z.Tags[zzpf][zrfe], bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						return
					}
				}
			}
		case "metatags":
			var zelx uint32
			zelx, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zelx) {
				z.MetaTags = (z.MetaTags)[:zelx]
			} else {
				z.MetaTags = make([][]string, zelx)
			}
			for zgmo := range z.MetaTags {
				var zbal uint32
				zbal, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(z.MetaTags[zgmo]) >= int(zbal) {
					z.MetaTags[zgmo] = (z.MetaTags[zgmo])[:zbal]
				} else {
					z.MetaTags[zgmo] = make([]string, zbal)
				}
				for ztaf := range z.MetaTags[zgmo] {
					z.MetaTags[zgmo][ztaf], bts, err = msgp.ReadStringBytes(bts)
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
	for zzpf := range z.Tags {
		s += msgp.ArrayHeaderSize
		for zrfe := range z.Tags[zzpf] {
			s += msgp.StringPrefixSize + len(z.Tags[zzpf][zrfe])
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zgmo := range z.MetaTags {
		s += msgp.ArrayHeaderSize
		for ztaf := range z.MetaTags[zgmo] {
			s += msgp.StringPrefixSize + len(z.MetaTags[zgmo][ztaf])
		}
	}
	return
}
