package schemas

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

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
				z.Tags = make([]*MetricTag, zwht)
			}
			for zxvk := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zxvk] = nil
				} else {
					if z.Tags[zxvk] == nil {
						z.Tags[zxvk] = new(MetricTag)
					}
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
						case "Tag":
							var zcua uint32
							zcua, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zxvk].Tag) >= int(zcua) {
								z.Tags[zxvk].Tag = (z.Tags[zxvk].Tag)[:zcua]
							} else {
								z.Tags[zxvk].Tag = make([]string, zcua)
							}
							for zbzg := range z.Tags[zxvk].Tag {
								z.Tags[zxvk].Tag[zbzg], err = dc.ReadString()
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
				}
			}
		case "MetaTags":
			var zxhx uint32
			zxhx, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zxhx) {
				z.MetaTags = (z.MetaTags)[:zxhx]
			} else {
				z.MetaTags = make([]*MetricTag, zxhx)
			}
			for zbai := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zbai] = nil
				} else {
					if z.MetaTags[zbai] == nil {
						z.MetaTags[zbai] = new(MetricTag)
					}
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
						case "Tag":
							var zdaf uint32
							zdaf, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[zbai].Tag) >= int(zdaf) {
								z.MetaTags[zbai].Tag = (z.MetaTags[zbai].Tag)[:zdaf]
							} else {
								z.MetaTags[zbai].Tag = make([]string, zdaf)
							}
							for zcmr := range z.MetaTags[zbai].Tag {
								z.MetaTags[zbai].Tag[zcmr], err = dc.ReadString()
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
	for zxvk := range z.Tags {
		if z.Tags[zxvk] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.Tags[zxvk].Tag)))
			if err != nil {
				return
			}
			for zbzg := range z.Tags[zxvk].Tag {
				err = en.WriteString(z.Tags[zxvk].Tag[zbzg])
				if err != nil {
					return
				}
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
	for zbai := range z.MetaTags {
		if z.MetaTags[zbai] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[zbai].Tag)))
			if err != nil {
				return
			}
			for zcmr := range z.MetaTags[zbai].Tag {
				err = en.WriteString(z.MetaTags[zbai].Tag[zcmr])
				if err != nil {
					return
				}
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
	for zxvk := range z.Tags {
		if z.Tags[zxvk] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zxvk].Tag)))
			for zbzg := range z.Tags[zxvk].Tag {
				o = msgp.AppendString(o, z.Tags[zxvk].Tag[zbzg])
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zbai := range z.MetaTags {
		if z.MetaTags[zbai] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zbai].Tag)))
			for zcmr := range z.MetaTags[zbai].Tag {
				o = msgp.AppendString(o, z.MetaTags[zbai].Tag[zcmr])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricName) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
			var zjfb uint32
			zjfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zjfb) {
				z.Tags = (z.Tags)[:zjfb]
			} else {
				z.Tags = make([]*MetricTag, zjfb)
			}
			for zxvk := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zxvk] = nil
				} else {
					if z.Tags[zxvk] == nil {
						z.Tags[zxvk] = new(MetricTag)
					}
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
						case "Tag":
							var zeff uint32
							zeff, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zxvk].Tag) >= int(zeff) {
								z.Tags[zxvk].Tag = (z.Tags[zxvk].Tag)[:zeff]
							} else {
								z.Tags[zxvk].Tag = make([]string, zeff)
							}
							for zbzg := range z.Tags[zxvk].Tag {
								z.Tags[zxvk].Tag[zbzg], bts, err = msgp.ReadStringBytes(bts)
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
				}
			}
		case "MetaTags":
			var zrsw uint32
			zrsw, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zrsw) {
				z.MetaTags = (z.MetaTags)[:zrsw]
			} else {
				z.MetaTags = make([]*MetricTag, zrsw)
			}
			for zbai := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zbai] = nil
				} else {
					if z.MetaTags[zbai] == nil {
						z.MetaTags[zbai] = new(MetricTag)
					}
					var zxpk uint32
					zxpk, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zxpk > 0 {
						zxpk--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zdnj uint32
							zdnj, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[zbai].Tag) >= int(zdnj) {
								z.MetaTags[zbai].Tag = (z.MetaTags[zbai].Tag)[:zdnj]
							} else {
								z.MetaTags[zbai].Tag = make([]string, zdnj)
							}
							for zcmr := range z.MetaTags[zbai].Tag {
								z.MetaTags[zbai].Tag[zcmr], bts, err = msgp.ReadStringBytes(bts)
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
	for zxvk := range z.Tags {
		if z.Tags[zxvk] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zbzg := range z.Tags[zxvk].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zxvk].Tag[zbzg])
			}
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zbai := range z.MetaTags {
		if z.MetaTags[zbai] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zcmr := range z.MetaTags[zbai].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[zbai].Tag[zcmr])
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricTag) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zsnv uint32
	zsnv, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zsnv > 0 {
		zsnv--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Tag":
			var zkgt uint32
			zkgt, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tag) >= int(zkgt) {
				z.Tag = (z.Tag)[:zkgt]
			} else {
				z.Tag = make([]string, zkgt)
			}
			for zobc := range z.Tag {
				z.Tag[zobc], err = dc.ReadString()
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
func (z *MetricTag) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "Tag"
	err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Tag)))
	if err != nil {
		return
	}
	for zobc := range z.Tag {
		err = en.WriteString(z.Tag[zobc])
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricTag) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Tag"
	o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tag)))
	for zobc := range z.Tag {
		o = msgp.AppendString(o, z.Tag[zobc])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricTag) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zema uint32
	zema, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zema > 0 {
		zema--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Tag":
			var zpez uint32
			zpez, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tag) >= int(zpez) {
				z.Tag = (z.Tag)[:zpez]
			} else {
				z.Tag = make([]string, zpez)
			}
			for zobc := range z.Tag {
				z.Tag[zobc], bts, err = msgp.ReadStringBytes(bts)
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
func (z *MetricTag) Msgsize() (s int) {
	s = 1 + 4 + msgp.ArrayHeaderSize
	for zobc := range z.Tag {
		s += msgp.StringPrefixSize + len(z.Tag[zobc])
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricValue) DecodeMsg(dc *msgp.Reader) (err error) {
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
	var zqyh uint32
	zqyh, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zqyh > 0 {
		zqyh--
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
	var zrfe uint32
	zrfe, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zrfe > 0 {
		zrfe--
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
			var zgmo uint32
			zgmo, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zgmo) {
				z.Tags = (z.Tags)[:zgmo]
			} else {
				z.Tags = make([]*MetricTag, zgmo)
			}
			for zyzr := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zyzr] = nil
				} else {
					if z.Tags[zyzr] == nil {
						z.Tags[zyzr] = new(MetricTag)
					}
					var ztaf uint32
					ztaf, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for ztaf > 0 {
						ztaf--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zeth uint32
							zeth, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zyzr].Tag) >= int(zeth) {
								z.Tags[zyzr].Tag = (z.Tags[zyzr].Tag)[:zeth]
							} else {
								z.Tags[zyzr].Tag = make([]string, zeth)
							}
							for zywj := range z.Tags[zyzr].Tag {
								z.Tags[zyzr].Tag[zywj], err = dc.ReadString()
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
				}
			}
		case "MetaTags":
			var zsbz uint32
			zsbz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zsbz) {
				z.MetaTags = (z.MetaTags)[:zsbz]
			} else {
				z.MetaTags = make([]*MetricTag, zsbz)
			}
			for zjpj := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zjpj] = nil
				} else {
					if z.MetaTags[zjpj] == nil {
						z.MetaTags[zjpj] = new(MetricTag)
					}
					var zrjx uint32
					zrjx, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zrjx > 0 {
						zrjx--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zawn uint32
							zawn, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[zjpj].Tag) >= int(zawn) {
								z.MetaTags[zjpj].Tag = (z.MetaTags[zjpj].Tag)[:zawn]
							} else {
								z.MetaTags[zjpj].Tag = make([]string, zawn)
							}
							for zzpf := range z.MetaTags[zjpj].Tag {
								z.MetaTags[zjpj].Tag[zzpf], err = dc.ReadString()
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
	for zyzr := range z.Tags {
		if z.Tags[zyzr] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.Tags[zyzr].Tag)))
			if err != nil {
				return
			}
			for zywj := range z.Tags[zyzr].Tag {
				err = en.WriteString(z.Tags[zyzr].Tag[zywj])
				if err != nil {
					return
				}
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
	for zjpj := range z.MetaTags {
		if z.MetaTags[zjpj] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[zjpj].Tag)))
			if err != nil {
				return
			}
			for zzpf := range z.MetaTags[zjpj].Tag {
				err = en.WriteString(z.MetaTags[zjpj].Tag[zzpf])
				if err != nil {
					return
				}
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
	for zyzr := range z.Tags {
		if z.Tags[zyzr] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zyzr].Tag)))
			for zywj := range z.Tags[zyzr].Tag {
				o = msgp.AppendString(o, z.Tags[zyzr].Tag[zywj])
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zjpj := range z.MetaTags {
		if z.MetaTags[zjpj] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zjpj].Tag)))
			for zzpf := range z.MetaTags[zjpj].Tag {
				o = msgp.AppendString(o, z.MetaTags[zjpj].Tag[zzpf])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zwel uint32
	zwel, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zwel > 0 {
		zwel--
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
			var zrbe uint32
			zrbe, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zrbe) {
				z.Tags = (z.Tags)[:zrbe]
			} else {
				z.Tags = make([]*MetricTag, zrbe)
			}
			for zyzr := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zyzr] = nil
				} else {
					if z.Tags[zyzr] == nil {
						z.Tags[zyzr] = new(MetricTag)
					}
					var zmfd uint32
					zmfd, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zmfd > 0 {
						zmfd--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zzdc uint32
							zzdc, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zyzr].Tag) >= int(zzdc) {
								z.Tags[zyzr].Tag = (z.Tags[zyzr].Tag)[:zzdc]
							} else {
								z.Tags[zyzr].Tag = make([]string, zzdc)
							}
							for zywj := range z.Tags[zyzr].Tag {
								z.Tags[zyzr].Tag[zywj], bts, err = msgp.ReadStringBytes(bts)
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
				}
			}
		case "MetaTags":
			var zelx uint32
			zelx, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zelx) {
				z.MetaTags = (z.MetaTags)[:zelx]
			} else {
				z.MetaTags = make([]*MetricTag, zelx)
			}
			for zjpj := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zjpj] = nil
				} else {
					if z.MetaTags[zjpj] == nil {
						z.MetaTags[zjpj] = new(MetricTag)
					}
					var zbal uint32
					zbal, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zbal > 0 {
						zbal--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zjqz uint32
							zjqz, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[zjpj].Tag) >= int(zjqz) {
								z.MetaTags[zjpj].Tag = (z.MetaTags[zjpj].Tag)[:zjqz]
							} else {
								z.MetaTags[zjpj].Tag = make([]string, zjqz)
							}
							for zzpf := range z.MetaTags[zjpj].Tag {
								z.MetaTags[zjpj].Tag[zzpf], bts, err = msgp.ReadStringBytes(bts)
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
	for zyzr := range z.Tags {
		if z.Tags[zyzr] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zywj := range z.Tags[zyzr].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zyzr].Tag[zywj])
			}
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zjpj := range z.MetaTags {
		if z.MetaTags[zjpj] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zzpf := range z.MetaTags[zjpj].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[zjpj].Tag[zzpf])
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SeriesMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var ztyy uint32
	ztyy, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for ztyy > 0 {
		ztyy--
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
			var zinl uint32
			zinl, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zinl) {
				z.Tags = (z.Tags)[:zinl]
			} else {
				z.Tags = make([]*MetricTag, zinl)
			}
			for zkct := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zkct] = nil
				} else {
					if z.Tags[zkct] == nil {
						z.Tags[zkct] = new(MetricTag)
					}
					var zare uint32
					zare, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zare > 0 {
						zare--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zljy uint32
							zljy, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zkct].Tag) >= int(zljy) {
								z.Tags[zkct].Tag = (z.Tags[zkct].Tag)[:zljy]
							} else {
								z.Tags[zkct].Tag = make([]string, zljy)
							}
							for ztmt := range z.Tags[zkct].Tag {
								z.Tags[zkct].Tag[ztmt], err = dc.ReadString()
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
				}
			}
		case "MetaTags":
			var zixj uint32
			zixj, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zixj) {
				z.MetaTags = (z.MetaTags)[:zixj]
			} else {
				z.MetaTags = make([]*MetricTag, zixj)
			}
			for ztco := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[ztco] = nil
				} else {
					if z.MetaTags[ztco] == nil {
						z.MetaTags[ztco] = new(MetricTag)
					}
					var zrsc uint32
					zrsc, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zrsc > 0 {
						zrsc--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zctn uint32
							zctn, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[ztco].Tag) >= int(zctn) {
								z.MetaTags[ztco].Tag = (z.MetaTags[ztco].Tag)[:zctn]
							} else {
								z.MetaTags[ztco].Tag = make([]string, zctn)
							}
							for zana := range z.MetaTags[ztco].Tag {
								z.MetaTags[ztco].Tag[zana], err = dc.ReadString()
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
	for zkct := range z.Tags {
		if z.Tags[zkct] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.Tags[zkct].Tag)))
			if err != nil {
				return
			}
			for ztmt := range z.Tags[zkct].Tag {
				err = en.WriteString(z.Tags[zkct].Tag[ztmt])
				if err != nil {
					return
				}
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
	for ztco := range z.MetaTags {
		if z.MetaTags[ztco] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[ztco].Tag)))
			if err != nil {
				return
			}
			for zana := range z.MetaTags[ztco].Tag {
				err = en.WriteString(z.MetaTags[ztco].Tag[zana])
				if err != nil {
					return
				}
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
	for zkct := range z.Tags {
		if z.Tags[zkct] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zkct].Tag)))
			for ztmt := range z.Tags[zkct].Tag {
				o = msgp.AppendString(o, z.Tags[zkct].Tag[ztmt])
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for ztco := range z.MetaTags {
		if z.MetaTags[ztco] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[ztco].Tag)))
			for zana := range z.MetaTags[ztco].Tag {
				o = msgp.AppendString(o, z.MetaTags[ztco].Tag[zana])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SeriesMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zswy uint32
	zswy, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zswy > 0 {
		zswy--
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
			var znsg uint32
			znsg, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(znsg) {
				z.Tags = (z.Tags)[:znsg]
			} else {
				z.Tags = make([]*MetricTag, znsg)
			}
			for zkct := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zkct] = nil
				} else {
					if z.Tags[zkct] == nil {
						z.Tags[zkct] = new(MetricTag)
					}
					var zrus uint32
					zrus, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zrus > 0 {
						zrus--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zsvm uint32
							zsvm, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zkct].Tag) >= int(zsvm) {
								z.Tags[zkct].Tag = (z.Tags[zkct].Tag)[:zsvm]
							} else {
								z.Tags[zkct].Tag = make([]string, zsvm)
							}
							for ztmt := range z.Tags[zkct].Tag {
								z.Tags[zkct].Tag[ztmt], bts, err = msgp.ReadStringBytes(bts)
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
				}
			}
		case "MetaTags":
			var zaoz uint32
			zaoz, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zaoz) {
				z.MetaTags = (z.MetaTags)[:zaoz]
			} else {
				z.MetaTags = make([]*MetricTag, zaoz)
			}
			for ztco := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[ztco] = nil
				} else {
					if z.MetaTags[ztco] == nil {
						z.MetaTags[ztco] = new(MetricTag)
					}
					var zfzb uint32
					zfzb, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zfzb > 0 {
						zfzb--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zsbo uint32
							zsbo, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[ztco].Tag) >= int(zsbo) {
								z.MetaTags[ztco].Tag = (z.MetaTags[ztco].Tag)[:zsbo]
							} else {
								z.MetaTags[ztco].Tag = make([]string, zsbo)
							}
							for zana := range z.MetaTags[ztco].Tag {
								z.MetaTags[ztco].Tag[zana], bts, err = msgp.ReadStringBytes(bts)
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
	for zkct := range z.Tags {
		if z.Tags[zkct] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for ztmt := range z.Tags[zkct].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zkct].Tag[ztmt])
			}
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for ztco := range z.MetaTags {
		if z.MetaTags[ztco] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zana := range z.MetaTags[ztco].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[ztco].Tag[zana])
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SingleMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zmvo uint32
	zmvo, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zmvo > 0 {
		zmvo--
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
			var zigk uint32
			zigk, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zigk) {
				z.Tags = (z.Tags)[:zigk]
			} else {
				z.Tags = make([]*MetricTag, zigk)
			}
			for zjif := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zjif] = nil
				} else {
					if z.Tags[zjif] == nil {
						z.Tags[zjif] = new(MetricTag)
					}
					var zopb uint32
					zopb, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zopb > 0 {
						zopb--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zuop uint32
							zuop, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zjif].Tag) >= int(zuop) {
								z.Tags[zjif].Tag = (z.Tags[zjif].Tag)[:zuop]
							} else {
								z.Tags[zjif].Tag = make([]string, zuop)
							}
							for zqgz := range z.Tags[zjif].Tag {
								z.Tags[zjif].Tag[zqgz], err = dc.ReadString()
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
				}
			}
		case "MetaTags":
			var zedl uint32
			zedl, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zedl) {
				z.MetaTags = (z.MetaTags)[:zedl]
			} else {
				z.MetaTags = make([]*MetricTag, zedl)
			}
			for zsnw := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zsnw] = nil
				} else {
					if z.MetaTags[zsnw] == nil {
						z.MetaTags[zsnw] = new(MetricTag)
					}
					var zupd uint32
					zupd, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zupd > 0 {
						zupd--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zome uint32
							zome, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[zsnw].Tag) >= int(zome) {
								z.MetaTags[zsnw].Tag = (z.MetaTags[zsnw].Tag)[:zome]
							} else {
								z.MetaTags[zsnw].Tag = make([]string, zome)
							}
							for ztls := range z.MetaTags[zsnw].Tag {
								z.MetaTags[zsnw].Tag[ztls], err = dc.ReadString()
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
	for zjif := range z.Tags {
		if z.Tags[zjif] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.Tags[zjif].Tag)))
			if err != nil {
				return
			}
			for zqgz := range z.Tags[zjif].Tag {
				err = en.WriteString(z.Tags[zjif].Tag[zqgz])
				if err != nil {
					return
				}
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
	for zsnw := range z.MetaTags {
		if z.MetaTags[zsnw] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[zsnw].Tag)))
			if err != nil {
				return
			}
			for ztls := range z.MetaTags[zsnw].Tag {
				err = en.WriteString(z.MetaTags[zsnw].Tag[ztls])
				if err != nil {
					return
				}
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
	for zjif := range z.Tags {
		if z.Tags[zjif] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zjif].Tag)))
			for zqgz := range z.Tags[zjif].Tag {
				o = msgp.AppendString(o, z.Tags[zjif].Tag[zqgz])
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zsnw := range z.MetaTags {
		if z.MetaTags[zsnw] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zsnw].Tag)))
			for ztls := range z.MetaTags[zsnw].Tag {
				o = msgp.AppendString(o, z.MetaTags[zsnw].Tag[ztls])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SingleMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zrvj uint32
	zrvj, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zrvj > 0 {
		zrvj--
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
			var zarz uint32
			zarz, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zarz) {
				z.Tags = (z.Tags)[:zarz]
			} else {
				z.Tags = make([]*MetricTag, zarz)
			}
			for zjif := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zjif] = nil
				} else {
					if z.Tags[zjif] == nil {
						z.Tags[zjif] = new(MetricTag)
					}
					var zknt uint32
					zknt, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zknt > 0 {
						zknt--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zxye uint32
							zxye, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zjif].Tag) >= int(zxye) {
								z.Tags[zjif].Tag = (z.Tags[zjif].Tag)[:zxye]
							} else {
								z.Tags[zjif].Tag = make([]string, zxye)
							}
							for zqgz := range z.Tags[zjif].Tag {
								z.Tags[zjif].Tag[zqgz], bts, err = msgp.ReadStringBytes(bts)
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
				}
			}
		case "MetaTags":
			var zucw uint32
			zucw, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zucw) {
				z.MetaTags = (z.MetaTags)[:zucw]
			} else {
				z.MetaTags = make([]*MetricTag, zucw)
			}
			for zsnw := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zsnw] = nil
				} else {
					if z.MetaTags[zsnw] == nil {
						z.MetaTags[zsnw] = new(MetricTag)
					}
					var zlsx uint32
					zlsx, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zlsx > 0 {
						zlsx--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zbgy uint32
							zbgy, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[zsnw].Tag) >= int(zbgy) {
								z.MetaTags[zsnw].Tag = (z.MetaTags[zsnw].Tag)[:zbgy]
							} else {
								z.MetaTags[zsnw].Tag = make([]string, zbgy)
							}
							for ztls := range z.MetaTags[zsnw].Tag {
								z.MetaTags[zsnw].Tag[ztls], bts, err = msgp.ReadStringBytes(bts)
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
	for zjif := range z.Tags {
		if z.Tags[zjif] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zqgz := range z.Tags[zjif].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zjif].Tag[zqgz])
			}
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zsnw := range z.MetaTags {
		if z.MetaTags[zsnw] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for ztls := range z.MetaTags[zsnw].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[zsnw].Tag[ztls])
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UnProcessedMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zzak uint32
	zzak, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zzak > 0 {
		zzak--
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
			var zbtz uint32
			zbtz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zbtz) {
				z.Tags = (z.Tags)[:zbtz]
			} else {
				z.Tags = make([]*MetricTag, zbtz)
			}
			for zrao := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zrao] = nil
				} else {
					if z.Tags[zrao] == nil {
						z.Tags[zrao] = new(MetricTag)
					}
					var zsym uint32
					zsym, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zsym > 0 {
						zsym--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zgeu uint32
							zgeu, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zrao].Tag) >= int(zgeu) {
								z.Tags[zrao].Tag = (z.Tags[zrao].Tag)[:zgeu]
							} else {
								z.Tags[zrao].Tag = make([]string, zgeu)
							}
							for zmbt := range z.Tags[zrao].Tag {
								z.Tags[zrao].Tag[zmbt], err = dc.ReadString()
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
				}
			}
		case "MetaTags":
			var zdtr uint32
			zdtr, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zdtr) {
				z.MetaTags = (z.MetaTags)[:zdtr]
			} else {
				z.MetaTags = make([]*MetricTag, zdtr)
			}
			for zvls := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zvls] = nil
				} else {
					if z.MetaTags[zvls] == nil {
						z.MetaTags[zvls] = new(MetricTag)
					}
					var zzqm uint32
					zzqm, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zzqm > 0 {
						zzqm--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zdqi uint32
							zdqi, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[zvls].Tag) >= int(zdqi) {
								z.MetaTags[zvls].Tag = (z.MetaTags[zvls].Tag)[:zdqi]
							} else {
								z.MetaTags[zvls].Tag = make([]string, zdqi)
							}
							for zjfj := range z.MetaTags[zvls].Tag {
								z.MetaTags[zvls].Tag[zjfj], err = dc.ReadString()
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
	for zrao := range z.Tags {
		if z.Tags[zrao] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.Tags[zrao].Tag)))
			if err != nil {
				return
			}
			for zmbt := range z.Tags[zrao].Tag {
				err = en.WriteString(z.Tags[zrao].Tag[zmbt])
				if err != nil {
					return
				}
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
	for zvls := range z.MetaTags {
		if z.MetaTags[zvls] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[zvls].Tag)))
			if err != nil {
				return
			}
			for zjfj := range z.MetaTags[zvls].Tag {
				err = en.WriteString(z.MetaTags[zvls].Tag[zjfj])
				if err != nil {
					return
				}
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
	for zrao := range z.Tags {
		if z.Tags[zrao] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zrao].Tag)))
			for zmbt := range z.Tags[zrao].Tag {
				o = msgp.AppendString(o, z.Tags[zrao].Tag[zmbt])
			}
		}
	}
	// string "MetaTags"
	o = append(o, 0xa8, 0x4d, 0x65, 0x74, 0x61, 0x54, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zvls := range z.MetaTags {
		if z.MetaTags[zvls] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zvls].Tag)))
			for zjfj := range z.MetaTags[zvls].Tag {
				o = msgp.AppendString(o, z.MetaTags[zvls].Tag[zjfj])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UnProcessedMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zyco uint32
	zyco, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zyco > 0 {
		zyco--
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
			var zhgh uint32
			zhgh, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zhgh) {
				z.Tags = (z.Tags)[:zhgh]
			} else {
				z.Tags = make([]*MetricTag, zhgh)
			}
			for zrao := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zrao] = nil
				} else {
					if z.Tags[zrao] == nil {
						z.Tags[zrao] = new(MetricTag)
					}
					var zovg uint32
					zovg, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zovg > 0 {
						zovg--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zsey uint32
							zsey, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zrao].Tag) >= int(zsey) {
								z.Tags[zrao].Tag = (z.Tags[zrao].Tag)[:zsey]
							} else {
								z.Tags[zrao].Tag = make([]string, zsey)
							}
							for zmbt := range z.Tags[zrao].Tag {
								z.Tags[zrao].Tag[zmbt], bts, err = msgp.ReadStringBytes(bts)
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
				}
			}
		case "MetaTags":
			var zcjp uint32
			zcjp, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zcjp) {
				z.MetaTags = (z.MetaTags)[:zcjp]
			} else {
				z.MetaTags = make([]*MetricTag, zcjp)
			}
			for zvls := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zvls] = nil
				} else {
					if z.MetaTags[zvls] == nil {
						z.MetaTags[zvls] = new(MetricTag)
					}
					var zjhy uint32
					zjhy, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zjhy > 0 {
						zjhy--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var znuf uint32
							znuf, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[zvls].Tag) >= int(znuf) {
								z.MetaTags[zvls].Tag = (z.MetaTags[zvls].Tag)[:znuf]
							} else {
								z.MetaTags[zvls].Tag = make([]string, znuf)
							}
							for zjfj := range z.MetaTags[zvls].Tag {
								z.MetaTags[zvls].Tag[zjfj], bts, err = msgp.ReadStringBytes(bts)
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
	for zrao := range z.Tags {
		if z.Tags[zrao] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zmbt := range z.Tags[zrao].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zrao].Tag[zmbt])
			}
		}
	}
	s += 9 + msgp.ArrayHeaderSize
	for zvls := range z.MetaTags {
		if z.MetaTags[zvls] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zjfj := range z.MetaTags[zvls].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[zvls].Tag[zjfj])
			}
		}
	}
	return
}
