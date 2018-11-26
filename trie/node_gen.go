package trie

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"fmt"
	"strconv"

	"github.com/tinylib/msgp/msgp"
)

var nodeTypeSize = 1

func nodetypebyte(t int) byte {
	return ([]byte(strconv.Itoa(t)))[0]
}

// DecodeMsg implements msgp.Decodable
func (z *FullNode) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	var zb0002 uint32
	zb0002, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0002 != uint32(17) {
		err = msgp.ArrayError{Wanted: uint32(17), Got: zb0002}
		return
	}
	for za0001 := range z.Children {
		err = z.Children[za0001].DecodeMsg(dc)
		if err != nil {
			return
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *FullNode) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(17))
	if err != nil {
		return
	}
	for za0001 := range z.Children {
		err = z.Children[za0001].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *FullNode) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendArrayHeader(o, uint32(17))
	for za0001 := range z.Children {
		o, err = z.Children[za0001].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *FullNode) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0002 != uint32(17) {
		err = msgp.ArrayError{Wanted: uint32(17), Got: zb0002}
		return
	}
	for za0001 := range z.Children {
		bts, err = z.Children[za0001].UnmarshalMsg(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *FullNode) Msgsize() (s int) {
	s = 1 + msgp.ArrayHeaderSize
	for za0001 := range z.Children {
		s += nodeTypeSize
		s += z.Children[za0001].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z HashNode) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var zb0001 []byte
		zb0001, err = dc.ReadBytes([]byte((z)))
		if err != nil {
			return
		}
		(z) = HashNode(zb0001)
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z HashNode) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteBytes([]byte(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z HashNode) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendBytes(o, []byte(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z HashNode) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zb0001 []byte
		zb0001, bts, err = msgp.ReadBytesBytes(bts, []byte((z)))
		if err != nil {
			return
		}
		(z) = HashNode(zb0001)
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z HashNode) Msgsize() (s int) {
	s = msgp.BytesPrefixSize + len([]byte(z))
	return
}

// DecodeMsg implements msgp.Decodable
func (z *ShortNode) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.Key, err = dc.ReadBytes(z.Key)
	if err != nil {
		return
	}
	err = z.Val.DecodeMsg(dc)
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *ShortNode) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 2
	err = en.Append(0x92)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Key)
	if err != nil {
		return
	}
	err = z.Val.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *ShortNode) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendBytes(o, z.Key)

	switch z.Val.(type) {
	case nil:
		o = append(o, nodetypebyte(nilnode))
		return
	case *FullNode:
		o = append(o, nodetypebyte(fullnode))
		break
	case *ShortNode:
		o = append(o, nodetypebyte(shortnode))
		break
	case *HashNode:
		o = append(o, nodetypebyte(hashnode))
		break
	case *ValueNode:
		o = append(o, nodetypebyte(valuenode))
		break
	}
	o, err = z.Val.MarshalMsg(o)
	if err != nil {
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ShortNode) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.Key, bts, err = msgp.ReadBytesBytes(bts, z.Key)
	if err != nil {
		return
	}
	// initiate z.Val
	bts, err = z.GenNode(bts)
	if err != nil {
		return
	}
	if z.Val == nil {
		return
	}
	bts, err = z.Val.UnmarshalMsg(bts)
	if err != nil {
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *ShortNode) Msgsize() (s int) {
	s = 1 + msgp.BytesPrefixSize + len(z.Key) + nodeTypeSize
	if z.Val != nil {
		s += z.Val.Msgsize()
	}
	return
}

// GenNode checks the type of z.Val and initiate an object entity for it.
func (z *ShortNode) GenNode(b []byte) (o []byte, err error) {
	t, errc := strconv.Atoi(string(b[0]))
	if errc != nil {
		err = fmt.Errorf("convert byte to int error: %v", errc)
		return
	}
	switch t {
	case nilnode:
		z.Val = nil
	case fullnode:
		z.Val = &FullNode{}
	case shortnode:
		z.Val = &ShortNode{}
	case valuenode:
		z.Val = &ValueNode{}
	case hashnode:
		z.Val = &HashNode{}
	}

	o = b[1:]
	return
}

// DecodeMsg implements msgp.Decodable
func (z ValueNode) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var zb0001 []byte
		zb0001, err = dc.ReadBytes([]byte((z)))
		if err != nil {
			return
		}
		(z) = ValueNode(zb0001)
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z ValueNode) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteBytes([]byte(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z ValueNode) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendBytes(o, []byte(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z ValueNode) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zb0001 []byte
		zb0001, bts, err = msgp.ReadBytesBytes(bts, []byte((z)))
		if err != nil {
			return
		}
		(z) = ValueNode(zb0001)
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z ValueNode) Msgsize() (s int) {
	s = msgp.BytesPrefixSize + len([]byte(z))
	return
}
