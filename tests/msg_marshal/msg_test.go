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
	"github.com/sirupsen/logrus"
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

type fooInt struct {
	a    int
	name string
}

func (f fooInt) String() string {
	fmt.Println("f1", f.a, f.name)
	return fmt.Sprintf("a %d name %s", f.a, f.name)

}

type fooHash struct {
	b   int
	c   float64
	foo *fooInt
}

func (f fooHash) String() string {
	fmt.Println("f2", f.b, f.foo)
	return fmt.Sprintf("f1 %s, b %d, c %f", f.foo, f.b, f.c)

}

//TestLogrus
func TestLogrus(t *testing.T) {
	var f1 fooInt
	f1.a = 100
	f1.name = " alice"
	f2 := fooHash{
		foo: &fooInt{
			a:    56,
			name: "bob",
		},
		b: 95,
		c: 4.56,
	}
	logrus.SetLevel(logrus.WarnLevel)
	logrus.Info(f1)
	logrus.Info(f2.String())
}
