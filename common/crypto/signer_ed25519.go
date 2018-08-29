package crypto

import (
	"golang.org/x/crypto/ed25519"
	"github.com/annchain/OG/types"
	"bytes"
	"golang.org/x/crypto/ripemd160"
)

type SignerEd25519 struct {
}

func (s *SignerEd25519) GetCryptoType() CryptoType {
	return CryptoTypeEd25519
}

func (s *SignerEd25519) Sign(privKey PrivateKey, msg []byte) Signature {
	signatureBytes := ed25519.Sign(privKey.Bytes, msg)
	return SignatureFromBytes(CryptoTypeEd25519, signatureBytes)
}

func (s *SignerEd25519) PubKey(privKey PrivateKey) PublicKey {
	pubkey := ed25519.PrivateKey(privKey.Bytes).Public()
	return PublicKeyFromBytes(CryptoTypeEd25519, []byte(pubkey.(ed25519.PublicKey)))
}

func (s *SignerEd25519) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	return ed25519.Verify(pubKey.Bytes, msg, signature.Bytes)
}

func (s *SignerEd25519) RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey, err error) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		return
	}
	publicKey = PublicKeyFromBytes(CryptoTypeEd25519, public)
	privateKey = PrivateKeyFromBytes(CryptoTypeEd25519, private)
	return
}

// Address calculate the address from the pubkey
func (s *SignerEd25519) Address(pubKey PublicKey) types.Address {
	var w bytes.Buffer
	w.Write([]byte{byte(pubKey.Type)})
	w.Write(pubKey.Bytes)
	hasher := ripemd160.New()
	hasher.Write(w.Bytes())
	result := hasher.Sum(nil)
	return types.BytesToAddress(result)
}
