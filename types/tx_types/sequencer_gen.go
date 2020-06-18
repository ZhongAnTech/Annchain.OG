package tx_types

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/annchain/OG/common"
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *BlsSigSet) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.PublicKey, err = dc.ReadBytes(z.PublicKey)
	if err != nil {
		err = msgp.WrapError(err, "PublicKey")
		return
	}
	z.BlsSignature, err = dc.ReadBytes(z.BlsSignature)
	if err != nil {
		err = msgp.WrapError(err, "BlsSignature")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *BlsSigSet) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.PublicKey)
	if err != nil {
		err = msgp.WrapError(err, "PublicKey")
		return
	}
	err = en.WriteBytes(z.BlsSignature)
	if err != nil {
		err = msgp.WrapError(err, "BlsSignature")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *BlsSigSet) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendBytes(o, z.PublicKey)
	o = msgp.AppendBytes(o, z.BlsSignature)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *BlsSigSet) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.PublicKey, bts, err = msgp.ReadBytesBytes(bts, z.PublicKey)
	if err != nil {
		err = msgp.WrapError(err, "PublicKey")
		return
	}
	z.BlsSignature, bts, err = msgp.ReadBytesBytes(bts, z.BlsSignature)
	if err != nil {
		err = msgp.WrapError(err, "BlsSignature")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *BlsSigSet) Msgsize() (s int) {
	s = 1 + msgp.BytesPrefixSize + len(z.PublicKey) + msgp.BytesPrefixSize + len(z.BlsSignature)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Sequencer) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 6 {
		err = msgp.ArrayError{Wanted: 6, Got: zb0001}
		return
	}
	err = z.TxBase.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "TxBase")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
		z.Issuer = nil
	} else {
		if z.Issuer == nil {
			z.Issuer = new(common.Address)
		}
		err = z.Issuer.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
	}
	err = z.BlsJointSig.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	err = z.BlsJointPubKey.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	err = z.StateRoot.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "StateRoot")
		return
	}
	z.Timestamp, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "Timestamp")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Sequencer) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 6
	err = en.Append(0x96)
	if err != nil {
		return
	}
	err = z.TxBase.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "TxBase")
		return
	}
	if z.Issuer == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Issuer.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
	}
	err = z.BlsJointSig.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	err = z.BlsJointPubKey.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	err = z.StateRoot.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "StateRoot")
		return
	}
	err = en.WriteInt64(z.Timestamp)
	if err != nil {
		err = msgp.WrapError(err, "Timestamp")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Sequencer) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 6
	o = append(o, 0x96)
	o, err = z.TxBase.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "TxBase")
		return
	}
	if z.Issuer == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Issuer.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
	}
	o, err = z.BlsJointSig.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	o, err = z.BlsJointPubKey.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	o, err = z.StateRoot.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "StateRoot")
		return
	}
	o = msgp.AppendInt64(o, z.Timestamp)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Sequencer) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 6 {
		err = msgp.ArrayError{Wanted: 6, Got: zb0001}
		return
	}
	bts, err = z.TxBase.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "TxBase")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.Issuer = nil
	} else {
		if z.Issuer == nil {
			z.Issuer = new(common.Address)
		}
		bts, err = z.Issuer.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
	}
	bts, err = z.BlsJointSig.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	bts, err = z.BlsJointPubKey.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	bts, err = z.StateRoot.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "StateRoot")
		return
	}
	z.Timestamp, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Timestamp")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Sequencer) Msgsize() (s int) {
	s = 1 + z.TxBase.Msgsize()
	if z.Issuer == nil {
		s += msgp.NilSize
	} else {
		s += z.Issuer.Msgsize()
	}
	s += z.BlsJointSig.Msgsize() + z.BlsJointPubKey.Msgsize() + z.StateRoot.Msgsize() + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SequencerJson) DecodeMsg(dc *msgp.Reader) (err error) {
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
	err = z.TxBaseJson.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "TxBaseJson")
		return
	}
	if dc.IsNil() {
		err = dc.ReadNil()
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
		z.Issuer = nil
	} else {
		if z.Issuer == nil {
			z.Issuer = new(common.Address)
		}
		err = z.Issuer.DecodeMsg(dc)
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
	}
	err = z.BlsJointSig.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	err = z.BlsJointPubKey.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	z.Timestamp, err = dc.ReadInt64()
	if err != nil {
		err = msgp.WrapError(err, "Timestamp")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *SequencerJson) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 5
	err = en.Append(0x95)
	if err != nil {
		return
	}
	err = z.TxBaseJson.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "TxBaseJson")
		return
	}
	if z.Issuer == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Issuer.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
	}
	err = z.BlsJointSig.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	err = z.BlsJointPubKey.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	err = en.WriteInt64(z.Timestamp)
	if err != nil {
		err = msgp.WrapError(err, "Timestamp")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SequencerJson) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 5
	o = append(o, 0x95)
	o, err = z.TxBaseJson.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "TxBaseJson")
		return
	}
	if z.Issuer == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Issuer.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
	}
	o, err = z.BlsJointSig.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	o, err = z.BlsJointPubKey.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	o = msgp.AppendInt64(o, z.Timestamp)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SequencerJson) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
	bts, err = z.TxBaseJson.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "TxBaseJson")
		return
	}
	if msgp.IsNil(bts) {
		bts, err = msgp.ReadNilBytes(bts)
		if err != nil {
			return
		}
		z.Issuer = nil
	} else {
		if z.Issuer == nil {
			z.Issuer = new(common.Address)
		}
		bts, err = z.Issuer.UnmarshalMsg(bts)
		if err != nil {
			err = msgp.WrapError(err, "Issuer")
			return
		}
	}
	bts, err = z.BlsJointSig.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	bts, err = z.BlsJointPubKey.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	z.Timestamp, bts, err = msgp.ReadInt64Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Timestamp")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *SequencerJson) Msgsize() (s int) {
	s = 1 + z.TxBaseJson.Msgsize()
	if z.Issuer == nil {
		s += msgp.NilSize
	} else {
		s += z.Issuer.Msgsize()
	}
	s += z.BlsJointSig.Msgsize() + z.BlsJointPubKey.Msgsize() + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *SequencerMsg) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "Type":
			z.Type, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "Type")
				return
			}
		case "Hash":
			z.Hash, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Hash")
				return
			}
		case "Parents":
			var zb0002 uint32
			zb0002, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "Parents")
				return
			}
			if cap(z.Parents) >= int(zb0002) {
				z.Parents = (z.Parents)[:zb0002]
			} else {
				z.Parents = make([]string, zb0002)
			}
			for za0001 := range z.Parents {
				z.Parents[za0001], err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "Parents", za0001)
					return
				}
			}
		case "Issuer":
			z.Issuer, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Issuer")
				return
			}
		case "Nonce":
			z.Nonce, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "Nonce")
				return
			}
		case "Height":
			z.Height, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "Height")
				return
			}
		case "Weight":
			z.Weight, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "Weight")
				return
			}
		case "BlsJointSig":
			z.BlsJointSig, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "BlsJointSig")
				return
			}
		case "BlsJointPubKey":
			z.BlsJointPubKey, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "BlsJointPubKey")
				return
			}
		case "Timestamp":
			z.Timestamp, err = dc.ReadInt64()
			if err != nil {
				err = msgp.WrapError(err, "Timestamp")
				return
			}
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
func (z *SequencerMsg) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 10
	// write "Type"
	err = en.Append(0x8a, 0xa4, 0x54, 0x79, 0x70, 0x65)
	if err != nil {
		return
	}
	err = en.WriteInt(z.Type)
	if err != nil {
		err = msgp.WrapError(err, "Type")
		return
	}
	// write "Hash"
	err = en.Append(0xa4, 0x48, 0x61, 0x73, 0x68)
	if err != nil {
		return
	}
	err = en.WriteString(z.Hash)
	if err != nil {
		err = msgp.WrapError(err, "Hash")
		return
	}
	// write "Parents"
	err = en.Append(0xa7, 0x50, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.Parents)))
	if err != nil {
		err = msgp.WrapError(err, "Parents")
		return
	}
	for za0001 := range z.Parents {
		err = en.WriteString(z.Parents[za0001])
		if err != nil {
			err = msgp.WrapError(err, "Parents", za0001)
			return
		}
	}
	// write "Issuer"
	err = en.Append(0xa6, 0x49, 0x73, 0x73, 0x75, 0x65, 0x72)
	if err != nil {
		return
	}
	err = en.WriteString(z.Issuer)
	if err != nil {
		err = msgp.WrapError(err, "Issuer")
		return
	}
	// write "Nonce"
	err = en.Append(0xa5, 0x4e, 0x6f, 0x6e, 0x63, 0x65)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.Nonce)
	if err != nil {
		err = msgp.WrapError(err, "Nonce")
		return
	}
	// write "Height"
	err = en.Append(0xa6, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.Height)
	if err != nil {
		err = msgp.WrapError(err, "Height")
		return
	}
	// write "Weight"
	err = en.Append(0xa6, 0x57, 0x65, 0x69, 0x67, 0x68, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.Weight)
	if err != nil {
		err = msgp.WrapError(err, "Weight")
		return
	}
	// write "BlsJointSig"
	err = en.Append(0xab, 0x42, 0x6c, 0x73, 0x4a, 0x6f, 0x69, 0x6e, 0x74, 0x53, 0x69, 0x67)
	if err != nil {
		return
	}
	err = en.WriteString(z.BlsJointSig)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointSig")
		return
	}
	// write "BlsJointPubKey"
	err = en.Append(0xae, 0x42, 0x6c, 0x73, 0x4a, 0x6f, 0x69, 0x6e, 0x74, 0x50, 0x75, 0x62, 0x4b, 0x65, 0x79)
	if err != nil {
		return
	}
	err = en.WriteString(z.BlsJointPubKey)
	if err != nil {
		err = msgp.WrapError(err, "BlsJointPubKey")
		return
	}
	// write "Timestamp"
	err = en.Append(0xa9, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70)
	if err != nil {
		return
	}
	err = en.WriteInt64(z.Timestamp)
	if err != nil {
		err = msgp.WrapError(err, "Timestamp")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SequencerMsg) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 10
	// string "Type"
	o = append(o, 0x8a, 0xa4, 0x54, 0x79, 0x70, 0x65)
	o = msgp.AppendInt(o, z.Type)
	// string "Hash"
	o = append(o, 0xa4, 0x48, 0x61, 0x73, 0x68)
	o = msgp.AppendString(o, z.Hash)
	// string "Parents"
	o = append(o, 0xa7, 0x50, 0x61, 0x72, 0x65, 0x6e, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Parents)))
	for za0001 := range z.Parents {
		o = msgp.AppendString(o, z.Parents[za0001])
	}
	// string "Issuer"
	o = append(o, 0xa6, 0x49, 0x73, 0x73, 0x75, 0x65, 0x72)
	o = msgp.AppendString(o, z.Issuer)
	// string "Nonce"
	o = append(o, 0xa5, 0x4e, 0x6f, 0x6e, 0x63, 0x65)
	o = msgp.AppendUint64(o, z.Nonce)
	// string "Height"
	o = append(o, 0xa6, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74)
	o = msgp.AppendUint64(o, z.Height)
	// string "Weight"
	o = append(o, 0xa6, 0x57, 0x65, 0x69, 0x67, 0x68, 0x74)
	o = msgp.AppendUint64(o, z.Weight)
	// string "BlsJointSig"
	o = append(o, 0xab, 0x42, 0x6c, 0x73, 0x4a, 0x6f, 0x69, 0x6e, 0x74, 0x53, 0x69, 0x67)
	o = msgp.AppendString(o, z.BlsJointSig)
	// string "BlsJointPubKey"
	o = append(o, 0xae, 0x42, 0x6c, 0x73, 0x4a, 0x6f, 0x69, 0x6e, 0x74, 0x50, 0x75, 0x62, 0x4b, 0x65, 0x79)
	o = msgp.AppendString(o, z.BlsJointPubKey)
	// string "Timestamp"
	o = append(o, 0xa9, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70)
	o = msgp.AppendInt64(o, z.Timestamp)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SequencerMsg) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Type":
			z.Type, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Type")
				return
			}
		case "Hash":
			z.Hash, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Hash")
				return
			}
		case "Parents":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Parents")
				return
			}
			if cap(z.Parents) >= int(zb0002) {
				z.Parents = (z.Parents)[:zb0002]
			} else {
				z.Parents = make([]string, zb0002)
			}
			for za0001 := range z.Parents {
				z.Parents[za0001], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Parents", za0001)
					return
				}
			}
		case "Issuer":
			z.Issuer, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Issuer")
				return
			}
		case "Nonce":
			z.Nonce, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Nonce")
				return
			}
		case "Height":
			z.Height, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Height")
				return
			}
		case "Weight":
			z.Weight, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Weight")
				return
			}
		case "BlsJointSig":
			z.BlsJointSig, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BlsJointSig")
				return
			}
		case "BlsJointPubKey":
			z.BlsJointPubKey, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BlsJointPubKey")
				return
			}
		case "Timestamp":
			z.Timestamp, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Timestamp")
				return
			}
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
func (z *SequencerMsg) Msgsize() (s int) {
	s = 1 + 5 + msgp.IntSize + 5 + msgp.StringPrefixSize + len(z.Hash) + 8 + msgp.ArrayHeaderSize
	for za0001 := range z.Parents {
		s += msgp.StringPrefixSize + len(z.Parents[za0001])
	}
	s += 7 + msgp.StringPrefixSize + len(z.Issuer) + 6 + msgp.Uint64Size + 7 + msgp.Uint64Size + 7 + msgp.Uint64Size + 12 + msgp.StringPrefixSize + len(z.BlsJointSig) + 15 + msgp.StringPrefixSize + len(z.BlsJointPubKey) + 10 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Sequencers) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(Sequencers, zb0002)
	}
	for zb0001 := range *z {
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				err = msgp.WrapError(err, zb0001)
				return
			}
			(*z)[zb0001] = nil
		} else {
			if (*z)[zb0001] == nil {
				(*z)[zb0001] = new(Sequencer)
			}
			err = (*z)[zb0001].DecodeMsg(dc)
			if err != nil {
				err = msgp.WrapError(err, zb0001)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Sequencers) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteArrayHeader(uint32(len(z)))
	if err != nil {
		err = msgp.WrapError(err)
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
				err = msgp.WrapError(err, zb0003)
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Sequencers) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendArrayHeader(o, uint32(len(z)))
	for zb0003 := range z {
		if z[zb0003] == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = z[zb0003].MarshalMsg(o)
			if err != nil {
				err = msgp.WrapError(err, zb0003)
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Sequencers) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if cap((*z)) >= int(zb0002) {
		(*z) = (*z)[:zb0002]
	} else {
		(*z) = make(Sequencers, zb0002)
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
				(*z)[zb0001] = new(Sequencer)
			}
			bts, err = (*z)[zb0001].UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, zb0001)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Sequencers) Msgsize() (s int) {
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
