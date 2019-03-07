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
package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/sha3"
)

// Define your own structure and then use messagepack to generate codes
// You can add functions for your struct here.
// RUN "go generate" to generate all helper codes

// DO NOT DELETE THIS TWO COMMENTS. THEY ARE FUNCTIONAL.

//go:generate msgp
//msgp:tuple Foo

type Foo struct {
	Bar     string             `msg:"bar"`
	Baz     float64            `msg:"baz"`
	Address Address            `msg:"address"`
	Parents Hashes             `msg:"parents"`
	KV      map[string]float64 `msg:"kv"`
	Seq     Sequencer          `msg:"seq"`
	TxInner Tx                 `msg:"tx"`
	//BIG *big.Int `msg:"big"`
}

func (f *Foo) CalcHash() (hash Hash, err error) {
	var buf bytes.Buffer

	if err := binary.Write(&buf, binary.LittleEndian, f.Bar); err != nil {
		return Hash{}, err
	}

	hasher := sha3.New512()
	v := hasher.Sum(buf.Bytes())
	hash.MustSetBytes(v, PaddingNone)
	fmt.Println(hash.Hex())
	return
}
