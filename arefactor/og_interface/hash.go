package og_interface

import (
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
)

const (
	Hash32Length = 32
)

type Hash32 [Hash32Length]byte

func (a *Hash32) HashShortString() string {
	return hexutil.ToHex(a[:10])
}

func (a *Hash32) HashString() string {
	return hexutil.ToHex(a.Bytes())
}

func (a *Hash32) Bytes() []byte {
	return a[:]
}

func (a *Hash32) Length() int {
	return Hash32Length
}

func (a *Hash32) FromBytes(b []byte) {
	copy(a[:], b)
}

func (a *Hash32) FromHex(s string) (err error) {
	bytes, err := hexutil.FromHex(s)
	if err != nil {
		return
	}
	a.FromBytes(bytes)
	return
}

func (a *Hash32) FromHexNoError(s string) {
	err := a.FromHex(s)
	utilfuncs.PanicIfError(err, "HexToHash32")
}
