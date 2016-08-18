package msgpacker

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *FullStat) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "t":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "n":
			z.Min, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "m":
			z.Max, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "s":
			z.Sum, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "l":
			z.Last, err = dc.ReadFloat64()
			if err != nil {
				return
			}
		case "c":
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
func (z *FullStat) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "t"
	err = en.Append(0x86, 0xa1, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "n"
	err = en.Append(0xa1, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Min)
	if err != nil {
		return
	}
	// write "m"
	err = en.Append(0xa1, 0x6d)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Max)
	if err != nil {
		return
	}
	// write "s"
	err = en.Append(0xa1, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Sum)
	if err != nil {
		return
	}
	// write "l"
	err = en.Append(0xa1, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Last)
	if err != nil {
		return
	}
	// write "c"
	err = en.Append(0xa1, 0x63)
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
func (z *FullStat) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "t"
	o = append(o, 0x86, 0xa1, 0x74)
	o = msgp.AppendInt64(o, z.Time)
	// string "n"
	o = append(o, 0xa1, 0x6e)
	o = msgp.AppendFloat64(o, z.Min)
	// string "m"
	o = append(o, 0xa1, 0x6d)
	o = msgp.AppendFloat64(o, z.Max)
	// string "s"
	o = append(o, 0xa1, 0x73)
	o = msgp.AppendFloat64(o, z.Sum)
	// string "l"
	o = append(o, 0xa1, 0x6c)
	o = msgp.AppendFloat64(o, z.Last)
	// string "c"
	o = append(o, 0xa1, 0x63)
	o = msgp.AppendInt64(o, z.Count)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *FullStat) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "t":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "n":
			z.Min, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "m":
			z.Max, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "s":
			z.Sum, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "l":
			z.Last, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				return
			}
		case "c":
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
func (z *FullStat) Msgsize() (s int) {
	s = 1 + 2 + msgp.Int64Size + 2 + msgp.Float64Size + 2 + msgp.Float64Size + 2 + msgp.Float64Size + 2 + msgp.Float64Size + 2 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Stat) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zbai uint32
	zbai, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zbai > 0 {
		zbai--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "t":
			z.StatType, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "s":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.Stat = nil
			} else {
				if z.Stat == nil {
					z.Stat = new(FullStat)
				}
				err = z.Stat.DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "m":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					return
				}
				z.SmallStat = nil
			} else {
				if z.SmallStat == nil {
					z.SmallStat = new(StatSmall)
				}
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
					case "t":
						z.SmallStat.Time, err = dc.ReadInt64()
						if err != nil {
							return
						}
					case "v":
						z.SmallStat.Val, err = dc.ReadFloat64()
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
func (z *Stat) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "t"
	err = en.Append(0x83, 0xa1, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.StatType)
	if err != nil {
		return
	}
	// write "s"
	err = en.Append(0xa1, 0x73)
	if err != nil {
		return err
	}
	if z.Stat == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Stat.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "m"
	err = en.Append(0xa1, 0x6d)
	if err != nil {
		return err
	}
	if z.SmallStat == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		// map header, size 2
		// write "t"
		err = en.Append(0x82, 0xa1, 0x74)
		if err != nil {
			return err
		}
		err = en.WriteInt64(z.SmallStat.Time)
		if err != nil {
			return
		}
		// write "v"
		err = en.Append(0xa1, 0x76)
		if err != nil {
			return err
		}
		err = en.WriteFloat64(z.SmallStat.Val)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Stat) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "t"
	o = append(o, 0x83, 0xa1, 0x74)
	o = msgp.AppendBool(o, z.StatType)
	// string "s"
	o = append(o, 0xa1, 0x73)
	if z.Stat == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Stat.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	// string "m"
	o = append(o, 0xa1, 0x6d)
	if z.SmallStat == nil {
		o = msgp.AppendNil(o)
	} else {
		// map header, size 2
		// string "t"
		o = append(o, 0x82, 0xa1, 0x74)
		o = msgp.AppendInt64(o, z.SmallStat.Time)
		// string "v"
		o = append(o, 0xa1, 0x76)
		o = msgp.AppendFloat64(o, z.SmallStat.Val)
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Stat) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zajw uint32
	zajw, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zajw > 0 {
		zajw--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "t":
			z.StatType, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "s":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Stat = nil
			} else {
				if z.Stat == nil {
					z.Stat = new(FullStat)
				}
				bts, err = z.Stat.UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		case "m":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.SmallStat = nil
			} else {
				if z.SmallStat == nil {
					z.SmallStat = new(StatSmall)
				}
				var zwht uint32
				zwht, bts, err = msgp.ReadMapHeaderBytes(bts)
				if err != nil {
					return
				}
				for zwht > 0 {
					zwht--
					field, bts, err = msgp.ReadMapKeyZC(bts)
					if err != nil {
						return
					}
					switch msgp.UnsafeString(field) {
					case "t":
						z.SmallStat.Time, bts, err = msgp.ReadInt64Bytes(bts)
						if err != nil {
							return
						}
					case "v":
						z.SmallStat.Val, bts, err = msgp.ReadFloat64Bytes(bts)
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
func (z *Stat) Msgsize() (s int) {
	s = 1 + 2 + msgp.BoolSize + 2
	if z.Stat == nil {
		s += msgp.NilSize
	} else {
		s += z.Stat.Msgsize()
	}
	s += 2
	if z.SmallStat == nil {
		s += msgp.NilSize
	} else {
		s += 1 + 2 + msgp.Int64Size + 2 + msgp.Float64Size
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *StatSmall) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "t":
			z.Time, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "v":
			z.Val, err = dc.ReadFloat64()
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
func (z StatSmall) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "t"
	err = en.Append(0x82, 0xa1, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Time)
	if err != nil {
		return
	}
	// write "v"
	err = en.Append(0xa1, 0x76)
	if err != nil {
		return err
	}
	err = en.WriteFloat64(z.Val)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z StatSmall) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "t"
	o = append(o, 0x82, 0xa1, 0x74)
	o = msgp.AppendInt64(o, z.Time)
	// string "v"
	o = append(o, 0xa1, 0x76)
	o = msgp.AppendFloat64(o, z.Val)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *StatSmall) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "t":
			z.Time, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "v":
			z.Val, bts, err = msgp.ReadFloat64Bytes(bts)
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
func (z StatSmall) Msgsize() (s int) {
	s = 1 + 2 + msgp.Int64Size + 2 + msgp.Float64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Stats) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "r":
			z.FullTimeResolution, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "s":
			var zdaf uint32
			zdaf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Stats) >= int(zdaf) {
				z.Stats = (z.Stats)[:zdaf]
			} else {
				z.Stats = make([]*Stat, zdaf)
			}
			for zxhx := range z.Stats {
				if dc.IsNil() {
					err = dc.ReadNil()
					if err != nil {
						return
					}
					z.Stats[zxhx] = nil
				} else {
					if z.Stats[zxhx] == nil {
						z.Stats[zxhx] = new(Stat)
					}
					err = z.Stats[zxhx].DecodeMsg(dc)
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
func (z *Stats) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "r"
	err = en.Append(0x82, 0xa1, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.FullTimeResolution)
	if err != nil {
		return
	}
	// write "s"
	err = en.Append(0xa1, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Stats)))
	if err != nil {
		return
	}
	for zxhx := range z.Stats {
		if z.Stats[zxhx] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z.Stats[zxhx].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Stats) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "r"
	o = append(o, 0x82, 0xa1, 0x72)
	o = msgp.AppendBool(o, z.FullTimeResolution)
	// string "s"
	o = append(o, 0xa1, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Stats)))
	for zxhx := range z.Stats {
		if z.Stats[zxhx] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z.Stats[zxhx].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Stats) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "r":
			z.FullTimeResolution, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "s":
			var zjfb uint32
			zjfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Stats) >= int(zjfb) {
				z.Stats = (z.Stats)[:zjfb]
			} else {
				z.Stats = make([]*Stat, zjfb)
			}
			for zxhx := range z.Stats {
				if msgp.IsNil(bts) {
					bts, err = msgp.ReadNilBytes(bts)
					if err != nil {
						return
					}
					z.Stats[zxhx] = nil
				} else {
					if z.Stats[zxhx] == nil {
						z.Stats[zxhx] = new(Stat)
					}
					bts, err = z.Stats[zxhx].UnmarshalMsg(bts)
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
func (z *Stats) Msgsize() (s int) {
	s = 1 + 2 + msgp.BoolSize + 2 + msgp.ArrayHeaderSize
	for zxhx := range z.Stats {
		if z.Stats[zxhx] == nil {
			s += msgp.NilSize
		} else {
			s += z.Stats[zxhx].Msgsize()
		}
	}
	return
}
