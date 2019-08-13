package communicator

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *SignedOgParnterMessage) DecodeMsg(dc *msgp.Reader) (err error) {
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
	err = z.BftMessage.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "BftMessage")
		return
	}
	z.TermId, err = dc.ReadUint32()
	if err != nil {
		err = msgp.WrapError(err, "TermId")
		return
	}
	z.ValidRound, err = dc.ReadInt()
	if err != nil {
		err = msgp.WrapError(err, "ValidRound")
		return
	}
	err = z.Signature.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Signature")
		return
	}
	err = z.PublicKey.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "PublicKey")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *SignedOgParnterMessage) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 5
	err = en.Append(0x95)
	if err != nil {
		return
	}
	err = z.BftMessage.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "BftMessage")
		return
	}
	err = en.WriteUint32(z.TermId)
	if err != nil {
		err = msgp.WrapError(err, "TermId")
		return
	}
	err = en.WriteInt(z.ValidRound)
	if err != nil {
		err = msgp.WrapError(err, "ValidRound")
		return
	}
	err = z.Signature.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Signature")
		return
	}
	err = z.PublicKey.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "PublicKey")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SignedOgParnterMessage) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 5
	o = append(o, 0x95)
	o, err = z.BftMessage.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "BftMessage")
		return
	}
	o = msgp.AppendUint32(o, z.TermId)
	o = msgp.AppendInt(o, z.ValidRound)
	o, err = z.Signature.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Signature")
		return
	}
	o, err = z.PublicKey.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "PublicKey")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SignedOgParnterMessage) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 5 {
		err = msgp.ArrayError{Wanted: 5, Got: zb0001}
		return
	}
	bts, err = z.BftMessage.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "BftMessage")
		return
	}
	z.TermId, bts, err = msgp.ReadUint32Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "TermId")
		return
	}
	z.ValidRound, bts, err = msgp.ReadIntBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "ValidRound")
		return
	}
	bts, err = z.Signature.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "Signature")
		return
	}
	bts, err = z.PublicKey.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "PublicKey")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *SignedOgParnterMessage) Msgsize() (s int) {
	s = 1 + z.BftMessage.Msgsize() + msgp.Uint32Size + msgp.IntSize + z.Signature.Msgsize() + z.PublicKey.Msgsize()
	return
}
