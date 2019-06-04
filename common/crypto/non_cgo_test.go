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
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/types"
	"testing"
	"time"
)


func TestSignerNewPrivKeyGO(t *testing.T) {
	t.Parallel()

	signer := SignerSecp256k1{}
	pk, priv := signer.RandomKeyPair()

	b := []byte("foo")
	sig := signer.Sign(priv, b)
	if !signer.Verify(pk, sig, b) {
		t.Fatalf("vertfy failed")
	}
	fmt.Println(hex.EncodeToString(sig.Bytes))

	signer2:= SignerSecp256k1Go{}
	sig2 := signer2.Sign(priv, b)
	fmt.Println(hex.EncodeToString(sig2.Bytes))
	if !signer2.Verify(pk, sig, b) {
		t.Fatalf("vertfy failed")
	}

}

func TestSignBenchMarks(t *testing.T) {
	signer := SignerSecp256k1{}
	pk, priv := signer.RandomKeyPair()
	signer2:= SignerSecp256k1Go{}
	var txs1 types.Txis
	var txs2  types.Txis
	N := 10000

	for i:=0;i<N;i++ {
		txs1 = append(txs1,types.RandomTx())
	}
	for i:=0;i<N;i++ {
		txs2 = append(txs2,types.RandomTx())
	}
	fmt.Println("started")
	start := time.Now()
	for i:=0;i<len(txs1);i++ {
		b:=txs1[i].SignatureTargets()
		sig := signer.Sign(priv, b)
		txs1[i].GetBase().Signature = sig.Bytes
		//if !signer.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp cgo used for signing ", time.Since(start))
	start = time.Now()
	for i:=0;i<len(txs1);i++ {
		b:=txs1[i].SignatureTargets()
		sig := signer2.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.Bytes
		//if !signer2.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp go used for signing ", time.Since(start))
	start = time.Now()
	for i:=0;i<len(txs2);i++ {
		b:=txs2[i].SignatureTargets()
		sig := signer2.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.Bytes
		//if !signer2.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp go used for signing ", time.Since(start))
	start = time.Now()
	for i:=0;i<len(txs2);i++ {
		b:=txs2[i].SignatureTargets()
		sig := signer.Sign(priv, b)
		txs2[i].GetBase().Signature = sig.Bytes
		//if !signer.Verify(pk, sig, b) {
		//	t.Fatalf("vertfy failed")
		//}
	}
	fmt.Println("secp cgo used for signing ", time.Since(start))
	start = time.Now()
	for i:=0;i<len(txs1);i++ {
		b := txs1[i].SignatureTargets()
		sig := txs1[i].GetBase().Signature
		if !signer2.Verify(pk, SignatureFromBytes(CryptoTypeSecp256k1, sig), b) {
			t.Fatalf("vertfy failed")
		}
	}
	fmt.Println("secp go used for verifying ", time.Since(start))
	start = time.Now()
	for i:=0;i<len(txs1);i++ {
		b := txs1[i].SignatureTargets()
		sig := txs1[i].GetBase().Signature
		if !signer.Verify(pk, SignatureFromBytes(CryptoTypeSecp256k1, sig), b) {
			t.Fatalf("vertfy failed")
		}
	}
	fmt.Println("secp cgo used for verifying ", time.Since(start))
	start = time.Now()
	for i:=0;i<len(txs1);i++ {
		b := txs2[i].SignatureTargets()
		sig := txs2[i].GetBase().Signature
		if !signer.Verify(pk, SignatureFromBytes(CryptoTypeSecp256k1, sig), b) {
			t.Fatalf("vertfy failed")
		}
	}
	fmt.Println("secp cgo used for verifying ", time.Since(start))
	start = time.Now()
	for i:=0;i<len(txs1);i++ {
		b := txs1[i].SignatureTargets()
		sig := txs1[i].GetBase().Signature
		if !signer2.Verify(pk, SignatureFromBytes(CryptoTypeSecp256k1, sig), b) {
			t.Fatalf("vertfy failed")
		}
	}
	fmt.Println("secp go used for verifying ", time.Since(start))

}
