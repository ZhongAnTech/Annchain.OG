package pool

import (
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/ledger"
)

type ILedger interface {
	GetTx(hash types2.Hash) types.Txi
	GetTxByNonce(addr common.Address, nonce uint64) types.Txi
	GetSequencerByHeight(id uint64) *types.Sequencer
	GetTxisByNumber(id uint64) types.Txis
	LatestSequencer() *types.Sequencer
	GetSequencer(hash types2.Hash, id uint64) *types.Sequencer
	Genesis() *types.Sequencer
	GetHeight() uint64
	GetSequencerByHash(hash types2.Hash) *types.Sequencer
	GetBalance(addr common.Address, tokenID int32) *math.BigInt
	GetLatestNonce(addr common.Address) (uint64, error)

	IsTxExists(hash types2.Hash) bool
	IsAddressExists(addr common.Address) bool

	Push(batch *ledger.ConfirmBatch) error
}

type PoolHashLocator interface {
	IsLocalHash(hash types2.Hash) bool
	Get(hash types2.Hash) types.Txi
}

type LedgerHashLocator interface {
	IsLocalHash(hash types2.Hash) bool
	GetTx(hash types2.Hash) types.Txi
}

type LocalGraphInfoProvider interface {
	GetMaxWeight() uint64
}
