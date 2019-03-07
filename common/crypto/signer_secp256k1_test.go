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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignerSecp(t *testing.T) {
	signer := SignerSecp256k1{}

	pub, priv, err := signer.RandomKeyPair()
	assert.NoError(t, err)

	fmt.Println(hex.Dump(pub.Bytes))
	fmt.Println(hex.Dump(priv.Bytes))
	address := signer.Address(pub)
	fmt.Println(hex.Dump(address.Bytes[:]))
	fmt.Println(signer.Address(pub).Hex())

	fmt.Printf("%x\n", priv.Bytes[:])
	fmt.Printf("%x\n", pub.Bytes[:])
	fmt.Printf("%x\n", address.Bytes[:])

	pub2 := signer.PubKey(priv)
	fmt.Println(hex.Dump(pub2.Bytes))
	assert.True(t, bytes.Equal(pub.Bytes, pub2.Bytes))

	content := []byte("This is a test")
	sig := signer.Sign(priv, content)
	fmt.Println(hex.Dump(sig.Bytes))

	assert.True(t, signer.Verify(pub2, sig, content))

	content[0] = 0x88
	assert.False(t, signer.Verify(pub2, sig, content))

}

func TestSignerNewPrivKey(t *testing.T) {
	t.Parallel()

	signer := SignerSecp256k1{}
	pk, priv, _ := signer.RandomKeyPair()

	b := []byte("foo")
	sig := signer.Sign(priv, b)
	if !signer.Verify(pk, sig, b) {
		t.Fatalf("vertfy failed")
	}

}
