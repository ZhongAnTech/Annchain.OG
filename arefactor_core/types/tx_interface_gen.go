package types

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

//// DecodeMsg implements msgp.Decodable
//func (z *RawTxis) DecodeMsg(dc *msgp.Reader) (err error) {
//	var zb0002 uint32
//	zb0002, err = dc.ReadArrayHeader()
//	if err != nil {
//		return
//	}
//	if cap((*z)) >= int(zb0002) {
//		(*z) = (*z)[:zb0002]
//	} else {
//		(*z) = make(RawTxis, zb0002)
//	}
//	for zb0001 := range *z {
//		err = (*z)[zb0001].DecodeMsg(dc)
//		if err != nil {
//			return
//		}
//	}
//	return
//}
//
//// EncodeMsg implements msgp.Encodable
//func (z RawTxis) EncodeMsg(en *msgp.Writer) (err error) {
//	err = en.WriteArrayHeader(uint32(len(z)))
//	if err != nil {
//		return
//	}
//	for zb0003 := range z {
//		err = z[zb0003].EncodeMsg(en)
//		if err != nil {
//			return
//		}
//	}
//	return
//}
//
//// MarshalMsg implements msgp.Marshaler
//func (z RawTxis) MarshalMsg(b []byte) (o []byte, err error) {
//	o = msgp.Require(b, z.Msgsize())
//	o = msgp.AppendArrayHeader(o, uint32(len(z)))
//	for zb0003 := range z {
//		o, err = z[zb0003].MarshalMsg(o)
//		if err != nil {
//			return
//		}
//	}
//	return
//}
//
//// UnmarshalMsg implements msgp.Unmarshaler
//func (z *RawTxis) UnmarshalMsg(bts []byte) (o []byte, err error) {
//	var zb0002 uint32
//	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
//	if err != nil {
//		return
//	}
//	if cap((*z)) >= int(zb0002) {
//		(*z) = (*z)[:zb0002]
//	} else {
//		(*z) = make(RawTxis, zb0002)
//	}
//	for zb0001 := range *z {
//		bts, err = (*z)[zb0001].UnmarshalMsg(bts)
//		if err != nil {
//			return
//		}
//	}
//	o = bts
//	return
//}
//
//// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
//func (z RawTxis) Msgsize() (s int) {
//	s = msgp.ArrayHeaderSize
//	for zb0003 := range z {
//		s += z[zb0003].Msgsize()
//	}
//	return
//}

// DecodeMsg implements msgp.Decodable
func (z *TxiSmallCaseMarshal) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	err = z.Txi.DecodeMsg(dc)
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *TxiSmallCaseMarshal) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = z.Txi.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *TxiSmallCaseMarshal) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o, err = z.Txi.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *TxiSmallCaseMarshal) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	bts, err = z.Txi.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *TxiSmallCaseMarshal) Msgsize() (s int) {
	s = 1 + z.Txi.Msgsize()
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Txis) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(Txis, zb0002)
	}
	for zb0001 := range *z {
		err = (*z)[zb0001].DecodeMsg(dc)
		if err != nil {
			return
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Txis) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zb0003 := range z {
		err = z[zb0003].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Txis) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zb0003 := range z {
		o, err = z[zb0003].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Txis) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(Txis, zb0002)
	}
	for zb0001 := range *z {
		bts, err = (*z)[zb0001].UnmarshalMsg(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Txis) Msgsize() (s int) {
	s = msgp.ArrayHeaderSize
	for zb0003 := range z {
		s += z[zb0003].Msgsize()
	}
	return
}
