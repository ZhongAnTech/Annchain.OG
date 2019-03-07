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
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types"
)

type CryptoType int8

const (
	CryptoTypeEd25519 CryptoType = iota
	CryptoTypeSecp256k1
)

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
	priv = PrivateKey{
		Type:  CryptoType(bytes[0]),
		Bytes: bytes[1:],
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

func (k *PrivateKey) String() string {
	var bytes []byte
	bytes = append(bytes, byte(k.Type))
	bytes = append(bytes, k.Bytes...)
	return hexutil.Encode(bytes)
}

func (p *PublicKey) String() string {
	var bytes []byte
	bytes = append(bytes, byte(p.Type))
	bytes = append(bytes, p.Bytes...)
	return hexutil.Encode(bytes)
}

func NewSigner(cryptoType CryptoType) Signer {
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
func CreateAddress(b types.Address, nonce uint64) types.Address {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, nonce)
	return types.BytesToAddress(Keccak256([]byte{0xff}, b.ToBytes()[:], bs)[12:])
}

// CreateAddress2 creates an ethereum address given the address bytes, initial
// contract code hash and a salt.
func CreateAddress2(b types.Address, salt [32]byte, inithash []byte) types.Address {
	return types.BytesToAddress(Keccak256([]byte{0xff}, b.ToBytes()[:], salt[:], inithash)[12:])
}
