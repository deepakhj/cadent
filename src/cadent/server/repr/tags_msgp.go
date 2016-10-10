package repr

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *SortingTags) DecodeMsg(dc *msgp.Reader) (err error) {
	var zbai uint32
	zbai, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zbai) {
		(*z) = (*z)[:zbai]
	} else {
		(*z) = make(SortingTags, zbai)
	}
	for zbzg := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zbzg] = nil
		} else {
			if (*z)[zbzg] == nil {
				(*z)[zbzg] = new(Tag)
			}
			err = (*z)[zbzg].DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z SortingTags) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zcmr := range z {
		if z[zcmr] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z[zcmr].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z SortingTags) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zcmr := range z {
		if z[zcmr] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zcmr].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SortingTags) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zwht uint32
	zwht, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zwht) {
		(*z) = (*z)[:zwht]
	} else {
		(*z) = make(SortingTags, zwht)
	}
	for zajw := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zajw] = nil
		} else {
			if (*z)[zajw] == nil {
				(*z)[zajw] = new(Tag)
			}
			bts, err = (*z)[zajw].UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z SortingTags) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zhct := range z {
		if z[zhct] == nil {
			s += msgp.NilSize
		} else {
			s += z[zhct].Msgsize()
		}
	}
	return
}
