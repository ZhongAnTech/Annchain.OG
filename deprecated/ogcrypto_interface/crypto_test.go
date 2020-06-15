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
package ogcrypto_interface

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/deprecated"
	ogcrypto2 "github.com/annchain/OG/deprecated/ogcrypto"
	"github.com/annchain/OG/deprecated/ogcrypto/ecies"

	"math/big"
	"testing"
)

func NewSigner(cryptoType CryptoType) ISigner {
	if cryptoType == CryptoTypeEd25519 {
		return &ogcrypto2.SignerEd25519{}
	} else if cryptoType == CryptoTypeSecp256k1 {
		return &ogcrypto2.SignerSecp256k1{}
	}
	return nil
}

func TestPublicKey_Encrypt(t *testing.T) {
	for i := 0; i < 2; i++ {
		cType := CryptoType(i)
		fmt.Println(cType)
		s := NewSigner(cType)
		pk, sk := s.RandomKeyPair()
		fmt.Println("pk", pk.String())
		fmt.Println("sk", sk.String())
		msg := []byte("hello og  this is a secret msg , no one knows wipkhfdii75438048584653543543skj76895804iri4356345h" +
			"ufidurehfkkjfri566878798y5rejiodijfjioi;454646855455uiyrsduihfi54sdodoootoprew5468rre")
		fmt.Println(string(msg))
		fmt.Println(len(msg))
		cf, err := s.Encrypt(pk, msg)
		fmt.Println(len(cf), hex.EncodeToString(cf), err)
		m, err := s.Decrypt(sk, cf)
		fmt.Println(string(m), err)
		if !bytes.Equal(m, msg) {
			t.Fatal("encrypt or decrypt error")
		}
	}
}

func TestPrivateKey_Decrypt(t *testing.T) {
	x, _ := big.NewInt(0).SetString("25860957663402118420385219759979651311493566861420536707157658171589998503027", 0)
	y, _ := big.NewInt(0).SetString("86240112990591864153796259798532127250481760794327649797956296817696385215375", 0)
	D, _ := big.NewInt(0).SetString("112339918029554102382967166016087130388477159317095521880422462723592214463164", 0)
	ecdsapub := ecdsa.PublicKey{
		Curve: ogcrypto2.S256(),
		X:     x,
		Y:     y,
	}
	ecdsapriv := ecdsa.PrivateKey{ecdsapub, D}
	fmt.Println(ecdsapriv)
	pk := deprecated.PublicKeyFromBytes(CryptoTypeSecp256k1, ogcrypto2.FromECDSAPub(&ecdsapub))
	sk := deprecated.PrivateKeyFromBytes(CryptoTypeSecp256k1, ogcrypto2.FromECDSA(&ecdsapriv))

	//priv:= PrivateKey{PublicKey{}}
	msg := []byte("hello og  this is a secret msg , no one knows wipkhfdii75438048584653543543skj76895804iri4356345h" +
		"ufidurehfkkjfri566878798y5rejiodijfjioi;454646855455uiyrsduihfi54sdodoootoprew5468rre")
	pub2, err := ogcrypto2.UnmarshalPubkey(pk.KeyBytes)
	if err != nil {
		panic(err)
	}
	if ecdsapub.X.Cmp(pub2.X) != 0 || ecdsapub.Y.Cmp(pub2.Y) != 0 {
		fmt.Println(*pub2)
		fmt.Println(ecdsapub)
		panic("not equal")
	}
	eciesPub := ecies.ImportECDSAPublic(pub2)
	cf, err := ecies.Encrypt(rand.Reader, eciesPub, msg, nil, nil)
	//cf, err := pk.Encrypt(msg)
	fmt.Println(len(cf), hex.EncodeToString(cf), err)
	prive, err := ogcrypto2.ToECDSA(sk.KeyBytes)
	fmt.Println(prive)
	ecisesPriv := ecies.ImportECDSA(prive)
	fmt.Println(ecisesPriv)
	m, err := ecisesPriv.Decrypt(cf, nil, nil)
	//m ,err:= sk.Decrypt(cf)
	fmt.Println(string(m), err)
	if err != nil || !bytes.Equal(m, msg) {
		t.Fatal("encrypt or decrypt error", err)
	}
}

