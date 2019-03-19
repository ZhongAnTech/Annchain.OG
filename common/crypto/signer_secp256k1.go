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
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/annchain/OG/common/crypto/ecies"
	"github.com/annchain/OG/common/crypto/secp256k1"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	ecdsabtcec "github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
)

type SignerSecp256k1 struct {
}

func (s *SignerSecp256k1) GetCryptoType() CryptoType {
	return CryptoTypeSecp256k1
}

func (s *SignerSecp256k1) Sign(privKey PrivateKey, msg []byte) Signature {
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

func (s *SignerSecp256k1) PubKey(privKey PrivateKey) PublicKey {
	_, ecdsapub := ecdsabtcec.PrivKeyFromBytes(ecdsabtcec.S256(), privKey.Bytes)
	pub := FromECDSAPub((*ecdsa.PublicKey)(ecdsapub))
	return PublicKeyFromBytes(CryptoTypeSecp256k1, pub[:])
}

func (s *SignerSecp256k1) AddressFromPubKeyBytes(pubKey []byte) types.Address {
	return s.Address(PublicKeyFromBytes(CryptoTypeSecp256k1, pubKey))
}

func (s *SignerSecp256k1) Verify(pubKey PublicKey, signature Signature, msg []byte) bool {
	signature = s.DealRecoverID(signature)
	sig := signature.Bytes
	return secp256k1.VerifySignature(pubKey.Bytes, Sha256(msg), sig)
}

func (s *SignerSecp256k1) RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey, err error) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], CRandBytes(32))

	privateKey = PrivateKeyFromBytes(CryptoTypeSecp256k1, privKeyBytes[:])
	publicKey = s.PubKey(privateKey)
	return
}

// Address calculate the address from the pubkey
func (s *SignerSecp256k1) Address(pubKey PublicKey) types.Address {
	return types.BytesToAddress(Keccak256((pubKey.Bytes)[1:])[12:])
}

func (s *SignerSecp256k1) Encrypt(p PublicKey, m []byte) (ct []byte, err error) {
	pub, err := UnmarshalPubkey(p.Bytes)
	if err != nil {
		panic(err)
	}
	eciesPub := ecies.ImportECDSAPublic(pub)
	return ecies.Encrypt(rand.Reader, eciesPub, m, nil, nil)
}

func (s *SignerSecp256k1) Decrypt(p PrivateKey, ct []byte) (m []byte, err error) {
	prive, err := ToECDSA(p.Bytes)
	ecisesPriv := ecies.ImportECDSA(prive)
	return ecisesPriv.Decrypt(ct, nil, nil)
}

const sigLength int = 64

func (s *SignerSecp256k1) DealRecoverID(sig Signature) Signature {
	l := len(sig.Bytes)
	if l == sigLength+1 {
		sig.Bytes = sig.Bytes[:l-1]
	}
	return sig
}
