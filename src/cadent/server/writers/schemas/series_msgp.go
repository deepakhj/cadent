package schemas

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
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
		case "metric":
			z.Metric, err = dc.ReadString()
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
		case "tags":
			var zcua uint32
			zcua, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zcua) {
				z.Tags = (z.Tags)[:zcua]
			} else {
				z.Tags = make([]*MetricTag, zcua)
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
						z.Tags[zbai] = new(MetricTag)
					}
					var zxhx uint32
					zxhx, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zxhx > 0 {
						zxhx--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zlqf uint32
							zlqf, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zbai].Tag) >= int(zlqf) {
								z.Tags[zbai].Tag = (z.Tags[zbai].Tag)[:zlqf]
							} else {
								z.Tags[zbai].Tag = make([]string, zlqf)
							}
							for zcmr := range z.Tags[zbai].Tag {
								z.Tags[zbai].Tag[zcmr], err = dc.ReadString()
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
		case "meta_tags":
			var zdaf uint32
			zdaf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zdaf) {
				z.MetaTags = (z.MetaTags)[:zdaf]
			} else {
				z.MetaTags = make([]*MetricTag, zdaf)
			}
			for zajw := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zajw] = nil
				} else {
					if z.MetaTags[zajw] == nil {
						z.MetaTags[zajw] = new(MetricTag)
					}
					var zpks uint32
					zpks, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zpks > 0 {
						zpks--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zjfb uint32
							zjfb, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[zajw].Tag) >= int(zjfb) {
								z.MetaTags[zajw].Tag = (z.MetaTags[zajw].Tag)[:zjfb]
							} else {
								z.MetaTags[zajw].Tag = make([]string, zjfb)
							}
							for zwht := range z.MetaTags[zajw].Tag {
								z.MetaTags[zajw].Tag[zwht], err = dc.ReadString()
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
	// write "metric"
	err = en.Append(0x85, 0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Metric)
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
		if z.Tags[zbai] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.Tags[zbai].Tag)))
			if err != nil {
				return
			}
			for zcmr := range z.Tags[zbai].Tag {
				err = en.WriteString(z.Tags[zbai].Tag[zcmr])
				if err != nil {
					return
				}
			}
		}
	}
	// write "meta_tags"
	err = en.Append(0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zajw := range z.MetaTags {
		if z.MetaTags[zajw] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[zajw].Tag)))
			if err != nil {
				return
			}
			for zwht := range z.MetaTags[zajw].Tag {
				err = en.WriteString(z.MetaTags[zajw].Tag[zwht])
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
	// string "metric"
	o = append(o, 0x85, 0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "id"
	o = append(o, 0xa2, 0x69, 0x64)
	o = msgp.AppendUint64(o, z.Id)
	// string "uid"
	o = append(o, 0xa3, 0x75, 0x69, 0x64)
	o = msgp.AppendString(o, z.Uid)
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zbai := range z.Tags {
		if z.Tags[zbai] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zbai].Tag)))
			for zcmr := range z.Tags[zbai].Tag {
				o = msgp.AppendString(o, z.Tags[zbai].Tag[zcmr])
			}
		}
	}
	// string "meta_tags"
	o = append(o, 0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zajw := range z.MetaTags {
		if z.MetaTags[zajw] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zajw].Tag)))
			for zwht := range z.MetaTags[zajw].Tag {
				o = msgp.AppendString(o, z.MetaTags[zajw].Tag[zwht])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricName) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "metric":
			z.Metric, bts, err = msgp.ReadStringBytes(bts)
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
		case "tags":
			var zeff uint32
			zeff, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zeff) {
				z.Tags = (z.Tags)[:zeff]
			} else {
				z.Tags = make([]*MetricTag, zeff)
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
						z.Tags[zbai] = new(MetricTag)
					}
					var zrsw uint32
					zrsw, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zrsw > 0 {
						zrsw--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zxpk uint32
							zxpk, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zbai].Tag) >= int(zxpk) {
								z.Tags[zbai].Tag = (z.Tags[zbai].Tag)[:zxpk]
							} else {
								z.Tags[zbai].Tag = make([]string, zxpk)
							}
							for zcmr := range z.Tags[zbai].Tag {
								z.Tags[zbai].Tag[zcmr], bts, err = msgp.ReadStringBytes(bts)
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
		case "meta_tags":
			var zdnj uint32
			zdnj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zdnj) {
				z.MetaTags = (z.MetaTags)[:zdnj]
			} else {
				z.MetaTags = make([]*MetricTag, zdnj)
			}
			for zajw := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zajw] = nil
				} else {
					if z.MetaTags[zajw] == nil {
						z.MetaTags[zajw] = new(MetricTag)
					}
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
						case "Tag":
							var zsnv uint32
							zsnv, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[zajw].Tag) >= int(zsnv) {
								z.MetaTags[zajw].Tag = (z.MetaTags[zajw].Tag)[:zsnv]
							} else {
								z.MetaTags[zajw].Tag = make([]string, zsnv)
							}
							for zwht := range z.MetaTags[zajw].Tag {
								z.MetaTags[zajw].Tag[zwht], bts, err = msgp.ReadStringBytes(bts)
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
	for zbai := range z.Tags {
		if z.Tags[zbai] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zcmr := range z.Tags[zbai].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zbai].Tag[zcmr])
			}
		}
	}
	s += 10 + msgp.ArrayHeaderSize
	for zajw := range z.MetaTags {
		if z.MetaTags[zajw] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zwht := range z.MetaTags[zajw].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[zajw].Tag[zwht])
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricTag) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zema uint32
	zema, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zema > 0 {
		zema--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "Tag":
			var zpez uint32
			zpez, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tag) >= int(zpez) {
				z.Tag = (z.Tag)[:zpez]
			} else {
				z.Tag = make([]string, zpez)
			}
			for zkgt := range z.Tag {
				z.Tag[zkgt], err = dc.ReadString()
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
	for zkgt := range z.Tag {
		err = en.WriteString(z.Tag[zkgt])
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
	for zkgt := range z.Tag {
		o = msgp.AppendString(o, z.Tag[zkgt])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricTag) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Tag":
			var zqyh uint32
			zqyh, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tag) >= int(zqyh) {
				z.Tag = (z.Tag)[:zqyh]
			} else {
				z.Tag = make([]string, zqyh)
			}
			for zkgt := range z.Tag {
				z.Tag[zkgt], bts, err = msgp.ReadStringBytes(bts)
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
	for zkgt := range z.Tag {
		s += msgp.StringPrefixSize + len(z.Tag[zkgt])
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricType) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
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
		case "type":
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
	// write "type"
	err = en.Append(0x81, 0xa4, 0x74, 0x79, 0x70, 0x65)
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
	// string "type"
	o = append(o, 0x81, 0xa4, 0x74, 0x79, 0x70, 0x65)
	o = msgp.AppendString(o, z.Type)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricType) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "type":
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
	var zjpj uint32
	zjpj, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zjpj > 0 {
		zjpj--
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
	// write "time"
	err = en.Append(0x86, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
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
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *MetricValue) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "time"
	o = append(o, 0x86, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
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
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricValue) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
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
		case "time":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
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
	var zsbz uint32
	zsbz, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zsbz > 0 {
		zsbz--
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
		case "value":
			z.Value, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "tags":
			var zrjx uint32
			zrjx, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zrjx) {
				z.Tags = (z.Tags)[:zrjx]
			} else {
				z.Tags = make([]*MetricTag, zrjx)
			}
			for zrfe := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zrfe] = nil
				} else {
					if z.Tags[zrfe] == nil {
						z.Tags[zrfe] = new(MetricTag)
					}
					var zawn uint32
					zawn, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zawn > 0 {
						zawn--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zwel uint32
							zwel, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zrfe].Tag) >= int(zwel) {
								z.Tags[zrfe].Tag = (z.Tags[zrfe].Tag)[:zwel]
							} else {
								z.Tags[zrfe].Tag = make([]string, zwel)
							}
							for zgmo := range z.Tags[zrfe].Tag {
								z.Tags[zrfe].Tag[zgmo], err = dc.ReadString()
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
		case "meta_tags":
			var zrbe uint32
			zrbe, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zrbe) {
				z.MetaTags = (z.MetaTags)[:zrbe]
			} else {
				z.MetaTags = make([]*MetricTag, zrbe)
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
						z.MetaTags[ztaf] = new(MetricTag)
					}
					var zmfd uint32
					zmfd, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zmfd > 0 {
						zmfd--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zzdc uint32
							zzdc, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[ztaf].Tag) >= int(zzdc) {
								z.MetaTags[ztaf].Tag = (z.MetaTags[ztaf].Tag)[:zzdc]
							} else {
								z.MetaTags[ztaf].Tag = make([]string, zzdc)
							}
							for zeth := range z.MetaTags[ztaf].Tag {
								z.MetaTags[ztaf].Tag[zeth], err = dc.ReadString()
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
	// write "time"
	err = en.Append(0x85, 0xa4, 0x74, 0x69, 0x6d, 0x65)
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
	// write "value"
	err = en.Append(0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Value)
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
	for zrfe := range z.Tags {
		if z.Tags[zrfe] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.Tags[zrfe].Tag)))
			if err != nil {
				return
			}
			for zgmo := range z.Tags[zrfe].Tag {
				err = en.WriteString(z.Tags[zrfe].Tag[zgmo])
				if err != nil {
					return
				}
			}
		}
	}
	// write "meta_tags"
	err = en.Append(0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
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
			// map header, size 1
			// write "Tag"
			err = en.Append(0x81, 0xa3, 0x54, 0x61, 0x67)
			if err != nil {
				return err
			}
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[ztaf].Tag)))
			if err != nil {
				return
			}
			for zeth := range z.MetaTags[ztaf].Tag {
				err = en.WriteString(z.MetaTags[ztaf].Tag[zeth])
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
	// string "time"
	o = append(o, 0x85, 0xa4, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendInt64(o, z.Time)
	// string "metric"
	o = append(o, 0xa6, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63)
	o = msgp.AppendString(o, z.Metric)
	// string "value"
	o = append(o, 0xa5, 0x76, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendFloat64(o, z.Value)
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zrfe := range z.Tags {
		if z.Tags[zrfe] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zrfe].Tag)))
			for zgmo := range z.Tags[zrfe].Tag {
				o = msgp.AppendString(o, z.Tags[zrfe].Tag[zgmo])
			}
		}
	}
	// string "meta_tags"
	o = append(o, 0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for ztaf := range z.MetaTags {
		if z.MetaTags[ztaf] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[ztaf].Tag)))
			for zeth := range z.MetaTags[ztaf].Tag {
				o = msgp.AppendString(o, z.MetaTags[ztaf].Tag[zeth])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zelx uint32
	zelx, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zelx > 0 {
		zelx--
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
		case "value":
			z.Value, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "tags":
			var zbal uint32
			zbal, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zbal) {
				z.Tags = (z.Tags)[:zbal]
			} else {
				z.Tags = make([]*MetricTag, zbal)
			}
			for zrfe := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zrfe] = nil
				} else {
					if z.Tags[zrfe] == nil {
						z.Tags[zrfe] = new(MetricTag)
					}
					var zjqz uint32
					zjqz, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zjqz > 0 {
						zjqz--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zkct uint32
							zkct, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zrfe].Tag) >= int(zkct) {
								z.Tags[zrfe].Tag = (z.Tags[zrfe].Tag)[:zkct]
							} else {
								z.Tags[zrfe].Tag = make([]string, zkct)
							}
							for zgmo := range z.Tags[zrfe].Tag {
								z.Tags[zrfe].Tag[zgmo], bts, err = msgp.ReadStringBytes(bts)
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
		case "meta_tags":
			var ztmt uint32
			ztmt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(ztmt) {
				z.MetaTags = (z.MetaTags)[:ztmt]
			} else {
				z.MetaTags = make([]*MetricTag, ztmt)
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
						z.MetaTags[ztaf] = new(MetricTag)
					}
					var ztco uint32
					ztco, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for ztco > 0 {
						ztco--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zana uint32
							zana, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[ztaf].Tag) >= int(zana) {
								z.MetaTags[ztaf].Tag = (z.MetaTags[ztaf].Tag)[:zana]
							} else {
								z.MetaTags[ztaf].Tag = make([]string, zana)
							}
							for zeth := range z.MetaTags[ztaf].Tag {
								z.MetaTags[ztaf].Tag[zeth], bts, err = msgp.ReadStringBytes(bts)
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
	for zrfe := range z.Tags {
		if z.Tags[zrfe] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zgmo := range z.Tags[zrfe].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zrfe].Tag[zgmo])
			}
		}
	}
	s += 10 + msgp.ArrayHeaderSize
	for ztaf := range z.MetaTags {
		if z.MetaTags[ztaf] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zeth := range z.MetaTags[ztaf].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[ztaf].Tag[zeth])
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SeriesMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zixj uint32
	zixj, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zixj > 0 {
		zixj--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
			z.Ttl, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "tags":
			var zrsc uint32
			zrsc, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zrsc) {
				z.Tags = (z.Tags)[:zrsc]
			} else {
				z.Tags = make([]*MetricTag, zrsc)
			}
			for ztyy := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[ztyy] = nil
				} else {
					if z.Tags[ztyy] == nil {
						z.Tags[ztyy] = new(MetricTag)
					}
					var zctn uint32
					zctn, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zctn > 0 {
						zctn--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zswy uint32
							zswy, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[ztyy].Tag) >= int(zswy) {
								z.Tags[ztyy].Tag = (z.Tags[ztyy].Tag)[:zswy]
							} else {
								z.Tags[ztyy].Tag = make([]string, zswy)
							}
							for zinl := range z.Tags[ztyy].Tag {
								z.Tags[ztyy].Tag[zinl], err = dc.ReadString()
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
		case "meta_tags":
			var znsg uint32
			znsg, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(znsg) {
				z.MetaTags = (z.MetaTags)[:znsg]
			} else {
				z.MetaTags = make([]*MetricTag, znsg)
			}
			for zare := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zare] = nil
				} else {
					if z.MetaTags[zare] == nil {
						z.MetaTags[zare] = new(MetricTag)
					}
					var zrus uint32
					zrus, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zrus > 0 {
						zrus--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zsvm uint32
							zsvm, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[zare].Tag) >= int(zsvm) {
								z.MetaTags[zare].Tag = (z.MetaTags[zare].Tag)[:zsvm]
							} else {
								z.MetaTags[zare].Tag = make([]string, zsvm)
							}
							for zljy := range z.MetaTags[zare].Tag {
								z.MetaTags[zare].Tag[zljy], err = dc.ReadString()
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
	// write "id"
	err = en.Append(0x8a, 0xa2, 0x69, 0x64)
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
	err = en.WriteUint32(z.Ttl)
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
	for ztyy := range z.Tags {
		if z.Tags[ztyy] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.Tags[ztyy].Tag)))
			if err != nil {
				return
			}
			for zinl := range z.Tags[ztyy].Tag {
				err = en.WriteString(z.Tags[ztyy].Tag[zinl])
				if err != nil {
					return
				}
			}
		}
	}
	// write "meta_tags"
	err = en.Append(0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zare := range z.MetaTags {
		if z.MetaTags[zare] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[zare].Tag)))
			if err != nil {
				return
			}
			for zljy := range z.MetaTags[zare].Tag {
				err = en.WriteString(z.MetaTags[zare].Tag[zljy])
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
	// string "id"
	o = append(o, 0x8a, 0xa2, 0x69, 0x64)
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
	o = msgp.AppendUint32(o, z.Ttl)
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for ztyy := range z.Tags {
		if z.Tags[ztyy] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[ztyy].Tag)))
			for zinl := range z.Tags[ztyy].Tag {
				o = msgp.AppendString(o, z.Tags[ztyy].Tag[zinl])
			}
		}
	}
	// string "meta_tags"
	o = append(o, 0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zare := range z.MetaTags {
		if z.MetaTags[zare] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zare].Tag)))
			for zljy := range z.MetaTags[zare].Tag {
				o = msgp.AppendString(o, z.MetaTags[zare].Tag[zljy])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SeriesMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zaoz uint32
	zaoz, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zaoz > 0 {
		zaoz--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
			z.Ttl, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "tags":
			var zfzb uint32
			zfzb, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zfzb) {
				z.Tags = (z.Tags)[:zfzb]
			} else {
				z.Tags = make([]*MetricTag, zfzb)
			}
			for ztyy := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[ztyy] = nil
				} else {
					if z.Tags[ztyy] == nil {
						z.Tags[ztyy] = new(MetricTag)
					}
					var zsbo uint32
					zsbo, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zsbo > 0 {
						zsbo--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zjif uint32
							zjif, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[ztyy].Tag) >= int(zjif) {
								z.Tags[ztyy].Tag = (z.Tags[ztyy].Tag)[:zjif]
							} else {
								z.Tags[ztyy].Tag = make([]string, zjif)
							}
							for zinl := range z.Tags[ztyy].Tag {
								z.Tags[ztyy].Tag[zinl], bts, err = msgp.ReadStringBytes(bts)
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
		case "meta_tags":
			var zqgz uint32
			zqgz, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zqgz) {
				z.MetaTags = (z.MetaTags)[:zqgz]
			} else {
				z.MetaTags = make([]*MetricTag, zqgz)
			}
			for zare := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zare] = nil
				} else {
					if z.MetaTags[zare] == nil {
						z.MetaTags[zare] = new(MetricTag)
					}
					var zsnw uint32
					zsnw, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zsnw > 0 {
						zsnw--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var ztls uint32
							ztls, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[zare].Tag) >= int(ztls) {
								z.MetaTags[zare].Tag = (z.MetaTags[zare].Tag)[:ztls]
							} else {
								z.MetaTags[zare].Tag = make([]string, ztls)
							}
							for zljy := range z.MetaTags[zare].Tag {
								z.MetaTags[zare].Tag[zljy], bts, err = msgp.ReadStringBytes(bts)
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
	for ztyy := range z.Tags {
		if z.Tags[ztyy] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zinl := range z.Tags[ztyy].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[ztyy].Tag[zinl])
			}
		}
	}
	s += 10 + msgp.ArrayHeaderSize
	for zare := range z.MetaTags {
		if z.MetaTags[zare] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zljy := range z.MetaTags[zare].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[zare].Tag[zljy])
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SingleMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zedl uint32
	zedl, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zedl > 0 {
		zedl--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
			z.Ttl, err = dc.ReadUint32()
			if err != nil {
				return
			}
		case "tags":
			var zupd uint32
			zupd, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zupd) {
				z.Tags = (z.Tags)[:zupd]
			} else {
				z.Tags = make([]*MetricTag, zupd)
			}
			for zmvo := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zmvo] = nil
				} else {
					if z.Tags[zmvo] == nil {
						z.Tags[zmvo] = new(MetricTag)
					}
					var zome uint32
					zome, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zome > 0 {
						zome--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zrvj uint32
							zrvj, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zmvo].Tag) >= int(zrvj) {
								z.Tags[zmvo].Tag = (z.Tags[zmvo].Tag)[:zrvj]
							} else {
								z.Tags[zmvo].Tag = make([]string, zrvj)
							}
							for zigk := range z.Tags[zmvo].Tag {
								z.Tags[zmvo].Tag[zigk], err = dc.ReadString()
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
		case "meta_tags":
			var zarz uint32
			zarz, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zarz) {
				z.MetaTags = (z.MetaTags)[:zarz]
			} else {
				z.MetaTags = make([]*MetricTag, zarz)
			}
			for zopb := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zopb] = nil
				} else {
					if z.MetaTags[zopb] == nil {
						z.MetaTags[zopb] = new(MetricTag)
					}
					var zknt uint32
					zknt, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zknt > 0 {
						zknt--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zxye uint32
							zxye, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[zopb].Tag) >= int(zxye) {
								z.MetaTags[zopb].Tag = (z.MetaTags[zopb].Tag)[:zxye]
							} else {
								z.MetaTags[zopb].Tag = make([]string, zxye)
							}
							for zuop := range z.MetaTags[zopb].Tag {
								z.MetaTags[zopb].Tag[zuop], err = dc.ReadString()
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
	// write "id"
	err = en.Append(0x8d, 0xa2, 0x69, 0x64)
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
	err = en.WriteUint32(z.Ttl)
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
	for zmvo := range z.Tags {
		if z.Tags[zmvo] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.Tags[zmvo].Tag)))
			if err != nil {
				return
			}
			for zigk := range z.Tags[zmvo].Tag {
				err = en.WriteString(z.Tags[zmvo].Tag[zigk])
				if err != nil {
					return
				}
			}
		}
	}
	// write "meta_tags"
	err = en.Append(0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zopb := range z.MetaTags {
		if z.MetaTags[zopb] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[zopb].Tag)))
			if err != nil {
				return
			}
			for zuop := range z.MetaTags[zopb].Tag {
				err = en.WriteString(z.MetaTags[zopb].Tag[zuop])
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
	// string "id"
	o = append(o, 0x8d, 0xa2, 0x69, 0x64)
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
	o = msgp.AppendUint32(o, z.Ttl)
	// string "tags"
	o = append(o, 0xa4, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Tags)))
	for zmvo := range z.Tags {
		if z.Tags[zmvo] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zmvo].Tag)))
			for zigk := range z.Tags[zmvo].Tag {
				o = msgp.AppendString(o, z.Tags[zmvo].Tag[zigk])
			}
		}
	}
	// string "meta_tags"
	o = append(o, 0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zopb := range z.MetaTags {
		if z.MetaTags[zopb] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zopb].Tag)))
			for zuop := range z.MetaTags[zopb].Tag {
				o = msgp.AppendString(o, z.MetaTags[zopb].Tag[zuop])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SingleMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zucw uint32
	zucw, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zucw > 0 {
		zucw--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
			z.Ttl, bts, err = msgp.ReadUint32Bytes(bts)
			if err != nil {
				return
			}
		case "tags":
			var zlsx uint32
			zlsx, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zlsx) {
				z.Tags = (z.Tags)[:zlsx]
			} else {
				z.Tags = make([]*MetricTag, zlsx)
			}
			for zmvo := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zmvo] = nil
				} else {
					if z.Tags[zmvo] == nil {
						z.Tags[zmvo] = new(MetricTag)
					}
					var zbgy uint32
					zbgy, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zbgy > 0 {
						zbgy--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zrao uint32
							zrao, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zmvo].Tag) >= int(zrao) {
								z.Tags[zmvo].Tag = (z.Tags[zmvo].Tag)[:zrao]
							} else {
								z.Tags[zmvo].Tag = make([]string, zrao)
							}
							for zigk := range z.Tags[zmvo].Tag {
								z.Tags[zmvo].Tag[zigk], bts, err = msgp.ReadStringBytes(bts)
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
		case "meta_tags":
			var zmbt uint32
			zmbt, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zmbt) {
				z.MetaTags = (z.MetaTags)[:zmbt]
			} else {
				z.MetaTags = make([]*MetricTag, zmbt)
			}
			for zopb := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zopb] = nil
				} else {
					if z.MetaTags[zopb] == nil {
						z.MetaTags[zopb] = new(MetricTag)
					}
					var zvls uint32
					zvls, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zvls > 0 {
						zvls--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zjfj uint32
							zjfj, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[zopb].Tag) >= int(zjfj) {
								z.MetaTags[zopb].Tag = (z.MetaTags[zopb].Tag)[:zjfj]
							} else {
								z.MetaTags[zopb].Tag = make([]string, zjfj)
							}
							for zuop := range z.MetaTags[zopb].Tag {
								z.MetaTags[zopb].Tag[zuop], bts, err = msgp.ReadStringBytes(bts)
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
	for zmvo := range z.Tags {
		if z.Tags[zmvo] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zigk := range z.Tags[zmvo].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zmvo].Tag[zigk])
			}
		}
	}
	s += 10 + msgp.ArrayHeaderSize
	for zopb := range z.MetaTags {
		if z.MetaTags[zopb] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zuop := range z.MetaTags[zopb].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[zopb].Tag[zuop])
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *UnProcessedMetric) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zdtr uint32
	zdtr, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zdtr > 0 {
		zdtr--
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
			var zzqm uint32
			zzqm, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zzqm) {
				z.Tags = (z.Tags)[:zzqm]
			} else {
				z.Tags = make([]*MetricTag, zzqm)
			}
			for zzak := range z.Tags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Tags[zzak] = nil
				} else {
					if z.Tags[zzak] == nil {
						z.Tags[zzak] = new(MetricTag)
					}
					var zdqi uint32
					zdqi, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zdqi > 0 {
						zdqi--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zyco uint32
							zyco, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.Tags[zzak].Tag) >= int(zyco) {
								z.Tags[zzak].Tag = (z.Tags[zzak].Tag)[:zyco]
							} else {
								z.Tags[zzak].Tag = make([]string, zyco)
							}
							for zbtz := range z.Tags[zzak].Tag {
								z.Tags[zzak].Tag[zbtz], err = dc.ReadString()
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
		case "meta_tags":
			var zhgh uint32
			zhgh, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zhgh) {
				z.MetaTags = (z.MetaTags)[:zhgh]
			} else {
				z.MetaTags = make([]*MetricTag, zhgh)
			}
			for zsym := range z.MetaTags {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.MetaTags[zsym] = nil
				} else {
					if z.MetaTags[zsym] == nil {
						z.MetaTags[zsym] = new(MetricTag)
					}
					var zovg uint32
					zovg, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					for zovg > 0 {
						zovg--
						field, err = dc.ReadMapKeyPtr()
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zsey uint32
							zsey, err = dc.ReadArrayHeader()
							if err != nil {
								return
							}
							if cap(z.MetaTags[zsym].Tag) >= int(zsey) {
								z.MetaTags[zsym].Tag = (z.MetaTags[zsym].Tag)[:zsey]
							} else {
								z.MetaTags[zsym].Tag = make([]string, zsey)
							}
							for zgeu := range z.MetaTags[zsym].Tag {
								z.MetaTags[zsym].Tag[zgeu], err = dc.ReadString()
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
	for zzak := range z.Tags {
		if z.Tags[zzak] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.Tags[zzak].Tag)))
			if err != nil {
				return
			}
			for zbtz := range z.Tags[zzak].Tag {
				err = en.WriteString(z.Tags[zzak].Tag[zbtz])
				if err != nil {
					return
				}
			}
		}
	}
	// write "meta_tags"
	err = en.Append(0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.MetaTags)))
	if err != nil {
		return
	}
	for zsym := range z.MetaTags {
		if z.MetaTags[zsym] == nil {
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
			err = en.WriteArrayHeader(uint32(len(z.MetaTags[zsym].Tag)))
			if err != nil {
				return
			}
			for zgeu := range z.MetaTags[zsym].Tag {
				err = en.WriteString(z.MetaTags[zsym].Tag[zgeu])
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
	for zzak := range z.Tags {
		if z.Tags[zzak] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.Tags[zzak].Tag)))
			for zbtz := range z.Tags[zzak].Tag {
				o = msgp.AppendString(o, z.Tags[zzak].Tag[zbtz])
			}
		}
	}
	// string "meta_tags"
	o = append(o, 0xa9, 0x6d, 0x65, 0x74, 0x61, 0x5f, 0x74, 0x61, 0x67, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags)))
	for zsym := range z.MetaTags {
		if z.MetaTags[zsym] == nil {
			o = msgp.AppendNil(o)
		} else {
			// map header, size 1
			// string "Tag"
			o = append(o, 0x81, 0xa3, 0x54, 0x61, 0x67)
			o = msgp.AppendArrayHeader(o, uint32(len(z.MetaTags[zsym].Tag)))
			for zgeu := range z.MetaTags[zsym].Tag {
				o = msgp.AppendString(o, z.MetaTags[zsym].Tag[zgeu])
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *UnProcessedMetric) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zcjp uint32
	zcjp, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zcjp > 0 {
		zcjp--
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
			var zjhy uint32
			zjhy, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Tags) >= int(zjhy) {
				z.Tags = (z.Tags)[:zjhy]
			} else {
				z.Tags = make([]*MetricTag, zjhy)
			}
			for zzak := range z.Tags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Tags[zzak] = nil
				} else {
					if z.Tags[zzak] == nil {
						z.Tags[zzak] = new(MetricTag)
					}
					var znuf uint32
					znuf, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for znuf > 0 {
						znuf--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var znjj uint32
							znjj, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.Tags[zzak].Tag) >= int(znjj) {
								z.Tags[zzak].Tag = (z.Tags[zzak].Tag)[:znjj]
							} else {
								z.Tags[zzak].Tag = make([]string, znjj)
							}
							for zbtz := range z.Tags[zzak].Tag {
								z.Tags[zzak].Tag[zbtz], bts, err = msgp.ReadStringBytes(bts)
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
		case "meta_tags":
			var zhhj uint32
			zhhj, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.MetaTags) >= int(zhhj) {
				z.MetaTags = (z.MetaTags)[:zhhj]
			} else {
				z.MetaTags = make([]*MetricTag, zhhj)
			}
			for zsym := range z.MetaTags {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.MetaTags[zsym] = nil
				} else {
					if z.MetaTags[zsym] == nil {
						z.MetaTags[zsym] = new(MetricTag)
					}
					var zuvr uint32
					zuvr, bts, err = msgp.ReadMapHeaderBytes(bts)
					if err != nil {
						return
					}
					for zuvr > 0 {
						zuvr--
						field, bts, err = msgp.ReadMapKeyZC(bts)
						if err != nil {
							return
						}
						switch msgp.UnsafeString(field) {
						case "Tag":
							var zusq uint32
							zusq, bts, err = msgp.ReadArrayHeaderBytes(bts)
							if err != nil {
								return
							}
							if cap(z.MetaTags[zsym].Tag) >= int(zusq) {
								z.MetaTags[zsym].Tag = (z.MetaTags[zsym].Tag)[:zusq]
							} else {
								z.MetaTags[zsym].Tag = make([]string, zusq)
							}
							for zgeu := range z.MetaTags[zsym].Tag {
								z.MetaTags[zsym].Tag[zgeu], bts, err = msgp.ReadStringBytes(bts)
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
	for zzak := range z.Tags {
		if z.Tags[zzak] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zbtz := range z.Tags[zzak].Tag {
				s += msgp.StringPrefixSize + len(z.Tags[zzak].Tag[zbtz])
			}
		}
	}
	s += 10 + msgp.ArrayHeaderSize
	for zsym := range z.MetaTags {
		if z.MetaTags[zsym] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + 4 + msgp.ArrayHeaderSize
			for zgeu := range z.MetaTags[zsym].Tag {
				s += msgp.StringPrefixSize + len(z.MetaTags[zsym].Tag[zgeu])
			}
		}
	}
	return
}
