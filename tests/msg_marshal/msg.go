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
	"encoding/binary"
	"fmt"
	"github.com/tinylib/msgp/msgp"
)

//go:generate msgp

type FooI interface {
	String() string
	// implemented by msgp
	DecodeMsg(dc *msgp.Reader) (err error)
	EncodeMsg(en *msgp.Writer) (err error)
	MarshalMsg(b []byte) (o []byte, err error)
	UnmarshalMsg(bts []byte) (o []byte, err error)
	Msgsize() (s int)
	GetType() uint16
	GetName() string
}

type Person struct {
	Name string
	Age  int
	Type uint16
}

type Student struct {
	Person
	Score int
}

type Teacher struct {
	Person
	Teach bool
}

func (p *Person) GetName() string {
	return p.Name
}

func (p *Person) GetType() uint16 {
	return p.Type
}

func (p Person) String() string {
	return p.Name + " person"
}

func (s Student) String() string {
	return s.Person.String() + " student"
}

func (s Teacher) String() string {
	return s.Person.String() + " teacher"
}

func MashalFoo(f FooI, b []byte) ([]byte, error) {
	if f == nil {
		panic("nil foo")
	}
	tail := make([]byte, 2)
	binary.BigEndian.PutUint16(tail, f.GetType())
	b = append(b, tail...)
	o, err := f.MarshalMsg(b)
	return o, err
}

func UnmarShalFoo(b []byte) (o []byte, f FooI, err error) {
	if len(b) < 3 {
		return b, nil, fmt.Errorf("size mismatch")
	}
	tp := binary.BigEndian.Uint16(b)
	switch tp {
	case 1:
		var s Student
		o, err := s.UnmarshalMsg(b[2:])
		if err != nil {
			return o, nil, err
		}
		return o, &s, nil
	case 2:
		var s Teacher
		o, err := s.UnmarshalMsg(b[2:])
		if err != nil {
			return o, nil, err
		}
		return o, &s, nil
	default:
		return b, nil, fmt.Errorf("unkown type")
	}
}
