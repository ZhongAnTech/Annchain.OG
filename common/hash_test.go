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
package common

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
	"testing"
)

func TestHash(t *testing.T) {

	var emHash types.Hash
	var nHash types.Hash
	nHash = types.HexToHash("0xc770f1dccb00c0b845d36d3baee2590defee2d6894f853eb63a60270612271a3")
	mHash := types.HexToHash("0xc770f1dccb00c0b845d36d3baee2590defee2d6894f853eb63a60270612271a3")
	if !emHash.Empty() {
		t.Fatalf("fail")
	}
	if nHash.Empty() {
		t.Fatalf("fail")
	}
	hashes := types.Hashes{nHash, emHash}
	fmt.Println(hashes.String())
	pHash := &nHash
	p2hash := &mHash
	if nHash != mHash {
		t.Fatal("should equal")
	}
	if pHash == p2hash {
		t.Fatal("should not  equal")
	}
}

func TestHexToHash(t *testing.T) {
	h := types.randomHash()
	d, err := json.Marshal(&h)
	fmt.Println(string(d), err)
}

func TestHash_Empty(t *testing.T) {
	var h types.Hash
	fmt.Println(h)
	if h.Empty() {
		fmt.Println("empty")
	} else {
		t.Fatal("should be empty")
	}
}
