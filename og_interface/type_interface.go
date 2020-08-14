package og_interface

import (
	"github.com/annchain/commongo/hexutil"
	"github.com/annchain/commongo/marshaller"
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
	AddressKey() AddressKey
	AddressString() string // just for type safety between Address and Hash
	AddressShortString() string

	marshaller.IMarshaller
}

type AddressKey string

func (k AddressKey) Bytes() []byte {
	b, _ := hexutil.FromHex(string(k))
	return b
}

type Hash interface {
	FixLengthBytes
	HashKey() HashKey
	HashString() string
	HashShortString() string

	marshaller.IMarshaller
}

type HashKey string

func (k HashKey) Bytes() []byte {
	b, _ := hexutil.FromHex(string(k))
	return b
}
