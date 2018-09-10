package crypto

import (
	"crypto/sha256"
	"github.com/annchain/OG/types"
	secp256k1 "github.com/btcsuite/btcd/btcec"
	"golang.org/x/crypto/ripemd160"
)

type SignerSecp256k1 struct {
}

func (s *SignerSecp256k1) GetCryptoType() CryptoType {
	return CryptoTypeSecp256k1
}

func (s *SignerSecp256k1) Sign(privKey PrivateKey, msg []byte) Signature {
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKey.Bytes)
	sig, err := priv.Sign(Sha256(msg))
	if err != nil {
		panic(err)
	}

	return SignatureFromBytes(CryptoTypeSecp256k1, sig.Serialize())
}

func (s *SignerSecp256k1) PubKey(privKey PrivateKey) PublicKey {
	_, pub__ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKey.Bytes)
	pub := [64]byte{}
	copy(pub[:], pub__.SerializeUncompressed()[1:])
	return PublicKeyFromBytes(CryptoTypeSecp256k1, pub[:])
}

func (s *SignerSecp256k1) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	pub65Bytes := append([]byte{0x04}, pubKey.Bytes...)
	pub__, err := secp256k1.ParsePubKey(pub65Bytes, secp256k1.S256())
	if err != nil {
		panic(err)
		return false
	}
	sig__, err := secp256k1.ParseDERSignature(signature.Bytes, secp256k1.S256())
	if err != nil {
		panic(err)
		return false
	}
	return sig__.Verify(Sha256(msg), pub__)
}

func (s *SignerSecp256k1) RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey, err error) {
	privKeyBytes := [32]byte{}
	pubKeyBytes := [64]byte{}
	copy(privKeyBytes[:], CRandBytes(32))
	priv, pub := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	copy (pubKeyBytes[:], pub.SerializeUncompressed()[1:])
	privateKey = PrivateKeyFromBytes(CryptoTypeSecp256k1, privKeyBytes[:])
	publicKey= PublicKeyFromBytes(CryptoTypeSecp256k1,pubKeyBytes[:])
	return
}

// Address calculate the address from the pubkey
func (s *SignerSecp256k1) Address(pubKey PublicKey) types.Address {
	hasherSHA256 := sha256.New()
	hasherSHA256.Write([]byte{byte(pubKey.Type)})
	hasherSHA256.Write(pubKey.Bytes) // does not error
	sha := hasherSHA256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error
	result := hasherRIPEMD160.Sum(nil)
	return types.BytesToAddress(result)
}
