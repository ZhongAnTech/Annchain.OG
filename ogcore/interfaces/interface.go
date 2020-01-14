package interfaces

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/model"
)

type OgStatusProvider interface {
	GetCurrentOgStatus() model.OgStatusData
}


type LedgerTxProvider interface {
	GetHeightTxs(height uint64, offset uint32, limit uint32) []types.Txi
}

type Syncer interface {
	Enqueue(hash *common.Hash, childHash common.Hash, sendBloomFilter bool)
	SyncHashList(seqHash common.Hash)
	ClearQueue()
	IsCachedHash(hash common.Hash) bool
}

type Hasher interface {
	CalcHash(tx types.Txi) (hash common.Hash)
}

type Miner interface {
	Mine(tx types.Txi, targetMax common.Hash, start uint64, responseChan chan uint64) bool
}
