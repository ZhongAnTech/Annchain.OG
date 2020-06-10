package types

import (
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
)

type Address []byte

const (
	AddressLength = 20
)

// HexToHash sets byte representation of s to Hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToAddress(s string) (address Address, err error) {
	bytes, err := hexutil.FromHex(s)
	if err != nil {
		return
	}
	address = bytes
	return
}

func HexToAddressNoError(s string) (address Address) {
	address, err := HexToAddress(s)
	utilfuncs.PanicIfError(err, "HexToAddress")
	return
}

func BytesToAddress(b []byte) Address {
	nb := make([]byte, AddressLength)
	copy(nb, b)
	return nb
}
