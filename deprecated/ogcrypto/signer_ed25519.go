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
package ogcrypto

import (
	"fmt"
	"github.com/annchain/OG/deprecated"
	"github.com/annchain/OG/deprecated/ogcrypto/extra25519"
	"github.com/annchain/OG/deprecated/ogcrypto_interface"
	"github.com/annchain/kyber/v3/encrypt/ecies"
	"github.com/annchain/kyber/v3/group/edwards25519"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ed25519"
	"strconv"
)

type SignerEd25519 struct {
}

func (s *SignerEd25519) GetCryptoType() ogcrypto_interface.CryptoType {
	return ogcrypto_interface.CryptoTypeEd25519
}

func (s *SignerEd25519) CanRecoverPubFromSig() bool {
	return false
}

func (s *SignerEd25519) Sign(privKey ogcrypto_interface.PrivateKey, msg []byte) ogcrypto_interface.Signature {
	signatureBytes := ed25519.Sign(privKey.KeyBytes, msg)
	return deprecated.SignatureFromBytes(ogcrypto_interface.CryptoTypeEd25519, signatureBytes)
}

func (s *SignerEd25519) PubKey(privKey ogcrypto_interface.PrivateKey) ogcrypto_interface.PublicKey {
	pubkey := ed25519.PrivateKey(privKey.KeyBytes).Public()
	return deprecated.PublicKeyFromBytes(ogcrypto_interface.CryptoTypeEd25519, []byte(pubkey.(ed25519.PublicKey)))
}

func (s *SignerEd25519) PublicKeyFromBytes(b []byte) ogcrypto_interface.PublicKey {
	return deprecated.PublicKeyFromBytes(s.GetCryptoType(), b)
}

func (s *SignerEd25519) Verify(pubKey ogcrypto_interface.PublicKey, signature ogcrypto_interface.Signature, msg []byte) bool {
	//validate to prevent panic
	if l := len(pubKey.KeyBytes); l != ed25519.PublicKeySize {
		err := fmt.Errorf("ed25519: bad public key length: " + strconv.Itoa(l))
		logrus.WithError(err).Warn("verify fail")
		return false
	}
	return ed25519.Verify(pubKey.KeyBytes, msg, signature.SignatureBytes)
}

func (s *SignerEd25519) RandomKeyPair() (publicKey ogcrypto_interface.PublicKey, privateKey ogcrypto_interface.PrivateKey) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	publicKey = deprecated.PublicKeyFromBytes(ogcrypto_interface.CryptoTypeEd25519, public)
	privateKey = deprecated.PrivateKeyFromBytes(ogcrypto_interface.CryptoTypeEd25519, private)
	return
}

func (s *SignerEd25519) Encrypt(publicKey ogcrypto_interface.PublicKey, m []byte) (ct []byte, err error) {
	//convert our pubkey key to kyber pubkey
	suite := edwards25519.NewBlakeSHA256Ed25519()
	pubKey, err := edwards25519.UnmarshalBinaryPoint(publicKey.KeyBytes)
	if err != nil {
		return nil, err
	}
	return ecies.Encrypt(suite, pubKey, m, suite.Hash)
}

func (s *SignerEd25519) Decrypt(p ogcrypto_interface.PrivateKey, ct []byte) (m []byte, err error) {
	//convert our priv key to kyber privkey
	var edPrivKey [32]byte
	var curvPrivKey [64]byte
	copy(curvPrivKey[:], p.KeyBytes[:64])
	extra25519.PrivateKeyToCurve25519(&edPrivKey, &curvPrivKey)
	privateKey, err := edwards25519.UnmarshalBinaryScalar(edPrivKey[:32])
	if err != nil {
		panic(err)
	}
	suite := edwards25519.NewBlakeSHA256Ed25519()
	return ecies.Decrypt(suite, privateKey, ct, suite.Hash)
}
