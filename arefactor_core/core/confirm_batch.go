package core

import (
	"fmt"
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/types"
	"github.com/annchain/OG/arefactor_core/core/state"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogdb"
)

type cachedConfirms struct {
	ledger Ledger

	fronts  []*confirmBatch
	batches map[ogTypes.HashKey]*confirmBatch
}

func newCachedConfirm(ledger Ledger) *cachedConfirms {
	return &cachedConfirms{
		ledger:  ledger,
		fronts:  make([]*confirmBatch, 0),
		batches: make(map[ogTypes.HashKey]*confirmBatch),
	}
}

func (c *cachedConfirms) getConfirmBatch(seqHash ogTypes.Hash) *confirmBatch {
	return c.batches[seqHash.HashKey()]
}

func (c *cachedConfirms) existTx(seqHash ogTypes.Hash, txHash ogTypes.Hash) bool {
	batch := c.batches[seqHash.HashKey()]
	if batch != nil {
		return batch.existTx(txHash)
	}
	return c.ledger.GetTx(txHash) != nil
}

func (c *cachedConfirms) preConfirm(batch *confirmBatch) {
	c.batches[batch.seq.GetTxHash().HashKey()] = batch
}

func (c *cachedConfirms) confirm(batch *confirmBatch) {
	// delete conflicts batches
	for _, batchToDelete := range c.fronts {
		//batchToDelete := c.fronts[i]
		if batchToDelete.isSame(batch) {
			continue
		}

		c.traverseFromRoot(batchToDelete, func(b *confirmBatch) {
			delete(c.batches, b.seq.GetTxHash().HashKey())
		})
	}
	c.fronts = batch.children
}

// traverseFromRoot traverse the cached confirmBatch trees and process the function "f" for
// every found confirm batches.
func (c *cachedConfirms) traverseFromRoot(root *confirmBatch, f func(b *confirmBatch)) {
	seekingPool := make([]*confirmBatch, 0)
	seekingPool = append(seekingPool, root)

	seeked := make(map[ogTypes.HashKey]struct{})
	for len(seekingPool) > 0 {
		batch := seekingPool[0]
		seekingPool = seekingPool[1:]

		f(batch)
		for _, newBatch := range batch.children {
			if _, alreadySeeked := seeked[newBatch.seq.GetTxHash().HashKey()]; alreadySeeked {
				continue
			}
			seekingPool = append(seekingPool, newBatch)
		}
		seeked[batch.seq.GetTxHash().HashKey()] = struct{}{}
	}
}

func (c *cachedConfirms) traverseFromLeaf(leaf *confirmBatch, f func(b *confirmBatch)) {
	seekingPool := make([]*confirmBatch, 0)
	seekingPool = append(seekingPool, leaf)

	for len(seekingPool) > 0 {
		batch := seekingPool[0]
		seekingPool = seekingPool[1:]

		f(batch)
		seekingPool = append(seekingPool, batch.parent)
	}
}

type confirmBatch struct {
	//tempLedger Ledger
	ledger Ledger

	parent   *confirmBatch
	children []*confirmBatch

	seq            *types.Sequencer
	elders         []types.Txi
	eldersQueryMap map[ogTypes.HashKey]types.Txi
	details        *state.StateDB

	//details        map[ogTypes.AddressKey]*batchDetail
}

//var emptyConfirmedBatch = &confirmBatch{}

func newConfirmBatch(ledger Ledger, seq *types.Sequencer, db ogdb.Database) (*confirmBatch, error) {
	stateDB, err := state.NewStateDB(state.DefaultStateDBConfig(), state.NewDatabase(db), seq.StateRoot)
	if err != nil {
		return nil, err
	}
	c := &confirmBatch{
		ledger:         ledger,
		parent:         nil,
		children:       make([]*confirmBatch, 0),
		seq:            seq,
		elders:         make([]types.Txi, 0),
		eldersQueryMap: make(map[ogTypes.HashKey]types.Txi),
		details:        stateDB,
	}
	return c, nil
}

func (c *confirmBatch) construct(elders map[ogTypes.HashKey]types.Txi) error {
	for _, txi := range elders {
		// return error if a sequencer confirm a tx that has same nonce as itself.
		if txi.Sender() == c.seq.Sender() && txi.GetNonce() == c.seq.GetNonce() {
			return fmt.Errorf("seq's nonce is the same as a tx it confirmed, nonce: %d, tx hash: %s",
				c.seq.GetNonce(), txi.GetTxHash())
		}

		switch tx := txi.(type) {
		case *types.Sequencer:
			break
		case *types.Tx:
			c.processTx(tx)
		default:
			c.addTx(tx)
		}
	}
	return nil
}

//func (c *confirmBatch) isValid() error {
//	// verify balance and nonce
//	for _, batchDetail := range c.getDetails() {
//		err := batchDetail.isValid()
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

func (c *confirmBatch) isSame(cb *confirmBatch) bool {
	return c.seq.GetTxHash().Cmp(cb.seq.GetTxHash()) == 0
}

