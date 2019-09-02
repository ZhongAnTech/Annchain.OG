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
package crypto

import (
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/kyber/v3"

	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/poc/extra25519"
	"github.com/annchain/kyber/v3/encrypt/ecies"
	"github.com/annchain/kyber/v3/group/edwards25519"
)

type CryptoType int8

const (
	CryptoTypeEd25519 CryptoType = iota
	CryptoTypeSecp256k1
)

var CryptoNameMap = map[string]CryptoType{
	"secp256k1": CryptoTypeSecp256k1,
	"ed25519":   CryptoTypeEd25519,
}

type PrivateKey struct {
	Type  CryptoType
	Bytes []byte
}

type PublicKey struct {
	Type  CryptoType
	Bytes []byte
}

type Signature struct {
	Type  CryptoType
	Bytes []byte
}

func PrivateKeyFromBytes(typev CryptoType, bytes []byte) PrivateKey {
	return PrivateKey{Type: typev, Bytes: bytes}
}
func PublicKeyFromBytes(typev CryptoType, bytes []byte) PublicKey {
	return PublicKey{Type: typev, Bytes: bytes}
}
func SignatureFromBytes(typev CryptoType, bytes []byte) Signature {
	return Signature{Type: typev, Bytes: bytes}
}

func PrivateKeyFromString(value string) (priv PrivateKey, err error) {
	bytes, err := hexutil.Decode(value)
	if err != nil {
		return
	}
	cryptoType := CryptoTypeSecp256k1
	if len(bytes) == 33 {
		cryptoType = CryptoType(bytes[0])
		bytes = bytes[1:]
	}
	priv = PrivateKey{
		Type:  cryptoType,
		Bytes: bytes,
	}
	return
}

func PublicKeyFromString(value string) (pub PublicKey, err error) {
	bytes, err := hexutil.Decode(value)
	if err != nil {
		return
	}
	pub = PublicKey{
		Type:  CryptoType(bytes[0]),
		Bytes: bytes[1:],
	}
	return
}

func Secp256k1PublicKeyFromString(pkStr string) (pub PublicKey, err error) {
	return PublicKeyFromStringWithCryptoType("secp256k1", pkStr)
}

func PublicKeyFromStringWithCryptoType(ct, pkstr string) (pub PublicKey, err error) {
	cryptoType, ok := CryptoNameMap[ct]
	if !ok {
		err = fmt.Errorf("unknown crypto type: %s", ct)
		return
	}
	pk, err := hexutil.Decode(pkstr)
	if err != nil {
		return
	}
	pub = PublicKey{
		Type:  cryptoType,
		Bytes: pk,
	}
	return
}

func (k *PrivateKey) String() string {
	var bytes []byte
	//bytes = append(bytes, byte(k.Type))
	bytes = append(bytes, k.Bytes...)
	return hexutil.Encode(bytes)
}

func (p *PrivateKey) PublicKey() *PublicKey {
	s := NewSigner(p.Type)
	pub := s.PubKey(*p)
	return &pub
}

func (p *PublicKey) Encrypt(m []byte) (ct []byte, err error) {
	s := NewSigner(p.Type)
	return s.Encrypt(*p, m)
}

func (p *PrivateKey) Decrypt(ct []byte) (m []byte, err error) {
	s := NewSigner(p.Type)
	return s.Decrypt(*p, ct)
}

type KyberEd22519PrivKey struct {
	PrivateKey kyber.Scalar
	Suit       *edwards25519.SuiteEd25519
}

func (p *KyberEd22519PrivKey) Decrypt(cipherText []byte) (m []byte, err error) {
	return ecies.Decrypt(p.Suit, p.PrivateKey, cipherText, p.Suit.Hash)
}

func (p *PrivateKey) ToKyberEd25519PrivKey() *KyberEd22519PrivKey {
	var edPrivKey [32]byte
	var curvPrivKey [64]byte
	copy(curvPrivKey[:], p.Bytes[:64])
	extra25519.PrivateKeyToCurve25519(&edPrivKey, &curvPrivKey)
	privateKey, err := edwards25519.UnmarshalBinaryScalar(edPrivKey[:32])
	suite := edwards25519.NewBlakeSHA256Ed25519()
	if err != nil {
		panic(err)
	}
	return &KyberEd22519PrivKey{
		PrivateKey: privateKey,
		Suit:       suite,
	}
}

func (p *PublicKey) Address() common.Address {
	s := NewSigner(p.Type)
	return s.Address(*p)
}

func (p *PublicKey) String() string {
	var bytes []byte
	//bytes = append(bytes, byte(p.Type))
	bytes = append(bytes, p.Bytes...)
	return hexutil.Encode(bytes)
}

func NewSigner(cryptoType CryptoType) ISigner {
	if cryptoType == CryptoTypeEd25519 {
		return &SignerEd25519{}
	} else if cryptoType == CryptoTypeSecp256k1 {
		return &SignerSecp256k1{}
	}
	return nil
}

func (c CryptoType) String() string {
	if c == CryptoTypeEd25519 {
		return "ed25519"
	} else if c == CryptoTypeSecp256k1 {
		return "secp256k1"
	}
	return "unknown"
}

// CreateAddress creates an ethereum address given the bytes and the nonce
func CreateAddress(b common.Address, nonce uint64) common.Address {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, nonce)
	return common.BytesToAddress(Keccak256([]byte{0xff}, b.ToBytes()[:], bs)[12:])
}

// CreateAddress2 creates an ethereum address given the address bytes, initial
// contract code hash and a salt.
func CreateAddress2(b common.Address, salt [32]byte, inithash []byte) common.Address {
	return common.BytesToAddress(Keccak256([]byte{0xff}, b.ToBytes()[:], salt[:], inithash)[12:])
}

type PublicKeys []PublicKey

func (h PublicKeys) Len() int {
	return len(h)
}
func (h PublicKeys) Less(i, j int) bool {
	return h[i].String() < h[j].String()
}

func (h PublicKeys) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}
