package miner

import (
	"github.com/annchain/OG/types"
)

type Miner interface{
	StartMine(tx types.Txi, targetMax types.Hash, start uint64, responseChan chan uint64)
	Stop()
}