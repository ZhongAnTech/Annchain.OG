package crypto

import (
	"github.com/annchain/OG/types"
	"crypto/elliptic"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"errors"
)


var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

var errInvalidPubkey = errors.New("invalid secp256k1 public key")

func Keccak256Hash(data ...[]byte) (h types.Hash) {
	return
}


func RecoverPubkey(msg []byte, sig []byte) ([]byte, error) {
	return nil,nil
}

 func Ecrecover (msg []byte,sig []byte)([]byte, error) {
	 return nil,nil
 }

func  S256()      elliptic.Curve {
 return   nil
}


func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256(), rand.Reader)
}


func Keccak256([]byte) []byte {
	return nil 
}

func Sign(data[]byte,priv *ecdsa.PrivateKey) ([]byte ,error){
	return nil,nil
}



// HexToECDSA parses a secp256k1 private key.
func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	return ToECDSA(b)
}

// ToECDSA creates a private key with the given D value.
func ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	return toECDSA(d, true)
}

// toECDSA creates a private key with the given D value. The strict parameter
// controls whether the key's length should be enforced at the curve size or
// it can also accept legacy encodings (0 prefixes).
func toECDSA(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}
