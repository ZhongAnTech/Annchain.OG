package og_interface

import (
	"github.com/annchain/OG/arefactor/utils/marshaller"
)

type FixLengthBytes interface {
	Length() int
	FromBytes(b []byte)
	FromHex(s string) error
	FromHexNoError(s string)
	Bytes() []byte
	Hex() string
	Cmp(FixLengthBytes) int
}

type AddressKey string

type Address interface {
	FixLengthBytes
	AddressKey() AddressKey
	AddressString() string // just for type safety between Address and Hash
	AddressShortString() string
}

type HashKey string

type Hash interface {
	marshaller.IMarshaller
	FixLengthBytes
	HashKey() HashKey
	HashString() string
	HashShortString() string
}
