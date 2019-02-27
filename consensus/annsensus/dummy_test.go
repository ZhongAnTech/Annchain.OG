package annsensus

import (
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"math/rand"
)

type DummyDag struct {
}

func (d *DummyDag) GetTx(hash types.Hash) types.Txi {
	return nil
}

func (d *DummyDag) GetTxByNonce(addr types.Address, nonce uint64) types.Txi {
	return nil
}

func (d *DummyDag) GetSequencerByHeight(id uint64) *types.Sequencer {
	return &types.Sequencer{
		TxBase: types.TxBase{Height: id},
	}
}

func (d *DummyDag) GetTxsByNumber(id uint64) types.Txs {
	return nil
}

func (d *DummyDag) LatestSequencer() *types.Sequencer {
	return &types.Sequencer{
		TxBase: types.TxBase{Height: rand.Uint64()},
	}
}

func (d *DummyDag) GetSequencer(hash types.Hash, id uint64) *types.Sequencer {
	return &types.Sequencer{
		TxBase: types.TxBase{Height: id,
			Hash: hash},
	}
}

func (d *DummyDag) Genesis() *types.Sequencer {
	return &types.Sequencer{
		TxBase: types.TxBase{Height: 0},
	}
}

func (d *DummyDag) GetSequencerByHash(hash types.Hash) *types.Sequencer {
	return nil
}

func (d *DummyDag) GetBalance(addr types.Address) *math.BigInt {
	return math.NewBigInt(100000)
}
