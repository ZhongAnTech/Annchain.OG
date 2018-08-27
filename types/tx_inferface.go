package types

import (
	"github.com/annchain/OG/common"
)

type TX interface{

	// Hash returns a tx hash
	Hash() common.Hash

}