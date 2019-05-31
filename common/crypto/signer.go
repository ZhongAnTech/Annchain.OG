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

import "github.com/annchain/OG/types"

type ISigner interface {
	GetCryptoType() CryptoType
	Sign(privKey PrivateKey, msg []byte) Signature
	PubKey(privKey PrivateKey) PublicKey
	Verify(pubKey PublicKey, signature Signature, msg []byte) bool
	RandomKeyPair() (publicKey PublicKey, privateKey PrivateKey)
	Address(pubKey PublicKey) types.Address
	AddressFromPubKeyBytes(pubKey []byte) types.Address
	Encrypt(publicKey PublicKey, m []byte) (ct []byte, err error)
	Decrypt(p PrivateKey, ct []byte) (m []byte, err error)
	PublicKeyFromBytes(b []byte) PublicKey
}

//set this value when you code run
var Signer ISigner

func init() {
	//default value
	Signer = NewSigner(CryptoTypeSecp256k1)
}
