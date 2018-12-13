package onode

import (
	"crypto/ecdsa"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/msg"
	"github.com/tinylib/msgp/msgp"
)

// Secp256k1 is the "secp256k1" key, which holds a public key.
type Secp256k1 ecdsa.PublicKey

func (v Secp256k1) ENRKey() string { return "secp256k1" }

// EncodeRLP implements rlp.Encoder.
func (v Secp256k1) MarshalMsg(b []byte) ([]byte, error) {
	key := crypto.CompressPubkey((*ecdsa.PublicKey)(&v))
	keyBytes := msg.Bytes(key)
	return keyBytes.MarshalMsg(b)
}

// DecodeRLP implements rlp.Decoder.
func (v *Secp256k1) UnmarshalMsg(b []byte) ([]byte, error) {
	var keyBytes msg.Bytes
	d, err := keyBytes.UnmarshalMsg(b)
	if err != nil {
		return d, err
	}
	pk, err := crypto.DecompressPubkey(keyBytes)
	if err != nil {
		return d, err
	}
	*v = (Secp256k1)(*pk)
	return d, nil
}

func (v *Secp256k1) DecodeMsg(en *msgp.Reader) (err error) {
	var keyBytes msg.Bytes
	err = keyBytes.DecodeMsg(en)
	if err != nil {
		return err
	}
	pk, err := crypto.DecompressPubkey(keyBytes)
	if err != nil {
		return err
	}
	*v = (Secp256k1)(*pk)
	return nil
}

func (v Secp256k1) EncodeMsg(en *msgp.Writer) (err error) {
	key := crypto.CompressPubkey((*ecdsa.PublicKey)(&v))
	keyBytes := msg.Bytes(key)
	return keyBytes.EncodeMsg(en)
}

func (v Secp256k1) Msgsize() int {
	key := crypto.CompressPubkey((*ecdsa.PublicKey)(&v))
	keyBytes := msg.Bytes(key)
	return keyBytes.Msgsize()
}
