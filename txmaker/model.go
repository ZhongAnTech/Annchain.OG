package txmaker

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
)

type TxWithSealBuildRequest struct {
	From    common.Address
	To      common.Address
	Value   *math.BigInt
	Data    []byte
	Nonce   uint64
	Pubkey  crypto.PublicKey
	Sig     crypto.Signature
	TokenId int32
}

type UnsignedTxBuildRequest struct {
	From         common.Address
	To           common.Address
	Value        *math.BigInt
	AccountNonce uint64
	TokenId      int32
}

type ActionTxBuildRequest struct {
	UnsignedTxBuildRequest
	Action    byte
	EnableSpo bool
	TokenName string
	Pubkey    crypto.PublicKey
	Sig       crypto.Signature
}

type SignedTxBuildRequest struct {
	UnsignedTxBuildRequest
	PrivateKey crypto.PrivateKey
}

type UnsignedSequencerBuildRequest struct {
	Issuer       common.Address
	Height       uint64
	AccountNonce uint64
}

type SignedSequencerBuildRequest struct {
	UnsignedSequencerBuildRequest
	PrivateKey crypto.PrivateKey
}
