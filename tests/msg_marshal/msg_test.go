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
package msg_marshal

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestPerson_MarshalMsg(t *testing.T) {
	p := Person{
		Name: "alice",
		Age:  10,
		Type: 1,
	}
	p2 := Person{
		Name: "bob",
		Age:  1522,
		Type: 2,
	}
	s1 := Student{p, 15}
	t1 := Teacher{p2, true}
	var fooi FooI
	fooi = &s1
	data, _ := MashalFoo(fooi, nil)
	fmt.Println(hex.EncodeToString(data))
	_, s2, err := UnmarShalFoo(data)
	fmt.Println(s2, err)
	if s1 != *s2.(*Student) {
		t.Fail()
	}
	_ = t1
}
