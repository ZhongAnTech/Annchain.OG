package msg

import  "github.com/tinylib/msgp/msgp"

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

type Bytes []byte
type Strings []string

type Uints []uint

type Messages []Message


//add new basic type here