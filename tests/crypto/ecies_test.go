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
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/poc/extra25519"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"testing"
)

type Stream struct {
}

func (t *Stream) XORKeyStream(dst, src []byte) {
	copy(dst, src)
	return
}

func TestEcies(t *testing.T) {
	s := crypto.NewSigner(crypto.CryptoTypeEd25519)
	b, err := hex.DecodeString("fbe35f7844e2434cec0f1ea3ec97080d52a1d10394f9c19d8b6e38a92bd0072f6fbf9980134f18a5f864b4e763ffbc78dbdee3ced30334c3efd8b8b3d7b363d2")
	if err != nil {
		panic(err)
	}
	priv := crypto.PrivateKeyFromBytes(s.GetCryptoType(), b)
	pub := priv.PublicKey()
	//*pub, priv, _ = s.RandomKeyPair()
	fmt.Println("pub ", pub.String())
	fmt.Println("priv", priv.String())

	var curvPriv [32]byte
	var edPriv [64]byte
	copy(edPriv[:], priv.Bytes[:32])
	extra25519.PrivateKeyToCurve25519(&curvPriv, &edPriv)
	fmt.Println("edPriv ", hex.EncodeToString(edPriv[:]))
	fmt.Println("curvPriv ", hex.EncodeToString(curvPriv[:]))

	//suite := edwards25519.NewBlakeSHA256Ed25519WithRand(&Stream{})
	suite := edwards25519.NewBlakeSHA256Ed25519()
	//private := suite.Scalar().Pick(random.New())
	//public := suite.Point().Mul(private, nil)
	//fmt.Println("priv", private)
	//fmt.Println("pub", public)
	sc, err := edwards25519.UnmarshalBinaryScalar(curvPriv[:32])
	if err != nil {
		panic(err)
	}
	pc := suite.Point().Mul(sc, nil)
	pk, _ := edwards25519.UnmarshalBinaryPoint(pub.Bytes)
	fmt.Println("kyber  prive  key ", sc)
	fmt.Println("kyber pub key ", pc, pk)
	var edPubkey [32]byte
	var curvPubkey [32]byte
	copy(edPubkey[:], pub.Bytes[:])
	extra25519.PublicKeyToCurve25519(&curvPubkey, &edPubkey)
	fmt.Println("ed pubkey ", hex.EncodeToString(edPubkey[:]))
	fmt.Println(" cur pubkey", hex.EncodeToString(curvPubkey[:]))
}
