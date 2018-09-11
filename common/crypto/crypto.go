package crypto

import (
	"github.com/annchain/OG/common/hexutil"
)

type CryptoType int8

const (
	CryptoTypeEd25519   CryptoType = iota
	CryptoTypeSecp256k1
)

type PrivateKey struct {
	Type  CryptoType
	Bytes []byte
}

type PublicKey struct {
	Type  CryptoType
	Bytes []byte
}

type Signature struct {
	Type  CryptoType
	Bytes []byte
}

func PrivateKeyFromBytes(typev CryptoType, bytes []byte) PrivateKey {
	return PrivateKey{Type: typev, Bytes: bytes}
}
func PublicKeyFromBytes(typev CryptoType, bytes []byte) PublicKey {
	return PublicKey{Type: typev, Bytes: bytes}
}
func SignatureFromBytes(typev CryptoType, bytes []byte) Signature {
	return Signature{Type: typev, Bytes: bytes}
}

func PrivateKeyFromString(value string) (priv PrivateKey, err error) {
	bytes, err := hexutil.Decode(value)
	if err != nil {
		return
	}
	priv = PrivateKey{
		Type:  CryptoType(bytes[0]),
		Bytes: bytes[1:],
	}
	return
}

func (k *PrivateKey) PrivateKeyToString() string {
	var bytes []byte
	bytes = append(bytes, byte(k.Type))
	bytes = append(bytes, k.Bytes...)
	return hexutil.Encode(bytes)
}
