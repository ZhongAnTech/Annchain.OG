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
	"fmt"
	"github.com/annchain/OG/arefactor/common/hexutil"
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
	Type     CryptoType
	KeyBytes []byte
}

func (p *PrivateKey) ToBytes() []byte {
	var bytes []byte
	bytes = append(bytes, byte(p.Type))
	bytes = append(bytes, p.KeyBytes...)
	return bytes
}

func (p *PrivateKey) DebugString() string {
	return fmt.Sprintf("privk%d:%s", p.Type, hexutil.ToHex(p.KeyBytes))
}

func (p *PrivateKey) String() string {
	return hexutil.ToHex(p.ToBytes())
}

type PublicKey struct {
	Type     CryptoType
	KeyBytes []byte
}

func (p *PublicKey) ToBytes() []byte {
	var bytes []byte
	bytes = append(bytes, byte(p.Type))
	bytes = append(bytes, p.KeyBytes...)
	return bytes
}

func (p *PublicKey) String() string {
	return hexutil.ToHex(p.ToBytes())
}

func (p *PublicKey) DebugString() string {
	return fmt.Sprintf("pubk%d:%s", p.Type, hexutil.ToHex(p.KeyBytes))
}

type Signature struct {
	Type           CryptoType
	SignatureBytes []byte
}

func (p *Signature) ToBytes() []byte {
	var bytes []byte
	bytes = append(bytes, byte(p.Type))
	bytes = append(bytes, p.SignatureBytes...)
	return bytes
}

func (p *Signature) String() string {
	return hexutil.ToHex(p.ToBytes())
}

func (p *Signature) DebugString() string {
	return fmt.Sprintf("sig%d:%s", p.Type, hexutil.ToHex(p.SignatureBytes))
}

//
//type KyberEd22519PrivKey struct {
//	PrivateKey kyber.Scalar
//	Suit       *edwards25519.SuiteEd25519
//}
//
//func (p *KyberEd22519PrivKey) Decrypt(cipherText []byte) (m []byte, err error) {
//	return ecies.Decrypt(p.Suit, p.PrivateKey, cipherText, p.Suit.Hash)
//}
//
//func (p *PrivateKey) ToKyberEd25519PrivKey() *KyberEd22519PrivKey {
//	var edPrivKey [32]byte
//	var curvPrivKey [64]byte
//	copy(curvPrivKey[:], p.KeyBytes[:64])
//	extra25519.PrivateKeyToCurve25519(&edPrivKey, &curvPrivKey)
//	privateKey, err := edwards25519.UnmarshalBinaryScalar(edPrivKey[:32])
//	suite := edwards25519.NewBlakeSHA256Ed25519()
//	if err != nil {
//		panic(err)
//	}
//	return &KyberEd22519PrivKey{
//		PrivateKey: privateKey,
//		Suit:       suite,
//	}
//}

func (c CryptoType) String() string {
	if c == CryptoTypeEd25519 {
		return "ed25519"
	} else if c == CryptoTypeSecp256k1 {
		return "secp256k1"
	}
	return "unknown"
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
