package ogcrypto_interface

type ISigner interface {
	GetCryptoType() CryptoType
	Sign(privKey PrivateKey, msg []byte) Signature
	PubKey(privKey PrivateKey) PublicKey
	Verify(pubKey PublicKey, signature Signature, msg []byte) bool
	RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey)
	Encrypt(publicKey PublicKey, m []byte) (ct []byte, err error)
	Decrypt(p PrivateKey, ct []byte) (m []byte, err error)
	PublicKeyFromBytes(b []byte) PublicKey
	CanRecoverPubFromSig() bool
}
