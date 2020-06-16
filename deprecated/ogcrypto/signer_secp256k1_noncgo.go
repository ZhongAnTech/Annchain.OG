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
// +build noncgo

package ogcrypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"github.com/annchain/OG/arefactor/ogcrypto"
	"github.com/annchain/OG/deprecated"
	"github.com/annchain/OG/deprecated/ogcrypto/ecies"
	"github.com/annchain/OG/deprecated/ogcrypto_interface"
	"github.com/btcsuite/btcd/btcec"
	"math/big"
)

type SignerSecp256k1 struct {
}

func (s *SignerSecp256k1) GetCryptoType() ogcrypto_interface.CryptoType {
	return ogcrypto_interface.CryptoTypeSecp256k1
}

// Sign calculates an ECDSA signature.
//
// This function is susceptible to chosen plaintext attacks that can leak
// information about the private key that is used for signing. Callers must
// be aware that the given hash cannot be chosen by an adversery. Common
// solution is to hash any input before calculating the signature.
//
// The produced signature is in the [R || S || V] format where V is 0 or 1.
func (s *SignerSecp256k1) Sign(privKey ogcrypto_interface.PrivateKey, msg []byte) ogcrypto_interface.Signature {
	prv, _ := ToECDSA(privKey.KeyBytes)
	hash := deprecated.Sha256(msg)
	if len(hash) != 32 {
		panic(fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash)))
	}
	if prv.Curve != btcec.S256() {
		panic(fmt.Errorf("private key curve is not secp256k1"))
	}
	sig, err := btcec.SignCompact(btcec.S256(), (*btcec.PrivateKey)(prv), hash, false)
	//sig, err := btcec.Sign(btcec.S256(), (*btcec.PrivateKey)(prv), hash, false)
	if err != nil {
		panic(err)
	}
	// Convert to Ethereum signature format with 'recovery id' v at the end.
	v := sig[0] - 27
	copy(sig, sig[1:])
	sig[64] = v
	return deprecated.SignatureFromBytes(s.GetCryptoType(), sig)
}

// VerifySignature checks that the given public key created signature over hash.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
// The signature should have the 64 byte [R || S] format.
func (s *SignerSecp256k1) Verify(pubKey ogcrypto_interface.PublicKey, signature ogcrypto_interface.Signature, msg []byte) bool {
	hash := deprecated.Sha256(msg)
	signature = s.DealRecoverID(signature)
	sigs := signature.SignatureBytes
	sig := &btcec.Signature{R: new(big.Int).SetBytes(sigs[:32]), S: new(big.Int).SetBytes(sigs[32:])}
	key, err := btcec.ParsePubKey(pubKey.KeyBytes, btcec.S256())
	if err != nil {
		fmt.Println(err)
		return false
	}
	// Reject malleable signatures. libsecp256k1 does this check but btcec doesn't.
	if sig.S.Cmp(secp256k1halfN) > 0 {
		fmt.Println("sec")
		return false
	}
	return sig.Verify(hash, key)
}

func (s *SignerSecp256k1) PubKey(privKey ogcrypto_interface.PrivateKey) ogcrypto_interface.PublicKey {
	_, ecdsapub := btcec.PrivKeyFromBytes(btcec.S256(), privKey.KeyBytes)
	pub := FromECDSAPub((*ecdsa.PublicKey)(ecdsapub))
	return deprecated.PublicKeyFromBytes(ogcrypto_interface.CryptoTypeSecp256k1, pub[:])
}

func (s *SignerSecp256k1) PublicKeyFromBytes(b []byte) ogcrypto_interface.PublicKey {
	return deprecated.PublicKeyFromBytes(s.GetCryptoType(), b)
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
	//return nil, nil
}

const sigLength int = 64

func (s *SignerSecp256k1) DealRecoverID(sig ogcrypto_interface.Signature) ogcrypto_interface.Signature {
	l := len(sig.SignatureBytes)
	if l == sigLength+1 {
		sig.SignatureBytes = sig.SignatureBytes[:l-1]
	}
	return sig
}

func (s *SignerSecp256k1) RandomKeyPair() (publicKey ogcrypto_interface.PublicKey, privateKey ogcrypto_interface.PrivateKey) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], ogcrypto.CRandBytes(32))

	privateKey = deprecated.PrivateKeyFromBytes(ogcrypto_interface.CryptoTypeSecp256k1, privKeyBytes[:])
	publicKey = s.PubKey(privateKey)
	return
}

// Ecrecover returns the uncompressed public key that created the given signature.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	pub, err := SigToPub(hash, sig)
	if err != nil {
		return nil, err
	}
	bytes := (*btcec.PublicKey)(pub).SerializeUncompressed()
	return bytes, err
}

// SigToPub returns the public key that created the given signature.
func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, 65)
	btcsig[0] = sig[64] + 27
	copy(btcsig[1:], sig)

	pub, _, err := btcec.RecoverCompact(btcec.S256(), btcsig, hash)
	return (*ecdsa.PublicKey)(pub), err
}

func (s *SignerSecp256k1) CanRecoverPubFromSig() bool {
	return false
}
