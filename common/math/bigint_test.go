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
package math

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMoney(t *testing.T) {
	s := "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	a, success := NewBigIntFromString(s, 10)
	if !success {
		panic(s)
	}

	a.SetString(s, 10)
	a2 := NewBigInt(2)
	a3 := NewBigIntFromBigInt(a.Value.Add(a.Value, a2.Value))

	fmt.Println(a)
	fmt.Println(a2)
	fmt.Println(a3)

	a3 = NewBigIntFromBigInt(a.Value.Add(a.Value, a2.Value))

	fmt.Println(a3)
}

func TestSerialization(t *testing.T) {
	s := "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	a, success := NewBigIntFromString(s, 10)
	if !success {
		panic(s)
	}
	bytes, err := a.MarshalMsg(nil)
	assert.NoError(t, err)
	hex.Dump(bytes)
	b := &BigInt{}
	bytes, err = b.UnmarshalMsg(bytes)
	assert.NoError(t, err)
	fmt.Println(b.String())
	assert.Equal(t, s, b.String())
}

func TestBigInt_Add(t *testing.T) {
	bi := NewBigInt(1000)
	fmt.Println(bi)
	bi = bi.Add(NewBigInt(99))
	fmt.Println(bi)
}
