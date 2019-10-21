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

package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto/secp256k1"
	"github.com/annchain/OG/common/math"

	ecdsabtcec "github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"
)

type SignerSecp256k1cgo struct {
}

func (s *SignerSecp256k1cgo) GetCryptoType() CryptoType {
	return CryptoTypeSecp256k1
}

func (s *SignerSecp256k1cgo) Sign(privKey PrivateKey, msg []byte) Signature {
	priv, _ := ToECDSA(privKey.Bytes)
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

func (s *SignerSecp256k1cgo) PubKey(privKey PrivateKey) PublicKey {
	_, ecdsapub := ecdsabtcec.PrivKeyFromBytes(ecdsabtcec.S256(), privKey.Bytes)
	pub := FromECDSAPub((*ecdsa.PublicKey)(ecdsapub))
	return PublicKeyFromBytes(CryptoTypeSecp256k1, pub[:])
}

func (s *SignerSecp256k1cgo) PublicKeyFromBytes(b []byte) PublicKey {
	return PublicKeyFromBytes(s.GetCryptoType(), b)
}

func (s *SignerSecp256k1cgo) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	signature = s.DealRecoverID(signature)
	sig := signature.Bytes
	return secp256k1.VerifySignature(pubKey.Bytes, Sha256(msg), sig)
}

func (s *SignerSecp256k1cgo) RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], CRandBytes(32))

	privateKey = PrivateKeyFromBytes(CryptoTypeSecp256k1, privKeyBytes[:])
	publicKey = s.PubKey(privateKey)
	return
}

func (s *SignerSecp256k1cgo) DealRecoverID(sig Signature) Signature {
	l := len(sig.Bytes)
	if l == sigLength+1 {
		sig.Bytes = sig.Bytes[:l-1]
	}
	return sig
}

func TestSignerNewPrivKeyCGO(t *testing.T) {
	t.Parallel()
	signer := SignerSecp256k1cgo{}
	signer2 := SignerSecp256k1{}
	for i := 0; i < 10; i++ {
		pk, priv := signer.RandomKeyPair()
		//fmt.Println(priv.String())
		//fmt.Println(pk.String())
		b := []byte("foohhhhjkhhj3488984984984jjjdjsdjks")
		sig := signer.Sign(priv, b)
		if !signer.Verify(pk, sig, b) {
			t.Fatalf("vertfy failed")
		}
		fmt.Println(hex.EncodeToString(sig.Bytes))

		sig2 := signer2.Sign(priv, b)
		fmt.Println(hex.EncodeToString(sig2.Bytes))
		if !signer2.Verify(pk, sig2, b) {
			t.Fatalf("vertfy failed")
		}
		fmt.Println(" ", i)
	}

}

func TestSignerNewPrivKeyCGOF(t *testing.T) {
	t.Parallel()
	signer := SignerSecp256k1cgo{}
	signer2 := SignerSecp256k1{}
	for i := 0; i < 1000; i++ {
		pk, priv := signer.RandomKeyPair()
		//fmt.Println(priv.String())
		//fmt.Println(pk.String())
		b := []byte("foohhhhjkhhj3488984984984jjjdjsdjks")
		//sig := signer.Sign(priv, b)
		//if !signer.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
		//fmt.Println(hex.EncodeToString(sig.Bytes))

		sig2 := signer2.Sign(priv, b)
		//fmt.Println(hex.EncodeToString(sig2.Bytes))
		if !signer2.Verify(pk, sig2, b) {
			t.Fatalf("vertfy failed")
		}
		//fmt.Println(" ", i)
	}

}

func TestSignBenchMarksCgo(t *testing.T) {
	signer := SignerSecp256k1cgo{}
	pk, priv := signer.RandomKeyPair()
	signer2 := SignerSecp256k1{}
	var txs1 ogmessage.Txis
	var txs2 ogmessage.Txis
	N := 10000

	for i := 0; i < N; i++ {
		txs1 = append(txs1, ogmessage.RandomTx())
	}
	for i := 0; i < N; i++ {
		txs2 = append(txs2, ogmessage.RandomTx())
	}
	fmt.Println("started")
	start := time.Now()
	for i := 0; i < len(txs1); i++ {
		b := txs1[i].SignatureTargets()
		sig := signer.Sign(priv, b)
		txs1[i].GetBase().Signature = sig.Bytes
		//if !signer.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp cgo used for signing ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs1); i++ {
		b := txs1[i].SignatureTargets()
		sig := signer2.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.Bytes
		//if !signer2.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp go used for signing ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs2); i++ {
		b := txs2[i].SignatureTargets()
		sig := signer2.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.Bytes
		//if !signer2.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp go used for signing ", time.Since(start))
	start = time.Now()
	for i := 0; i < len(txs2); i++ {
		b := txs2[i].SignatureTargets()
		sig := signer.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.Bytes
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
