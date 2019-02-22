package crypto

import "github.com/annchain/OG/types"

type Signer interface {
	GetCryptoType() CryptoType
	Sign(privKey PrivateKey, msg []byte) Signature
	PubKey(privKey PrivateKey) PublicKey
	Verify(pubKey PublicKey, signature Signature, msg []byte) bool
	RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey, err error)
	Address(pubKey PublicKey) types.Address
	AddressFromPubKeyBytes(pubKey []byte) types.Address
	Encrypt(publicKey PublicKey, m []byte) (ct []byte, err error)
	Decrypt(p PrivateKey, ct []byte) (m []byte, err error)
}
