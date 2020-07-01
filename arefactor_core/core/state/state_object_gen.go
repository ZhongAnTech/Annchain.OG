package state

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *AccountData) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 5 {
		err = msgp.ArrayError{Wanted: 5, Got: zb0001}
		return
	}
	err = z.Address.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	//err = z.Balances.DecodeMsg(dc)
	//if err != nil {
	//	err = msgp.WrapError(err, "Balances")
	//	return
	//}
	z.Nonce, err = dc.ReadUint64()
	if err != nil {
		err = msgp.WrapError(err, "Nonce")
		return
	}
	err = z.Root.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Root")
		return
	}
	z.CodeHash, err = dc.ReadBytes(z.CodeHash)
	if err != nil {
		err = msgp.WrapError(err, "CodeHash")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *AccountData) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 5
	err = en.Append(0x95)
	if err != nil {
		return
	}
	err = z.Address.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	//err = z.Balances.EncodeMsg(en)
	//if err != nil {
	//	err = msgp.WrapError(err, "Balances")
	//	return
	//}
	err = en.WriteUint64(z.Nonce)
	if err != nil {
		err = msgp.WrapError(err, "Nonce")
		return
	}
	err = z.Root.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Root")
		return
	}
	err = en.WriteBytes(z.CodeHash)
	if err != nil {
		err = msgp.WrapError(err, "CodeHash")
		return
	}
	return
}

//// MarshalMsg implements msgp.Marshaler
//func (z *AccountData) MarshalMsg(b []byte) (o []byte, err error) {
//	o = msgp.Require(b, z.Msgsize())
//	// array header, size 5
//	o = append(o, 0x95)
//	o, err = z.Address.MarshalMsg(o)
//	if err != nil {
//		err = msgp.WrapError(err, "Address")
//		return
//	}
//	o, err = z.Balances.MarshalMsg(o)
//	if err != nil {
//		err = msgp.WrapError(err, "Balances")
//		return
//	}
//	o = msgp.AppendUint64(o, z.Nonce)
//	o, err = z.Root.MarshalMsg(o)
//	if err != nil {
//		err = msgp.WrapError(err, "Root")
//		return
//	}
//	o = msgp.AppendBytes(o, z.CodeHash)
//	return
//}
//
//// UnmarshalMsg implements msgp.Unmarshaler
//func (z *AccountData) UnmarshalMsg(bts []byte) (o []byte, err error) {
//	var zb0001 uint32
//	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
//	if err != nil {
//		err = msgp.WrapError(err)
//		return
//	}
//	if zb0001 != 5 {
//		err = msgp.ArrayError{Wanted: 5, Got: zb0001}
//		return
//	}
//	bts, err = z.Address.UnmarshalMsg(bts)
//	if err != nil {
//		err = msgp.WrapError(err, "Address")
//		return
//	}
//	bts, err = z.Balances.UnmarshalMsg(bts)
//	if err != nil {
//		err = msgp.WrapError(err, "Balances")
//		return
//	}
//	z.Nonce, bts, err = msgp.ReadUint64Bytes(bts)
//	if err != nil {
//		err = msgp.WrapError(err, "Nonce")
//		return
//	}
//	bts, err = z.Root.UnmarshalMsg(bts)
//	if err != nil {
//		err = msgp.WrapError(err, "Root")
//		return
//	}
//	z.CodeHash, bts, err = msgp.ReadBytesBytes(bts, z.CodeHash)
//	if err != nil {
//		err = msgp.WrapError(err, "CodeHash")
//		return
//	}
//	o = bts
//	return
//}
//
//// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
//func (z *AccountData) Msgsize() (s int) {
//	s = 1 + z.Address.Msgsize() + z.Balances.Msgsize() + msgp.Uint64Size + z.Root.Msgsize() + msgp.BytesPrefixSize + len(z.CodeHash)
//	return
//}

// DecodeMsg implements msgp.Decodable
func (z *StateObject) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z StateObject) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 0
	err = en.Append(0x80)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z StateObject) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 0
	o = append(o, 0x80)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *StateObject) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z StateObject) Msgsize() (s int) {
	s = 1
	return
}