//func (c *confirmBatch) getOrCreateDetail(addr ogTypes.Address) *batchDetail {
//	detailCost := c.getDetail(addr)
//	if detailCost == nil {
//		detailCost = c.createDetail(addr)
//	}
//	return detailCost
//}
//
//func (c *confirmBatch) getDetail(addr ogTypes.Address) *batchDetail {
//	return c.details[addr.AddressKey()]
//}
//
//func (c *confirmBatch) getDetails() map[ogTypes.AddressKey]*batchDetail {
//	return c.details
//}
//
//func (c *confirmBatch) createDetail(addr ogTypes.Address) *batchDetail {
//	c.details[addr.AddressKey()] = newBatchDetail(addr, c)
//	return c.details[addr.AddressKey()]
//}
//
//func (c *confirmBatch) setDetail(detail *batchDetail) {
//	c.details[detail.address.AddressKey()] = detail
//}

func (c *confirmBatch) processTx(txi types.Txi) {
	if txi.GetType() != types.TxBaseTypeNormal {
		return
	}
	tx := txi.(*types.Tx)

	detailCost := c.getOrCreateDetail(tx.From)
	detailCost.addCost(tx.TokenId, tx.Value)
	detailCost.addTx(tx)
	c.setDetail(detailCost)

	detailEarn := c.getOrCreateDetail(tx.To)
	detailEarn.addEarn(tx.TokenId, tx.Value)
	c.setDetail(detailEarn)

	c.addTxToElders(tx)
}

// addTx adds tx only without any processing.
func (c *confirmBatch) addTx(tx types.Txi) {
	//detailSender := c.getOrCreateDetail(tx.Sender())
	//detailSender.addTx(tx)

	c.addTxToElders(tx)
}

func (c *confirmBatch) addTxToElders(tx types.Txi) {
	c.elders = append(c.elders, tx)
	c.eldersQueryMap[tx.GetTxHash().HashKey()] = tx
}

func (c *confirmBatch) existCurrentTx(hash ogTypes.Hash) bool {
	return c.eldersQueryMap[hash.HashKey()] != nil
}

// existTx checks if input tx exists in current confirm batch. If not
// exists, then check parents confirm batch and check DAG ledger at last.
func (c *confirmBatch) existTx(hash ogTypes.Hash) bool {
	exists := c.existCurrentTx(hash)
	if exists {
		return exists
	}
	if c.parent != nil {
		return c.parent.existTx(hash)
	}
	return c.ledger.GetTx(hash) != nil
}

func (c *confirmBatch) existSeq(seqHash ogTypes.Hash) bool {
	if c.seq.GetTxHash().Cmp(seqHash) == 0 {
		return true
	}
	if c.parent != nil {
		return c.parent.existSeq(seqHash)
	}
	return c.ledger.GetTx(seqHash) != nil
}

//func (c *confirmBatch) getCurrentBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
//	detail := c.getDetail(addr)
//	if detail == nil {
//		return nil
//	}
//	return detail.getBalance(tokenID)
//}

// getBalance get balance from its own StateDB
func (c *confirmBatch) getBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
	return c.details.GetBalance(addr)

	//blc := c.getCurrentBalance(addr, tokenID)
	//if blc != nil {
	//	return blc
	//}
	//return c.getConfirmedBalance(addr, tokenID)
}

//func (c *confirmBatch) getConfirmedBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
//	if c.parent != nil {
//		return c.parent.getBalance(addr, tokenID)
//	}
//	return c.ledger.GetBalance(addr, tokenID)
//}

//func (c *confirmBatch) getCurrentLatestNonce(addr ogTypes.Address) (uint64, error) {
//	detail := c.getDetail(addr)
//	if detail == nil {
//		return 0, fmt.Errorf("can't find latest nonce for addr: %s", addr.Hex())
//	}
//	return detail.getNonce()
//}

// getLatestNonce get latest nonce from its own StateDB
func (c *confirmBatch) getLatestNonce(addr ogTypes.Address) uint64 {
	return c.details.GetNonce(addr)

	//nonce, err := c.getCurrentLatestNonce(addr)
	//if err == nil {
	//	return nonce, nil
	//}
	//return c.getConfirmedLatestNonce(addr)
}

//// getConfirmedLatestNonce get latest nonce from parents and DAG ledger. Note
//// that this confirm batch itself is not included in this nonce search.
//func (c *confirmBatch) getConfirmedLatestNonce(addr ogTypes.Address) (uint64, error) {
//	if c.parent != nil {
//		return c.parent.getLatestNonce(addr)
//	}
//	return c.ledger.GetLatestNonce(addr)
//}

func (c *confirmBatch) bindParent(parent *confirmBatch) {
	c.parent = parent
}

func (c *confirmBatch) bindChildren(child *confirmBatch) {
	c.children = append(c.children, child)
}

func (c *confirmBatch) confirmParent(confirmed *confirmBatch) {
	if c.parent.seq.Hash.Cmp(confirmed.seq.Hash) != 0 {
		return
	}
	c.parent = nil
}
