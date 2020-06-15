package og_interface

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
	FixLengthBytes
	HashString() string
	HashShortString() string
}
