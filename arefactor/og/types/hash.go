package types

import (
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
)

type Hash []byte

// HexToHash sets byte representation of s to Hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) (hash Hash, err error) {
	bytes, err := hexutil.FromHex(s)
	if err != nil {
		return
	}
	hash = bytes
	return
}

func HexToHashNoError(s string) (hash Hash) {
	hash, err := HexToHash(s)
	utilfuncs.PanicIfError(err, "hexToHash")
	return
}
