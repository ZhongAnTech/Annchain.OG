package transport

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *WireMessage) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: zb0001}
		return
	}
	z.MsgType, err = dc.ReadInt()
	if err != nil {
		err = msgp.WrapError(err, "MsgType")
		return
	}
	z.SenderId, err = dc.ReadString()
	if err != nil {
		err = msgp.WrapError(err, "SenderId")
		return
	}
	z.ContentBytes, err = dc.ReadBytes(z.ContentBytes)
	if err != nil {
		err = msgp.WrapError(err, "ContentBytes")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *WireMessage) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 3
	err = en.Append(0x93)
	if err != nil {
		return
	}
	err = en.WriteInt(z.MsgType)
	if err != nil {
		err = msgp.WrapError(err, "MsgType")
		return
	}
	err = en.WriteString(z.SenderId)
	if err != nil {
		err = msgp.WrapError(err, "SenderId")
		return
	}
	err = en.WriteBytes(z.ContentBytes)
	if err != nil {
		err = msgp.WrapError(err, "ContentBytes")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *WireMessage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 3
	o = append(o, 0x93)
	o = msgp.AppendInt(o, z.MsgType)
	o = msgp.AppendString(o, z.SenderId)
	o = msgp.AppendBytes(o, z.ContentBytes)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *WireMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 3 {
		err = msgp.ArrayError{Wanted: 3, Got: zb0001}
		return
	}
	z.MsgType, bts, err = msgp.ReadIntBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "MsgType")
		return
	}
	z.SenderId, bts, err = msgp.ReadStringBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "SenderId")
		return
	}
	z.ContentBytes, bts, err = msgp.ReadBytesBytes(bts, z.ContentBytes)
	if err != nil {
		err = msgp.WrapError(err, "ContentBytes")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *WireMessage) Msgsize() (s int) {
	s = 1 + msgp.IntSize + msgp.StringPrefixSize + len(z.SenderId) + msgp.BytesPrefixSize + len(z.ContentBytes)
	return
}