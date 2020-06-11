// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// +build !noncgo

package ogcrypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"github.com/annchain/OG/arefactor/common/math"
	"github.com/annchain/OG/arefactor/ogcrypto/ecies"
	"github.com/annchain/OG/arefactor/ogcrypto/secp256k1"
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
	ecdsabtcec "github.com/btcsuite/btcd/btcec"
	"github.com/sirupsen/logrus"
)

type SignerSecp256k1 struct {
}

func (s *SignerSecp256k1) GetCryptoType() ogcrypto_interface.CryptoType {
	return ogcrypto_interface.CryptoTypeSecp256k1
}

func (s *SignerSecp256k1) Sign(privKey ogcrypto_interface.PrivateKey, msg []byte) ogcrypto_interface.Signature {
	priv, err := ToECDSA(privKey.KeyBytes)
	if err != nil {
		fmt.Println(fmt.Sprintf("ToECDSA error: %v. priv bytes: %x", err, privKey.KeyBytes))
	}
	hash := Sha256(msg)
	if len(hash) != 32 {
		logrus.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
		return ogcrypto_interface.Signature{}
	}
	seckey := math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
	defer zeroBytes(seckey)
	sig, _ := secp256k1.Sign(hash, seckey)

	return SignatureFromBytes(ogcrypto_interface.CryptoTypeSecp256k1, sig)
}

func (s *SignerSecp256k1) PubKey(privKey ogcrypto_interface.PrivateKey) ogcrypto_interface.PublicKey {
	_, ecdsapub := ecdsabtcec.PrivKeyFromBytes(ecdsabtcec.S256(), privKey.KeyBytes)
	pub := FromECDSAPub((*ecdsa.PublicKey)(ecdsapub))
	return PublicKeyFromBytes(ogcrypto_interface.CryptoTypeSecp256k1, pub[:])
}

func (s *SignerSecp256k1) PublicKeyFromBytes(b []byte) ogcrypto_interface.PublicKey {
	return PublicKeyFromBytes(s.GetCryptoType(), b)
}

func (s *SignerSecp256k1) Verify(pubKey ogcrypto_interface.PublicKey, signature ogcrypto_interface.Signature, msg []byte) bool {
	signature = s.DealRecoverID(signature)
	sig := signature.SignatureBytes

	//fmt.Println(fmt.Sprintf("pubkey bytes: %x", pubKey.KeyBytes))
	//fmt.Println(fmt.Sprintf("msg: %x", msg))
	//fmt.Println(fmt.Sprintf("sig: %x", sig))

	return secp256k1.VerifySignature(pubKey.KeyBytes, Sha256(msg), sig)
}

func (s *SignerSecp256k1) RandomKeyPair() (publicKey ogcrypto_interface.PublicKey, privateKey ogcrypto_interface.PrivateKey) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], CRandBytes(32))

	privateKey = PrivateKeyFromBytes(ogcrypto_interface.CryptoTypeSecp256k1, privKeyBytes[:])
	publicKey = s.PubKey(privateKey)
	return
}

func (s *SignerSecp256k1) Encrypt(p ogcrypto_interface.PublicKey, m []byte) (ct []byte, err error) {
	pub, err := UnmarshalPubkey(p.KeyBytes)
	if err != nil {
		panic(err)
	}
	eciesPub := ecies.ImportECDSAPublic(pub)
	return ecies.Encrypt(rand.Reader, eciesPub, m, nil, nil)
}

func (s *SignerSecp256k1) Decrypt(p ogcrypto_interface.PrivateKey, ct []byte) (m []byte, err error) {
	prive, err := ToECDSA(p.KeyBytes)
	ecisesPriv := ecies.ImportECDSA(prive)
	return ecisesPriv.Decrypt(ct, nil, nil)
}

const sigLength int = 64

func (s *SignerSecp256k1) DealRecoverID(sig ogcrypto_interface.Signature) ogcrypto_interface.Signature {
	l := len(sig.SignatureBytes)
	if l == sigLength+1 {
		sig.SignatureBytes = sig.SignatureBytes[:l-1]
	}
	return sig
}

// Ecrecover returns the uncompressed public key that created the given signature.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(hash, sig)
}

//// SigToPub returns the public key that created the given signature.
//func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
//	s, err := Ecrecover(hash, sig)
//	if err != nil {
//		return nil, err
//	}
//
//	x, y := elliptic.Unmarshal(S256(), s)
//	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
//}

func (s *SignerSecp256k1) CanRecoverPubFromSig() bool {
	//return true
	return false
}
