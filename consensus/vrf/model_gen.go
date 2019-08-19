package vrf

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *VrfInfo) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0001 != 4 {
		err = msgp.ArrayError{Wanted: 4, Got: zb0001}
		return
	}
	z.Message, err = dc.ReadBytes(z.Message)
	if err != nil {
		return
	}
	z.Proof, err = dc.ReadBytes(z.Proof)
	if err != nil {
		return
	}
	z.PublicKey, err = dc.ReadBytes(z.PublicKey)
	if err != nil {
		return
	}
	z.Vrf, err = dc.ReadBytes(z.Vrf)
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *VrfInfo) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 4
	err = en.Append(0x94)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Message)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Proof)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.PublicKey)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Vrf)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *VrfInfo) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 4
	o = append(o, 0x94)
	o = msgp.AppendBytes(o, z.Message)
	o = msgp.AppendBytes(o, z.Proof)
	o = msgp.AppendBytes(o, z.PublicKey)
	o = msgp.AppendBytes(o, z.Vrf)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *VrfInfo) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0001 != 4 {
		err = msgp.ArrayError{Wanted: 4, Got: zb0001}
		return
	}
	z.Message, bts, err = msgp.ReadBytesBytes(bts, z.Message)
	if err != nil {
		return
	}
	z.Proof, bts, err = msgp.ReadBytesBytes(bts, z.Proof)
	if err != nil {
		return
	}
	z.PublicKey, bts, err = msgp.ReadBytesBytes(bts, z.PublicKey)
	if err != nil {
		return
	}
	z.Vrf, bts, err = msgp.ReadBytesBytes(bts, z.Vrf)
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *VrfInfo) Msgsize() (s int) {
	s = 1 + msgp.BytesPrefixSize + len(z.Message) + msgp.BytesPrefixSize + len(z.Proof) + msgp.BytesPrefixSize + len(z.PublicKey) + msgp.BytesPrefixSize + len(z.Vrf)
	return
}
