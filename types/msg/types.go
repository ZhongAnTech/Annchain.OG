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
package msg

import (
	"github.com/tinylib/msgp/msgp"
)

//go:generate msgp

type Message interface {
	msgp.MarshalSizer
	msgp.Unmarshaler
	msgp.Encodable
	msgp.Decodable
}

type Bool bool

type Uint8 uint8

type Uint16 uint16

type Uint32 uint32

type Uint64 uint64

type Int8 int8

type Int16 int16

type Int32 int32

type Int64 int64

type Float32 float32

type Float64 float64

type String string

type Int int

type Uint uint

// integer values.
type Byte byte

//msgp:tuple Bytes
type Bytes []byte

type Strings []string

type Uints []uint

type Messages []Message

//add new basic type here
