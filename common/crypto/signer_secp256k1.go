package crypto

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/annchain/OG/common/crypto/secp256k1"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	ecdsabtcec "github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
)

type SignerSecp256k1 struct {
}

func (s *SignerSecp256k1) GetCryptoType() CryptoType {
	return CryptoTypeSecp256k1
}

func (s *SignerSecp256k1) Sign(privKey PrivateKey, msg []byte) Signature {
	priv, _ := HexToECDSA(fmt.Sprintf("%x", privKey.Bytes))
	hash := Sha256(msg)
	if len(hash) != 32 {
		log.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
		return Signature{}
	}
	seckey := math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
	defer zeroBytes(seckey)
	sig, _ := secp256k1.Sign(hash, seckey)

	return SignatureFromBytes(CryptoTypeSecp256k1, sig)
}

func (s *SignerSecp256k1) PubKey(privKey PrivateKey) PublicKey {
	_, ecdsapub := ecdsabtcec.PrivKeyFromBytes(ecdsabtcec.S256(), privKey.Bytes)
	pub := FromECDSAPub((*ecdsa.PublicKey)(ecdsapub))

	return PublicKeyFromBytes(CryptoTypeSecp256k1, pub[:])
}

func (s *SignerSecp256k1) AddressFromPubKeyBytes(pubKey []byte) types.Address {
	return s.Address(PublicKeyFromBytes(CryptoTypeSecp256k1, pubKey))
}

func (s *SignerSecp256k1) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	sig := (signature.Bytes)[:len(signature.Bytes)-1]
	return secp256k1.VerifySignature(pubKey.Bytes, Sha256(msg), sig)
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
	return types.BytesToAddress(Keccak256((pubKey.Bytes)[1:])[12:])
}
