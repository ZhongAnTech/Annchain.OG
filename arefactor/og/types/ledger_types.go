package types

import (
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
)

type OgTx struct {
	Hash        og_interface.Hash
	ParentsHash []og_interface.Hash
	MineNonce   uint64
	From        og_interface.Address
	To          og_interface.Address
	Value       string // bigint
	TokenId     int32
	PublicKey   []byte
	Data        []byte
	Signature   ogcrypto_interface.Signature
}

type OgSequencer struct {
}