//tese signer benchmarks
//func TestBenchMarks(t *testing.T) {
//	type TestTx struct {
//		types.Txi
//		PlainData  []byte
//		CipherData []byte
//	}
//	var txs []*TestTx
//	var Txlen = 40000
//	for i := 0; i < Txlen; i++ {
//		tx := archive.RandomTx()
//		tx.Data = []byte("jhfffhhgfhgf46666856544563544535636568654864546546ewfjnfdjlfjldkjkflkjflkdsl;kfdfkjjkfsd;lsdl;kdfl;kjfjfsj;sd54645656854545435454")
//		testTx := TestTx{tx, tx.SignatureTargets(), nil}
//		txs = append(txs, &testTx)
//	}
//	signerSecp := NewSigner(CryptoTypeSecp256k1)
//	pubkey, privKey := signerSecp.RandomKeyPair()
//	signerEd := NewSigner(CryptoTypeEd25519)
//	pubEd, privEd := signerEd.RandomKeyPair()
//	start := time.Now()
//	fmt.Println("number of tx ", Txlen)
//	for i, tx := range txs {
//		txs[i].GetBase().Signature = signerSecp.Sign(privKey, tx.PlainData).SignatureBytes
//	}
//	second := time.Now()
//	fmt.Println(second.Sub(start), "for  secp256k1 sign", " pubKeyLen:", len(pubkey.KeyBytes), "len signatue:",
//		len(txs[0].GetBase().Signature), "len target:", len(txs[0].PlainData))
//	for i, tx := range txs {
//		ok := signerSecp.Verify(pubkey, Signature{signerSecp.GetCryptoType(), tx.GetBase().Signature}, tx.PlainData)
//		if !ok {
//			t.Fatal(ok, i, tx)
//		}
//	}
//	third := time.Now()
//	fmt.Println(third.Sub(second), "used for secp256k1 verify")
//
//	for i, tx := range txs {
//		txs[i].GetBase().Signature = signerEd.Sign(privEd, tx.PlainData).SignatureBytes
//	}
//	four := time.Now()
//	fmt.Println(four.Sub(third), "for ed25519 sign", " pubKeyLen:", len(pubEd.KeyBytes), "len signatue:",
//		len(txs[0].GetBase().Signature), "len target:", len(txs[0].PlainData))
//	for i, tx := range txs {
//		ok := signerEd.Verify(pubEd, Signature{signerEd.GetCryptoType(), tx.GetBase().Signature}, tx.PlainData)
//		if !ok {
//			t.Fatal(ok, i, tx)
//		}
//	}
//	five := time.Now()
//	fmt.Println(five.Sub(four), "used for ed25519 verify")
//
//	for i, tx := range txs {
//		var err error
//		txs[i].CipherData, err = signerSecp.Encrypt(pubkey, tx.PlainData)
//		if err != nil {
//			t.Fatal(err, i)
//		}
//	}
//	fmt.Println()
//	six := time.Now()
//	fmt.Println(six.Sub(five), "for secp256k1 enc", "len cipherdata:", len(txs[0].CipherData), "len plaindata:", len(txs[0].PlainData))
//	for i, tx := range txs {
//		d, err := signerSecp.Decrypt(privKey, tx.CipherData)
//		if err != nil {
//			t.Fatal(err, i, tx)
//		}
//		if !bytes.Equal(d, tx.PlainData) {
//			t.Fatal(fmt.Sprintf("secp dec error , i %d got %v, want %v", i, d, tx.PlainData))
//		}
//	}
//	seven := time.Now()
//	fmt.Println(seven.Sub(six), "for secp256k1 dec")
//	for i, tx := range txs {
//		var err error
//		txs[i].CipherData, err = signerEd.Encrypt(pubEd, tx.PlainData)
//		if err != nil {
//			t.Fatal(err, i)
//		}
//	}
//	eight := time.Now()
//	kyberKey := privEd.ToKyberEd25519PrivKey()
//	fmt.Println(eight.Sub(seven), "for ed25519 enc", "len cipherdata:", len(txs[0].CipherData), "len plaindata:", len(txs[0].PlainData))
//	for i, tx := range txs {
//		d, err := kyberKey.Decrypt(tx.CipherData)
//		if err != nil {
//			t.Fatal(err, i, tx)
//		}
//		if !bytes.Equal(d, tx.PlainData) {
//			t.Fatal(fmt.Sprintf("ed25519 dec error , i %d got %v, want %v", i, d, tx.PlainData))
//		}
//
//	}
//	nine := time.Now()
//	fmt.Println(nine.Sub(eight), "for ed25519 dec")
//}

//=== RUN   TestBenchMarks
//number of tx  40000
//12.62s for  secp256k1 sign  pubKeyLen: 65 len signatue: 65 len target: 185
//11.212s used for secp256k1 verify
//4.433s for ed25519 sign  pubKeyLen: 32 len signatue: 64 len target: 185
//11.887s used for ed25519 verify
//
//15.905s for secp256k1 enc len cipherdata: 298 len plaindata: 185
//15.72s for secp256k1 dec
//22.045s for ed25519 enc len cipherdata: 233 len plaindata: 185
//11.402s for ed25519 dec
//--- PASS: TestBenchMarks (105.89s)
//PASS
