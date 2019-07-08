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
	"bytes"
	"fmt"
	"github.com/annchain/OG/poc/extra25519"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"go.dedis.ch/kyber/v3/encrypt/ecies"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ripemd160"
	"strconv"
)

type SignerEd25519 struct {
}

func (s *SignerEd25519) GetCryptoType() CryptoType {
	return CryptoTypeEd25519
}

func (s *SignerEd25519) CanRecoverPubFromSig() bool {
	return false
}

func (s *SignerEd25519) Sign(privKey PrivateKey, msg []byte) Signature {
	signatureBytes := ed25519.Sign(privKey.Bytes, msg)
	return SignatureFromBytes(CryptoTypeEd25519, signatureBytes)
}

func (s *SignerEd25519) PubKey(privKey PrivateKey) PublicKey {
	pubkey := ed25519.PrivateKey(privKey.Bytes).Public()
	return PublicKeyFromBytes(CryptoTypeEd25519, []byte(pubkey.(ed25519.PublicKey)))
}

func (s *SignerEd25519) AddressFromPubKeyBytes(pubKey []byte) types.Address {
	return s.Address(PublicKeyFromBytes(CryptoTypeEd25519, pubKey))
}

func (s *SignerEd25519) PublicKeyFromBytes(b []byte) PublicKey {
	return PublicKeyFromBytes(s.GetCryptoType(), b)
}

func (s *SignerEd25519) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	//validate to prevent panic
	if l := len(pubKey.Bytes); l != ed25519.PublicKeySize {
		err := fmt.Errorf("ed25519: bad public key length: " + strconv.Itoa(l))
		logrus.WithError(err).Warn("verify fail")
		return false
	}
	return ed25519.Verify(pubKey.Bytes, msg, signature.Bytes)
}

func (s *SignerEd25519) RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	publicKey = PublicKeyFromBytes(CryptoTypeEd25519, public)
	privateKey = PrivateKeyFromBytes(CryptoTypeEd25519, private)
	return
}

// Address calculate the address from the pubkey
func (s *SignerEd25519) Address(pubKey PublicKey) types.Address {
	var w bytes.Buffer
	w.Write([]byte{byte(pubKey.Type)})
	w.Write(pubKey.Bytes)
	hasher := ripemd160.New()
	hasher.Write(w.Bytes())
	result := hasher.Sum(nil)
	return types.BytesToAddress(result)
}

func (s *SignerEd25519) Encrypt(publicKey PublicKey, m []byte) (ct []byte, err error) {
	//convert our pubkey key to kyber pubkey
	suite := edwards25519.NewBlakeSHA256Ed25519()
	pubKey, err := edwards25519.UnmarshalBinaryPoint(publicKey.Bytes)
	if err != nil {
		return nil, err
	}
	return ecies.Encrypt(suite, pubKey, m, suite.Hash)
}

func (s *SignerEd25519) Decrypt(p PrivateKey, ct []byte) (m []byte, err error) {
	//convert our priv key to kyber privkey
	var edPrivKey [32]byte
	var curvPrivKey [64]byte
	copy(curvPrivKey[:], p.Bytes[:64])
	extra25519.PrivateKeyToCurve25519(&edPrivKey, &curvPrivKey)
	privateKey, err := edwards25519.UnmarshalBinaryScalar(edPrivKey[:32])
	if err != nil {
		panic(err)
	}
	suite := edwards25519.NewBlakeSHA256Ed25519()
	return ecies.Decrypt(suite, privateKey, ct, suite.Hash)
}
