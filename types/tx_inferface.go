package types

import (
	"github.com/annchain/OG/common"
)

type TX interface{

	// Hash returns a tx hash
	Hash() common.Hash

	// Parents returns the hash of txs that it directly proves.
	Parents() []common.Hash
}