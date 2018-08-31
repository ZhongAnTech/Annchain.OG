package core

import (
	"github.com/annchain/OG/types"
)



var (
	prefixTxLookUp = []byte("tlu")
)

func txLookUpKey(hash types.Hash) []byte {
	return append(prefixTxLookUp, hash.ToBytes()...)
}





