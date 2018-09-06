package og

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
)

type Verifier struct {
	signer     crypto.Signer
	cryptoType crypto.CryptoType
}

func NewVerifier(signer crypto.Signer) *Verifier {
	return &Verifier{signer: signer, cryptoType: signer.GetCryptoType()}
}

func (v *Verifier) VerifyHash(t types.Txi) bool {
	return t.CalcTxHash() == t.GetTxHash()
}

func (v *Verifier) VerifySignature(t types.Txi) bool {
	base := t.GetBase()
	return v.signer.Verify(crypto.PublicKey{Type: v.cryptoType, Bytes: base.PublicKey},
		crypto.Signature{Type: v.cryptoType, Bytes: base.Signature},
		base.Hash.Bytes[:])
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
