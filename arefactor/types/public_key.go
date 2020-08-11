package types

import (
	"encoding/json"
	"github.com/annchain/commongo/hexutil"
	"github.com/tinylib/msgp/msgp"
)

////no go:generate msgp

type PublicKey []byte

// DecodeMsg implements msgp.Decodable
func (z *PublicKey) DecodeMsg(dc *msgp.Reader) (err error) {
	if CanRecoverPubFromSig {
		return nil
	}
	{
		var zb0001 []byte
		zb0001, err = dc.ReadBytes([]byte((*z)))
		if err != nil {
			return
		}
		(*z) = PublicKey(zb0001)
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z PublicKey) EncodeMsg(en *msgp.Writer) (err error) {
	if CanRecoverPubFromSig {
		return nil
	}
	err = en.WriteBytes([]byte(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z PublicKey) MarshalMsg(b []byte) (o []byte, err error) {
	if CanRecoverPubFromSig {
		return nil, nil
	}
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendBytes(o, []byte(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *PublicKey) UnmarshalMsg(bts []byte) (o []byte, err error) {
	if CanRecoverPubFromSig {
		return bts, nil
	}
	{
		var zb0001 []byte
		zb0001, bts, err = msgp.ReadBytesBytes(bts, []byte((*z)))
		if err != nil {
			return
		}
		(*z) = PublicKey(zb0001)
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z PublicKey) Msgsize() (s int) {
	if CanRecoverPubFromSig {
		return 0
	}
	s = msgp.BytesPrefixSize + len([]byte(z))
	return
}

func (z PublicKey) String() string {
	return hexutil.Encode(z)
}

func (b PublicKey) MarshalJson() ([]byte, error) {
	s := b.String()
	return json.Marshal(&s)
}
