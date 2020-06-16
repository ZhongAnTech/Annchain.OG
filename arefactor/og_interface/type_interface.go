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

type Address interface {
	FixLengthBytes
	AddressString() string // just for type safety between Address and Hash
	AddressShortString() string
}

type Hash interface {
	marshaller.IMarshaller
	FixLengthBytes
	HashString() string
	HashShortString() string
}
