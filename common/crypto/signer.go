package crypto

import "github.com/annchain/OG/types"

type Signer interface {
	GetCryptoType() CryptoType
	Sign(privKey PrivateKey, msg []byte) Signature
	PubKey(privKey PrivateKey) PublicKey
	Verify(pubKey PublicKey, signature Signature, msg []byte) bool
	RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey, err error)
	Address(pubKey PublicKey) types.Address
}
