package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto/ecies"
	"github.com/annchain/OG/types"
	"math/big"
	"testing"
	"time"
)

func TestPublicKey_Encrypt(t *testing.T) {
	for i := 0; i < 2; i++ {
		cType := CryptoType(i)
		fmt.Println(cType)
		s := NewSigner(cType)
		pk, sk, _ := s.RandomKeyPair()
		fmt.Println("pk", pk.String())
		fmt.Println("sk", sk.String())
		msg := []byte("hello og  this is a secret msg , no one knows wipkhfdii75438048584653543543skj76895804iri4356345h" +
			"ufidurehfkkjfri566878798y5rejiodijfjioi;454646855455uiyrsduihfi54sdodoootoprew5468rre")
		fmt.Println(string(msg))
		fmt.Println(len(msg))
		cf, err := pk.Encrypt(msg)
		fmt.Println(len(cf), hex.EncodeToString(cf), err)
		m, err := sk.Decrypt(cf)
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
		Curve: S256(),
		X:     x,
		Y:     y,
	}
	ecdsapriv := ecdsa.PrivateKey{ecdsapub, D}
	fmt.Println(ecdsapriv)
	pk := PublicKeyFromBytes(CryptoTypeSecp256k1, FromECDSAPub(&ecdsapub))
	sk := PrivateKeyFromBytes(CryptoTypeSecp256k1, FromECDSA(&ecdsapriv))

	//priv:= PrivateKey{PublicKey{}}
	msg := []byte("hello og  this is a secret msg , no one knows wipkhfdii75438048584653543543skj76895804iri4356345h" +
		"ufidurehfkkjfri566878798y5rejiodijfjioi;454646855455uiyrsduihfi54sdodoootoprew5468rre")
	pub2, err := UnmarshalPubkey(pk.Bytes)
	if err != nil {
		panic(err)
	}
	if ecdsapub != *pub2 {
		fmt.Println(pub2)
		panic("not equal ")
	}
	eciesPub := ecies.ImportECDSAPublic(pub2)
	cf, err := ecies.Encrypt(rand.Reader, eciesPub, msg, nil, nil)
	//cf, err := pk.Encrypt(msg)
	fmt.Println(len(cf), hex.EncodeToString(cf), err)
	prive, err := ToECDSA(sk.Bytes)
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
func TestBenchMarks(t *testing.T) {
	type TestTx struct {
		types.Txi
		PlainData []byte
		CipherData []byte
	}
	var  txs []*TestTx
	var Txlen = 40000
	for i:=0;i<Txlen;i++ {
		tx := types.RandomTx()
		tx.Data = []byte("jhfffhhgfhgf46666856544563544535636568654864546546ewfjnfdjlfjldkjkflkjflkdsl;kfdfkjjkfsd;lsdl;kdfl;kjfjfsj;sd54645656854545435454")
		testTx := TestTx{tx,tx.SignatureTargets(),nil}
		txs = append(txs,&testTx)
	}
	signerSecp:= NewSigner(CryptoTypeSecp256k1)
	pubkey,privKey,_:= signerSecp.RandomKeyPair()
	signerEd:=NewSigner(CryptoTypeEd25519)
	pubEd,privEd,_ := signerEd.RandomKeyPair()
	start:= time.Now()
	fmt.Println("number of tx ",Txlen)
	for i, tx:= range txs {
		txs[i].GetBase().Signature = signerSecp.Sign(privKey,tx.PlainData).Bytes
	}
	second := time.Now()
	fmt.Println(second.Sub(start),"for  secp256k1 sign"," pubKeyLen:", len(pubkey.Bytes),"len signatue:",
		len(txs[0].GetBase().Signature),"len target:",len(txs[0].PlainData))
	for i, tx:= range txs {
		ok:= signerSecp.Verify(pubkey ,Signature{ signerSecp.GetCryptoType(), tx.GetBase().Signature}, tx.PlainData)
		if !ok {
			t.Fatal(ok,i,tx)
		}
	}
	third:= time.Now()
	fmt.Println(third.Sub(second),"used for secp256k1 verify")

	for i, tx:= range txs {
		txs[i].GetBase().Signature = signerEd.Sign(privEd,tx.PlainData).Bytes
	}
	four := time.Now()
	fmt.Println(four.Sub(third),"for ed25519 sign"," pubKeyLen:", len(pubEd.Bytes),"len signatue:",
		len(txs[0].GetBase().Signature),"len target:",len(txs[0].PlainData))
	for i , tx:= range txs {
		ok:= signerEd.Verify(pubEd ,Signature{ signerEd.GetCryptoType(), tx.GetBase().Signature}, tx.PlainData)
		if !ok {
			t.Fatal(ok,i,tx)
		}
	}
	five := time.Now()
	fmt.Println(five.Sub(four),"used for ed25519 verify")

	for i, tx:= range txs {
		var err error
		txs[i].CipherData,err = signerSecp.Encrypt(pubkey,tx.PlainData)
		if err!=nil {
			t.Fatal(err,i)
		}
	}
	fmt.Println()
	six := time.Now()
	fmt.Println(six.Sub(five),"for secp256k1 enc","len cipherdata:",len(txs[0].CipherData),"len plaindata:",len(txs[0].PlainData))
	for i , tx:= range txs {
		d, err:= signerSecp.Decrypt(privKey ,tx.CipherData)
		if err!=nil  {
			t.Fatal(err,i,tx)
		}
		if !bytes.Equal(d, tx.PlainData) {
			t.Fatal(fmt.Sprintf("secp dec error , i %d got %v, want %v",i,d,tx.PlainData))
		}
	}
	seven:= time.Now()
	fmt.Println(seven.Sub(six),"for secp256k1 dec")
	for i, tx:= range txs {
		var err error
		txs[i].CipherData,err = signerEd.Encrypt(pubEd,tx.PlainData)
		if err!=nil {
			t.Fatal(err,i)
		}
	}
	eight := time.Now()
	kyberKey:= privEd.ToKyberEd25519PrivKey()
	fmt.Println(eight.Sub(seven),"for ed25519 enc","len cipherdata:",len(txs[0].CipherData),"len plaindata:",len(txs[0].PlainData))
	for i, tx:= range txs {
		d, err:= kyberKey.Decrypt(tx.CipherData)
		if err!=nil  {
			t.Fatal(err,i,tx)
		}
		if !bytes.Equal(d, tx.PlainData) {
			t.Fatal(fmt.Sprintf("ed25519 dec error , i %d got %v, want %v",i,d,tx.PlainData))
		}

	}
	nine  := time.Now()
	fmt.Println(nine.Sub(eight),"for ed25519 dec")
}

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