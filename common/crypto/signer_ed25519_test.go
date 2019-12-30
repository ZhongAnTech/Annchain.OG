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
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSigner(t *testing.T) {
	signer := SignerEd25519{}

	pub, priv := signer.RandomKeyPair()

	fmt.Println(hex.Dump(pub.KeyBytes))

	fmt.Println(hex.Dump(priv.KeyBytes))
	address := signer.Address(pub)
	fmt.Println(hex.Dump(address.Bytes[:]))
	fmt.Println(signer.Address(pub).Hex())

	fmt.Printf("%x\n", priv.KeyBytes[:])
	fmt.Printf("%x\n", pub.KeyBytes[:])
	fmt.Printf("%x\n", address.Bytes[:])

	pub2 := signer.PubKey(priv)
	fmt.Println(hex.Dump(pub2.KeyBytes))
	assert.True(t, bytes.Equal(pub.KeyBytes, pub2.KeyBytes))

	content := []byte("This is a test")
	sig := signer.Sign(priv, content)
	fmt.Println(hex.Dump(sig.SignatureBytes))

	assert.True(t, signer.Verify(pub2, sig, content))

	content[0] = 0x88
	assert.False(t, signer.Verify(pub2, sig, content))

}

func TestSignerEd25519_Sign(t *testing.T) {
	data := common.FromHex("0x000000000000000099fa25cbb3d8fdd1ec29aa499cf651972f0ee67b0000000000000001")
	signer := &SignerEd25519{}
	//privKey,_:= PrivateKeyFromString("0x009d9d0fe5e9ef0bb3bb4934db878688500fd0fd8e026c1ff1249b7e268c8a363aa7d45d13a5accb299dc7fe0f3b5fb0e9526b67008f7ead02c51c7b1f5a1d7b00")
	_, privKey := signer.RandomKeyPair()
	sig := signer.Sign(privKey, data)
	fmt.Printf("%x\n", sig.SignatureBytes[:])
	ok := signer.Verify(signer.PubKey(privKey), sig, data)
	if !ok {
		t.Fatal(ok)
	}
}
