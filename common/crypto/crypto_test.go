package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto/ecies"
	"math/big"
	"testing"
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
