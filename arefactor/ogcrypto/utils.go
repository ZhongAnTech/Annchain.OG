package ogcrypto

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ripemd160"
	"math/big"
)

func PrivateKeyFromBytes(typev ogcrypto_interface.CryptoType, bytes []byte) ogcrypto_interface.PrivateKey {
	return ogcrypto_interface.PrivateKey{Type: typev, KeyBytes: bytes}
}
func PublicKeyFromBytes(typev ogcrypto_interface.CryptoType, bytes []byte) ogcrypto_interface.PublicKey {
	return ogcrypto_interface.PublicKey{Type: typev, KeyBytes: bytes}
}

func SignatureValues(sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		return r, s, v, fmt.Errorf("wrong size for signature: got %d, want 65", len(sig))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

func PublicKeyFromSignature(sighash types.Hash, signature *ogcrypto_interface.Signature) (pubKey ogcrypto_interface.PublicKey, err error) {
	// only some signature types can be used to recover pubkey
	R, S, Vb, err := SignatureValues(signature.SignatureBytes)
	if err != nil {
		logrus.WithError(err).Debug("verify sigBytes failed")
		return
	}
	if Vb.BitLen() > 8 {
		err = errors.New("v len error")
		logrus.WithError(err).Debug("v len error")
		return
	}
	V := byte(Vb.Uint64() - 27)
	if !ValidateSignatureValues(V, R, S, false) {
		err = errors.New("vrs error")
		logrus.WithError(err).Debug("validate signature error")
		return
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sigBytes := make([]byte, 65)
	copy(sigBytes[32-len(r):32], r)
	copy(sigBytes[64-len(s):64], s)
	sigBytes[64] = V
	// recover the public key from the signature
	//pub, err := Ecrecover(sighash.Bytes[:], sigBytes)
	pub, err := Ecrecover(sighash, sigBytes)
	if err != nil {
		logrus.WithError(err).Debug("sigBytes verify failed")
	}
	if len(pub) == 0 || pub[0] != 4 {
		err := errors.New("invalid public key")
		logrus.WithError(err).Debug("verify sigBytes failed")
	}
	return PublicKeyFromRawBytes(pub), nil
}

func SignatureFromBytes(typev ogcrypto_interface.CryptoType, bytes []byte) ogcrypto_interface.Signature {
	return ogcrypto_interface.Signature{Type: typev, SignatureBytes: bytes}
}

func PrivateKeyFromRawBytes(bytes []byte) ogcrypto_interface.PrivateKey {
	cryptoType := ogcrypto_interface.CryptoTypeSecp256k1
	if len(bytes) == 33 {
		cryptoType = ogcrypto_interface.CryptoType(bytes[0])
		bytes = bytes[1:]
	}
	return PrivateKeyFromBytes(cryptoType, bytes)
}
func PublicKeyFromRawBytes(bytes []byte) ogcrypto_interface.PublicKey {
	cryptoType := ogcrypto_interface.CryptoTypeSecp256k1
	if len(bytes) == 33 {
		cryptoType = ogcrypto_interface.CryptoType(bytes[0])
		bytes = bytes[1:]
	}
	return PublicKeyFromBytes(cryptoType, bytes)
}
func SignatureFromRawBytes(bytes []byte) ogcrypto_interface.Signature {
	cryptoType := ogcrypto_interface.CryptoTypeSecp256k1
	if len(bytes) == 33 {
		cryptoType = ogcrypto_interface.CryptoType(bytes[0])
		bytes = bytes[1:]
	}
	return SignatureFromBytes(cryptoType, bytes)
}

func PrivateKeyFromString(value string) (priv ogcrypto_interface.PrivateKey, err error) {
	bytes, err := hexutil.FromHex(value)
	if err != nil {
		return
	}
	priv = PrivateKeyFromRawBytes(bytes)
	return priv, err
}

func PublicKeyFromString(value string) (pub ogcrypto_interface.PublicKey, err error) {
	bytes, err := hexutil.FromHex(value)
	if err != nil {
		return
	}
	pub = PublicKeyFromRawBytes(bytes)
	return pub, err
}

func PublicKeyFromStringWithCryptoType(ct, pkstr string) (pub ogcrypto_interface.PublicKey, err error) {
	cryptoType, ok := ogcrypto_interface.CryptoNameMap[ct]
	if !ok {
		err = fmt.Errorf("unknown ogcrypto type: %s", ct)
		return
	}
	pk, err := hexutil.FromHex(pkstr)
	if err != nil {
		return
	}
	pub = ogcrypto_interface.PublicKey{
		Type:     cryptoType,
		KeyBytes: pk,
	}
	return
}

func Sha256(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

func Ripemd160(bytes []byte) []byte {
	hasher := ripemd160.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}
