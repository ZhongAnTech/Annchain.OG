package pool

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/types"
)

type ILedger interface {
	GetTx(hash common.Hash) types.Txi
	GetTxByNonce(addr common.Address, nonce uint64) types.Txi
	GetSequencerByHeight(id uint64) *types.Sequencer
	GetTxisByNumber(id uint64) types.Txis
	LatestSequencer() *types.Sequencer
	GetSequencer(hash common.Hash, id uint64) *types.Sequencer
	Genesis() *types.Sequencer
	GetHeight() uint64
	GetSequencerByHash(hash common.Hash) *types.Sequencer
	GetBalance(addr common.Address, tokenID int32) *math.BigInt
	GetLatestNonce(addr common.Address) (uint64, error)

	IsTxExists(hash common.Hash) bool
	IsAddressExists(addr common.Address) bool
}
