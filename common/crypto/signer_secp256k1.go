package crypto

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"

	"github.com/annchain/OG/types"
	ecdsabtcec "github.com/btcsuite/btcd/btcec"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ripemd160"
)

type SignerSecp256k1 struct {
}

func (s *SignerSecp256k1) GetCryptoType() CryptoType {
	return CryptoTypeSecp256k1
}

func (s *SignerSecp256k1) Sign(privKey PrivateKey, msg []byte) Signature {
	priv, _ := ethcrypto.HexToECDSA(fmt.Sprintf("%x", privKey.Bytes))

	hash := Sha256(msg)
	sig, _ := ethcrypto.Sign(hash, priv)

	return SignatureFromBytes(CryptoTypeSecp256k1, sig)
}

func (s *SignerSecp256k1) PubKey(privKey PrivateKey) PublicKey {
	_, ecdsapub := ecdsabtcec.PrivKeyFromBytes(ecdsabtcec.S256(), privKey.Bytes)
	pub := ethcrypto.FromECDSAPub((*ecdsa.PublicKey)(ecdsapub))

	return PublicKeyFromBytes(CryptoTypeSecp256k1, pub[:])
}

func (s *SignerSecp256k1) AddressFromPubKeyBytes(pubKey []byte) types.Address {
	return s.Address(PublicKeyFromBytes(CryptoTypeSecp256k1, pubKey))
}

func (s *SignerSecp256k1) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	sig := (signature.Bytes)[:len(signature.Bytes)-1]
	return ethcrypto.VerifySignature(pubKey.Bytes, Sha256(msg), sig)
}

func (s *SignerSecp256k1) RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey, err error) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], CRandBytes(32))

	privateKey = PrivateKeyFromBytes(CryptoTypeSecp256k1, privKeyBytes[:])
	publicKey = s.PubKey(privateKey)
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
