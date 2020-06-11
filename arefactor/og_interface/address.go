package og_interface

import (
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
)

const (
	Address20Length = 20
)

type Address20 [Address20Length]byte

func (a *Address20) AddressShortString() string {
	return hexutil.ToHex(a[:10])
}

func (a *Address20) AddressString() string {
	return hexutil.ToHex(a.Bytes())
}

func (a *Address20) Bytes() []byte {
	return a[:]
}

func (a *Address20) Length() int {
	return Address20Length
}

func (a *Address20) FromBytes(b []byte) {
	copy(a[:], b)
}

func (a *Address20) FromHex(s string) (err error) {
	bytes, err := hexutil.FromHex(s)
	if err != nil {
		return
	}
	a.FromBytes(bytes)
	return
}

func (a *Address20) FromHexNoError(s string) {
	err := a.FromHex(s)
	utilfuncs.PanicIfError(err, "HexToAddress20")
}
