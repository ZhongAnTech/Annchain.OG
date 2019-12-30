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

package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/og/types/archive"

	"github.com/btcsuite/btcd/btcec"
	"math/big"
	"testing"
	"time"
)

type SignerSecp256k1Go struct {
}

func (s *SignerSecp256k1Go) GetCryptoType() CryptoType {
	return CryptoTypeSecp256k1
}

// Sign calculates an ECDSA signature.
//
// This function is susceptible to chosen plaintext attacks that can leak
// information about the private key that is used for signing. Callers must
// be aware that the given hash cannot be chosen by an adversery. Common
// solution is to hash any input before calculating the signature.
//
// The produced signature is in the [R || S || V] format where V is 0 or 1.
func (s *SignerSecp256k1Go) Sign(privKey PrivateKey, msg []byte) Signature {
	prv, _ := ToECDSA(privKey.KeyBytes)
	hash := Sha256(msg)
	if len(hash) != 32 {
		panic(fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash)))
	}
	if prv.Curve != btcec.S256() {
		panic(fmt.Errorf("private key curve is not secp256k1"))
	}
	key := (*btcec.PrivateKey)(prv)
	signature, err := key.Sign(hash)
	if err != nil {
		panic(err)
	}
	sig := toByte(signature.R, signature.S)
	// Convert to Ethereum signature format with 'recovery id' v at the end.
	if len(sig) == sigLength+1 {
		v := sig[0]
		copy(sig, sig[1:])
		//sig[64] = v
		_ = v
	}
	//copy(sig, sig[1:])
	//v := sig[0] - 27
	//sig[64] = v
	return SignatureFromBytes(s.GetCryptoType(), sig)
}

func toByte(R *big.Int, S *big.Int) []byte {
	rb := canonicalizeInt(R)
	//sigS := sig.S
	//if sigS.Cmp(S256().halfOrder) == 1 {
	//	sigS = new(big.Int).Sub(S256().N, sigS)
	//}
	sb := canonicalizeInt(S)
	b := make([]byte, len(rb)+len(sb))
	copy(b, rb)
	copy(b[len(rb):], sb)
	return b
}

func canonicalizeInt(val *big.Int) []byte {
	b := val.Bytes()
	if len(b) == 0 {
		b = []byte{0x00}
	}
	if b[0]&0x80 != 0 {
		paddedBytes := make([]byte, len(b)+1)
		copy(paddedBytes[1:], b)
		b = paddedBytes
	}
	return b
}

// VerifySignature checks that the given public key created signature over hash.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
// The signature should have the 64 byte [R || S] format.
func (s *SignerSecp256k1Go) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	hash := Sha256(msg)
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

func (s *SignerSecp256k1Go) DealRecoverID(sig Signature) Signature {
	l := len(sig.SignatureBytes)
	if l == sigLength+1 {
		sig.SignatureBytes = sig.SignatureBytes[:l-1]
	}
	return sig
}

func (s *SignerSecp256k1Go) PubKey(privKey PrivateKey) PublicKey {
	_, ecdsapub := btcec.PrivKeyFromBytes(btcec.S256(), privKey.KeyBytes)
	pub := FromECDSAPub((*ecdsa.PublicKey)(ecdsapub))
	return PublicKeyFromBytes(CryptoTypeSecp256k1, pub[:])
}

func TestSignerNewPrivKeyGO(t *testing.T) {
	t.Parallel()
	signer := SignerSecp256k1{}
	for i := 0; i < 10; i++ {
		pk, priv := signer.RandomKeyPair()
		//fmt.Println(priv.String())
		//fmt.Println(pk.String())
		b := []byte("foohhhhjkhhj3488984984984jjjdjsdjks")
		sig := signer.Sign(priv, b)
		if !signer.Verify(pk, sig, b) {
			t.Fatalf("vertfy failed")
		}
		fmt.Println(hex.EncodeToString(sig.SignatureBytes))

		signer2 := SignerSecp256k1Go{}
		sig2 := signer2.Sign(priv, b)
		fmt.Println(hex.EncodeToString(sig2.SignatureBytes))
		if !signer2.Verify(pk, sig2, b) {
			t.Fatalf("vertfy failed")
		}
		fmt.Println(i)
	}

}

func TestSignBenchMarks(t *testing.T) {
	signer := SignerSecp256k1{}
	pk, priv := signer.RandomKeyPair()
	signer2 := SignerSecp256k1Go{}
	var txs1 types.Txis
	var txs2 types.Txis
	N := 10000

	for i := 0; i < N; i++ {
		txs1 = append(txs1, archive.RandomTx())
	}
	for i := 0; i < N; i++ {
		txs2 = append(txs2, archive.RandomTx())
	}
	fmt.Println("started")
	start := time.Now()
	for i := 0; i < len(txs1); i++ {
		b := txs1[i].SignatureTargets()
		sig := signer.Sign(priv, b)
		txs1[i].GetBase().Signature = sig.SignatureBytes
		//if !signer.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp cgo used for signing ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs1); i++ {
		b := txs1[i].SignatureTargets()
		sig := signer2.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.SignatureBytes
		//if !signer2.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp go used for signing ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs2); i++ {
		b := txs2[i].SignatureTargets()
		sig := signer2.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.SignatureBytes
		//if !signer2.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp go used for signing ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs2); i++ {
		b := txs2[i].SignatureTargets()
		sig := signer.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.SignatureBytes
		//if !signer.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp cgo used for signing ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs1); i++ {
		b := txs1[i].SignatureTargets()
		sig := txs1[i].GetBase().Signature
		if !signer2.Verify(pk, SignatureFromBytes(CryptoTypeSecp256k1, sig), b) {
			t.Fatalf("vertfy failed")
		}
	}
	fmt.Println("secp go used for verifying ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs1); i++ {
		b := txs1[i].SignatureTargets()
		sig := txs1[i].GetBase().Signature
		if !signer.Verify(pk, SignatureFromBytes(CryptoTypeSecp256k1, sig), b) {
			t.Fatalf("vertfy failed")
		}
	}
	fmt.Println("secp cgo used for verifying ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs1); i++ {
		b := txs2[i].SignatureTargets()
		sig := txs2[i].GetBase().Signature
		if !signer.Verify(pk, SignatureFromBytes(CryptoTypeSecp256k1, sig), b) {
			t.Fatalf("vertfy failed")
		}
	}
	fmt.Println("secp cgo used for verifying ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs1); i++ {
		b := txs1[i].SignatureTargets()
		sig := txs1[i].GetBase().Signature
		if !signer2.Verify(pk, SignatureFromBytes(CryptoTypeSecp256k1, sig), b) {
			t.Fatalf("vertfy failed")
		}
	}
	fmt.Println("secp go used for verifying ", time.Since(start))

}
