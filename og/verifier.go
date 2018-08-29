package og

import (
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/crypto"
)

type Verifier struct {
	signer     crypto.Signer
	cryptoType crypto.CryptoType
}

func NewVerifier(signer crypto.Signer) *Verifier {
	return &Verifier{signer: signer, cryptoType: signer.GetCryptoType()}
}

func (v *Verifier) VerifyHash(t *types.Tx) bool {
	return t.Hash() == t.TxBase.Hash
}

func (v *Verifier) VerifySignature(t *types.Tx) bool {
	return v.signer.Verify(crypto.PublicKey{Type:  v.cryptoType, Bytes: t.PublicKey},
		crypto.Signature{Type:  v.cryptoType, Bytes: t.Signature},
		t.TxBase.Hash.Bytes[:])
}
