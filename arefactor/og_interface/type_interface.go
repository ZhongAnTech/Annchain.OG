package og_interface

type FixLengthBytes interface {
	Length() int
	FromBytes(b []byte)
	FromHex(s string) error
	FromHexNoError(s string)
	Bytes() []byte
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
	FixLengthBytes
	HashKey() HashKey
	HashString() string
	HashShortString() string
}
