package crypto

import (
	"bytes"
	"fmt"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ripemd160"
	"strconv"
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

func (s *SignerEd25519) AddressFromPubKeyBytes(pubKey []byte) types.Address {
	return s.Address(PublicKeyFromBytes(CryptoTypeEd25519, pubKey))
}

func (s *SignerEd25519) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	//validate to prevent panic
	if l := len(pubKey.Bytes); l != ed25519.PublicKeySize {
		err := fmt.Errorf("ed25519: bad public key length: " + strconv.Itoa(l))
		logrus.WithError(err).Warn("verify fail")
		return false
	}
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

func (s*SignerEd25519)Encrypt(publicKey PublicKey, m []byte) (ct []byte, err error){
	panic("not supported")
	return nil ,nil
}

func (s*SignerEd25519) Decrypt(p PrivateKey, ct []byte) ( m []byte, err error) {
	panic("not supported")
	return nil ,nil
}
