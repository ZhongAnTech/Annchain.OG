package interfaces

import (
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/model"
)

type OgStatusProvider interface {
	GetCurrentOgStatus() model.OgStatusData
}


type LedgerTxProvider interface {
	GetHeightTxs(height uint64, offset int, limit int) []types.Txi
	GetTxis(hashes types2.Hashes) types.Txis
}

type Syncer interface {
	//Enqueue(hash *common.Hash, childHash common.Hash, sendBloomFilter bool)
	//SyncHashList(seqHash common.Hash)
	ClearQueue()
	//IsCachedHash(hash common.Hash) bool
}

type Hasher interface {
	CalcHash(tx types.Txi) (hash types2.Hash)
}

type Miner interface {
	Mine(tx types.Txi, targetMax types2.Hash, start uint64, responseChan chan uint64) bool
}
