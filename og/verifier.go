package og

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
)

type Verifier struct {
	signer       crypto.Signer
	cryptoType   crypto.CryptoType
	MaxTxHash    types.Hash // The difficultiy of TxHash
	MaxMinedHash types.Hash // The difficultiy of MinedHash
}

func NewVerifier(signer crypto.Signer, maxTxHash types.Hash, maxMineHash types.Hash) *Verifier {
	return &Verifier{
		signer:       signer,
		cryptoType:   signer.GetCryptoType(),
		MaxTxHash:    maxTxHash,
		MaxMinedHash: maxMineHash,
	}
}

func (v *Verifier) VerifyHash(t types.Txi) bool {
	return (t.CalcMinedHash().Cmp(v.MaxMinedHash) < 0) &&
		t.CalcTxHash() == t.GetTxHash() &&
		(t.GetTxHash().Cmp(v.MaxTxHash) < 0)

}

func (v *Verifier) VerifySignature(t types.Txi) bool {
	base := t.GetBase()
	return v.signer.Verify(
		crypto.PublicKey{Type: v.cryptoType, Bytes: base.PublicKey},
		crypto.Signature{Type: v.cryptoType, Bytes: base.Signature},
		t.SignatureTargets())
}

func (v *Verifier) VerifySourceAddress(t types.Txi) bool {
	switch t.(type) {
	case *types.Tx:
		return t.(*types.Tx).From.Bytes == v.signer.Address(crypto.PublicKeyFromBytes(v.cryptoType, t.GetBase().PublicKey)).Bytes
	case *types.Sequencer:
		return t.(*types.Sequencer).Issuer.Bytes == v.signer.Address(crypto.PublicKeyFromBytes(v.cryptoType, t.GetBase().PublicKey)).Bytes
	default:
		return true
	}
}
