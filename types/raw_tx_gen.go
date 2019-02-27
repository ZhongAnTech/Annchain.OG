package types

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/annchain/OG/common/math"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *RawCampaign) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0001 != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: zb0001}
		return
	}
	err = z.TxBase.DecodeMsg(dc)
	if err != nil {
		return
	}
	z.DkgPublicKey, err = dc.ReadBytes(z.DkgPublicKey)
	if err != nil {
		return
	}
	err = z.Vrf.DecodeMsg(dc)
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RawCampaign) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 3
	err = en.Append(0x93)
	if err != nil {
		return
	}
	err = z.TxBase.EncodeMsg(en)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.DkgPublicKey)
	if err != nil {
		return
	}
	err = z.Vrf.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawCampaign) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 3
	o = append(o, 0x93)
	o, err = z.TxBase.MarshalMsg(o)
	if err != nil {
		return
	}
	o = msgp.AppendBytes(o, z.DkgPublicKey)
	o, err = z.Vrf.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawCampaign) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0001 != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: zb0001}
		return
	}
	bts, err = z.TxBase.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	z.DkgPublicKey, bts, err = msgp.ReadBytesBytes(bts, z.DkgPublicKey)
	if err != nil {
		return
	}
	bts, err = z.Vrf.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RawCampaign) Msgsize() (s int) {
	s = 1 + z.TxBase.Msgsize() + msgp.BytesPrefixSize + len(z.DkgPublicKey) + z.Vrf.Msgsize()
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawCampaigns) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(RawCampaigns, zb0002)
	}
	for zb0001 := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(RawCampaign)
			}
			var zb0003 uint32
			zb0003, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if zb0003 != 3 {
				err = msgp.ArrayError{Wanted: 3, Got: zb0003}
				return
			}
			err = (*z)[zb0001].TxBase.DecodeMsg(dc)
			if err != nil {
				return
			}
			(*z)[zb0001].DkgPublicKey, err = dc.ReadBytes((*z)[zb0001].DkgPublicKey)
			if err != nil {
				return
			}
			err = (*z)[zb0001].Vrf.DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RawCampaigns) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zb0004 := range z {
		if z[zb0004] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// array header, size 3
			err = en.Append(0x93)
			if err != nil {
				return
			}
			err = z[zb0004].TxBase.EncodeMsg(en)
			if err != nil {
				return
			}
			err = en.WriteBytes(z[zb0004].DkgPublicKey)
			if err != nil {
				return
			}
			err = z[zb0004].Vrf.EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RawCampaigns) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zb0004 := range z {
		if z[zb0004] == nil {
			o = msgp.AppendNil(o)
		} else {
			// array header, size 3
			o = append(o, 0x93)
			o, err = z[zb0004].TxBase.MarshalMsg(o)
			if err != nil {
				return
			}
			o = msgp.AppendBytes(o, z[zb0004].DkgPublicKey)
			o, err = z[zb0004].Vrf.MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawCampaigns) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(RawCampaigns, zb0002)
	}
	for zb0001 := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(RawCampaign)
			}
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if zb0003 != 3 {
				err = msgp.ArrayError{Wanted: 3, Got: zb0003}
				return
			}
			bts, err = (*z)[zb0001].TxBase.UnmarshalMsg(bts)
			if err != nil {
				return
			}
			(*z)[zb0001].DkgPublicKey, bts, err = msgp.ReadBytesBytes(bts, (*z)[zb0001].DkgPublicKey)
			if err != nil {
				return
			}
			bts, err = (*z)[zb0001].Vrf.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z RawCampaigns) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zb0004 := range z {
		if z[zb0004] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + z[zb0004].TxBase.Msgsize() + msgp.BytesPrefixSize + len(z[zb0004].DkgPublicKey) + z[zb0004].Vrf.Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawSequencer) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	err = z.TxBase.DecodeMsg(dc)
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RawSequencer) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = z.TxBase.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawSequencer) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o, err = z.TxBase.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawSequencer) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	bts, err = z.TxBase.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RawSequencer) Msgsize() (s int) {
	s = 1 + z.TxBase.Msgsize()
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawSequencers) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(RawSequencers, zb0002)
	}
	for zb0001 := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(RawSequencer)
			}
			var zb0003 uint32
			zb0003, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if zb0003 != 1 {
				err = msgp.ArrayError{Wanted: 1, Got: zb0003}
				return
			}
			err = (*z)[zb0001].TxBase.DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RawSequencers) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zb0004 := range z {
		if z[zb0004] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// array header, size 1
			err = en.Append(0x91)
			if err != nil {
				return
			}
			err = z[zb0004].TxBase.EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RawSequencers) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zb0004 := range z {
		if z[zb0004] == nil {
			o = msgp.AppendNil(o)
		} else {
			// array header, size 1
			o = append(o, 0x91)
			o, err = z[zb0004].TxBase.MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawSequencers) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(RawSequencers, zb0002)
	}
	for zb0001 := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(RawSequencer)
			}
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if zb0003 != 1 {
				err = msgp.ArrayError{Wanted: 1, Got: zb0003}
				return
			}
			bts, err = (*z)[zb0001].TxBase.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z RawSequencers) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zb0004 := range z {
		if z[zb0004] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + z[zb0004].TxBase.Msgsize()
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawTermChange) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	err = z.TxBase.DecodeMsg(dc)
	if err != nil {
		return
	}
	z.PkBls, err = dc.ReadBytes(z.PkBls)
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RawTermChange) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = z.TxBase.EncodeMsg(en)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.PkBls)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawTermChange) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o, err = z.TxBase.MarshalMsg(o)
	if err != nil {
		return
	}
	o = msgp.AppendBytes(o, z.PkBls)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawTermChange) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	bts, err = z.TxBase.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	z.PkBls, bts, err = msgp.ReadBytesBytes(bts, z.PkBls)
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RawTermChange) Msgsize() (s int) {
	s = 1 + z.TxBase.Msgsize() + msgp.BytesPrefixSize + len(z.PkBls)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawTermChanges) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(RawTermChanges, zb0002)
	}
	for zb0001 := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(RawTermChange)
			}
			var zb0003 uint32
			zb0003, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if zb0003 != 2 {
				err = msgp.ArrayError{Wanted: 2, Got: zb0003}
				return
			}
			err = (*z)[zb0001].TxBase.DecodeMsg(dc)
			if err != nil {
				return
			}
			(*z)[zb0001].PkBls, err = dc.ReadBytes((*z)[zb0001].PkBls)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RawTermChanges) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zb0004 := range z {
		if z[zb0004] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			// array header, size 2
			err = en.Append(0x92)
			if err != nil {
				return
			}
			err = z[zb0004].TxBase.EncodeMsg(en)
			if err != nil {
				return
			}
			err = en.WriteBytes(z[zb0004].PkBls)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RawTermChanges) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zb0004 := range z {
		if z[zb0004] == nil {
			o = msgp.AppendNil(o)
		} else {
			// array header, size 2
			o = append(o, 0x92)
			o, err = z[zb0004].TxBase.MarshalMsg(o)
			if err != nil {
				return
			}
			o = msgp.AppendBytes(o, z[zb0004].PkBls)
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawTermChanges) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(RawTermChanges, zb0002)
	}
	for zb0001 := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(RawTermChange)
			}
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if zb0003 != 2 {
				err = msgp.ArrayError{Wanted: 2, Got: zb0003}
				return
			}
			bts, err = (*z)[zb0001].TxBase.UnmarshalMsg(bts)
			if err != nil {
				return
			}
			(*z)[zb0001].PkBls, bts, err = msgp.ReadBytesBytes(bts, (*z)[zb0001].PkBls)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z RawTermChanges) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zb0004 := range z {
		if z[zb0004] == nil {
			s += msgp.NilSize
		} else {
			s += 1 + z[zb0004].TxBase.Msgsize() + msgp.BytesPrefixSize + len(z[zb0004].PkBls)
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawTx) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0001 != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: zb0001}
		return
	}
	err = z.TxBase.DecodeMsg(dc)
	if err != nil {
		return
	}
	err = z.To.DecodeMsg(dc)
	if err != nil {
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			return
		}
		z.Value = nil
	} else {
		if z.Value == nil {
			z.Value = new(math.BigInt)
		}
		err = z.Value.DecodeMsg(dc)
		if err != nil {
			return
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *RawTx) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 3
	err = en.Append(0x93)
	if err != nil {
		return
	}
	err = z.TxBase.EncodeMsg(en)
	if err != nil {
		return
	}
	err = z.To.EncodeMsg(en)
	if err != nil {
		return
	}
	if z.Value == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Value.EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *RawTx) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 3
	o = append(o, 0x93)
	o, err = z.TxBase.MarshalMsg(o)
	if err != nil {
		return
	}
	o, err = z.To.MarshalMsg(o)
	if err != nil {
		return
	}
	if z.Value == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Value.MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawTx) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0001 != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: zb0001}
		return
	}
	bts, err = z.TxBase.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	bts, err = z.To.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.Value = nil
	} else {
		if z.Value == nil {
			z.Value = new(math.BigInt)
		}
		bts, err = z.Value.UnmarshalMsg(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RawTx) Msgsize() (s int) {
	s = 1 + z.TxBase.Msgsize() + z.To.Msgsize()
	if z.Value == nil {
		s += msgp.NilSize
	} else {
		s += z.Value.Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *RawTxs) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(RawTxs, zb0002)
	}
	for zb0001 := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(RawTx)
			}
			err = (*z)[zb0001].DecodeMsg(dc)
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z RawTxs) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zb0003 := range z {
		if z[zb0003] == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = z[zb0003].EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z RawTxs) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zb0003 := range z {
		if z[zb0003] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zb0003].MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *RawTxs) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(RawTxs, zb0002)
	}
	for zb0001 := range *z {
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(RawTx)
			}
			bts, err = (*z)[zb0001].UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z RawTxs) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zb0003 := range z {
		if z[zb0003] == nil {
			s += msgp.NilSize
		} else {
			s += z[zb0003].Msgsize()
		}
	}
	return
}
