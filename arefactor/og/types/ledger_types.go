package types

import (
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
)

type OgTx struct {
	Hash        Hash
	ParentsHash []Hash
	MineNonce   uint64
	From        Address
	To          Address
	Value       string // bigint
	TokenId     int32
	PublicKey   []byte
	Data        []byte
	Signature   ogcrypto_interface.Signature
}

type OgSequencer struct {
}
