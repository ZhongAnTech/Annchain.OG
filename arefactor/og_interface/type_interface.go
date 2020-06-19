package og_interface

type FixLengthBytes interface {
	Length() int
	FromBytes(b []byte)
	FromHex(s string) error
	FromHexNoError(s string)
	Bytes() []byte
}

type Address interface {
	FixLengthBytes
	AddressKey() string
	AddressString() string // just for type safety between Address and Hash
	AddressShortString() string
}

type Hash interface {
	FixLengthBytes
	HashKey() string
	HashString() string
	HashShortString() string
}
