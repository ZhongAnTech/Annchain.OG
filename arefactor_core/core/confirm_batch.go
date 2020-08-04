package core

import (
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/types"
	"github.com/annchain/OG/arefactor_core/core/state"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogdb"
)

type CachedConfirms struct {
	//highest *ConfirmBatch
	fronts  []*ConfirmBatch
	batches map[ogTypes.HashKey]*ConfirmBatch
}

func newCachedConfirms() *CachedConfirms {
	return &CachedConfirms{
		//highest: nil,
		fronts:  make([]*ConfirmBatch, 0),
		batches: make(map[ogTypes.HashKey]*ConfirmBatch),
	}
}

func (c *CachedConfirms) stop() {
	for _, batch := range c.batches {
		batch.stop()
	}
}

func (c *CachedConfirms) getConfirmBatch(seqHash ogTypes.Hash) *ConfirmBatch {
	return c.batches[seqHash.HashKey()]
}

//func (c *CachedConfirms) existTx(seqHash ogTypes.Hash, txHash ogTypes.Hash) bool {
//	batch := c.batches[seqHash.HashKey()]
//	if batch != nil {
//		return batch.existTx(txHash)
//	}
//	//return c.ledger.GetTx(txHash) != nil
//	return false
//}

func (c *CachedConfirms) existTx(baseBatchHash ogTypes.Hash, hash ogTypes.Hash) bool {
	baseBatch := c.getConfirmBatch(baseBatchHash)
	for baseBatch != nil {
		if baseBatch.existCurrentTx(hash) {
			return true
		}
		baseBatch = baseBatch.parent
	}
	return false
}

func (c *CachedConfirms) getTxAndReceipt(hash ogTypes.Hash) (types.Txi, *Receipt) {
	for _, batch := range c.batches {
		if batch.existCurrentTx(hash) {
			return batch.getCurrentTx(hash), batch.getCurrentReceipt(hash)
		}
	}
	return nil, nil
}

func (c *CachedConfirms) getTxByNonce(baseBatchHash ogTypes.Hash, addr ogTypes.Address, nonce uint64) types.Txi {
	baseBatch := c.getConfirmBatch(baseBatchHash)
	for baseBatch != nil {
		txi := baseBatch.getCurrentTxByNonce(addr, nonce)
		if txi != nil {
			return txi
		}
		baseBatch = baseBatch.parent
	}
	return nil
}

// getTxsByHeight searching txs by sequencer height, traverse from leaf to root
func (c *CachedConfirms) getTxsByHeight(baseBatchHash ogTypes.Hash, height uint64) []types.Txi {
	baseBatch := c.getConfirmBatch(baseBatchHash)
	for baseBatch != nil {
		if baseBatch.seq.GetHeight() < height {
			return nil
		}
		if baseBatch.seq.GetHeight() == height {
			return baseBatch.elders
		}
		baseBatch = baseBatch.parent
	}
	return nil
}

// getSeqByHeight searching sequencer by sequencer height, traverse from leaf to root
func (c *CachedConfirms) getSeqByHeight(baseBatchHash ogTypes.Hash, height uint64) *types.Sequencer {
	baseBatch := c.getConfirmBatch(baseBatchHash)
	for baseBatch != nil {
		if baseBatch.seq.GetHeight() < height {
			return nil
		}
		if baseBatch.seq.GetHeight() == height {
			return baseBatch.seq
		}
		baseBatch = baseBatch.parent
	}
	return nil
}

func (c *CachedConfirms) push(hashKey ogTypes.Hash, batch *ConfirmBatch) {
	parentBatch := c.getConfirmBatch(batch.seq.GetParentSeqHash())
	if parentBatch == nil {
		c.fronts = append(c.fronts, batch)
	} else {
		parentBatch.bindChildren(batch)
		batch.bindParent(parentBatch)
	}
	c.batches[hashKey.HashKey()] = batch

	//// set highest batch
	//if c.highest != nil && c.highest.seq.GetHeight() >= batch.seq.GetHeight() {
	//	return
	//}
	//c.highest = batch
}

// purePush only store the ConfirmBatch to the batch map, regardless the fronts
func (c *CachedConfirms) purePush(hash ogTypes.Hash, batch *ConfirmBatch) {
	c.batches[hash.HashKey()] = batch
}

// pureDelete only delete the ConfirmBatch from batch map, regardless fronts
func (c *CachedConfirms) pureDelete(hash ogTypes.Hash) {
	delete(c.batches, hash.HashKey())
}

func (c *CachedConfirms) confirm(batch *ConfirmBatch) {
	// delete conflicts batches
	for _, batchToDelete := range c.fronts {
		//batchToDelete := c.fronts[i]
		if batchToDelete.isSame(batch) {
			continue
		}
		c.traverseFromRoot(batchToDelete, func(b *ConfirmBatch) {
			delete(c.batches, b.seq.GetTxHash().HashKey())
		})
	}
	c.fronts = batch.children

	// unbind children's parent to nil
	for _, childBatch := range batch.children {
		childBatch.confirmParent(batch)
	}
}

// traverseFromRoot traverse the cached ConfirmBatch trees and process the function "f" for
// every found confirm batches.
func (c *CachedConfirms) traverseFromRoot(root *ConfirmBatch, f func(b *ConfirmBatch)) {
	seekingPool := make([]*ConfirmBatch, 0)
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

func (c *CachedConfirms) traverseFromLeaf(leaf *ConfirmBatch, f func(b *ConfirmBatch)) {
	seekingPool := make([]*ConfirmBatch, 0)
	seekingPool = append(seekingPool, leaf)

	for len(seekingPool) > 0 {
		batch := seekingPool[0]
		seekingPool = seekingPool[1:]

		f(batch)
		seekingPool = append(seekingPool, batch.parent)
	}
}

type ConfirmBatch struct {
	//tempLedger Ledger
	//ledger Ledger

	parent   *ConfirmBatch
	children []*ConfirmBatch

	db               *state.StateDB
	seq              *types.Sequencer
	seqReceipt       *Receipt
	txReceipts       ReceiptSet
	elders           []types.Txi
	eldersQueryMap   map[ogTypes.HashKey]types.Txi
	eldersQueryNonce map[ogTypes.AddressKey]*TxList

	//db        map[ogTypes.AddressKey]*accountDetail
}

func newConfirmBatch(seq *types.Sequencer, db ogdb.Database, baseRoot ogTypes.Hash) (*ConfirmBatch, error) {
	stateDB, err := state.NewStateDB(state.DefaultStateDBConfig(), state.NewDatabase(db), baseRoot)
	if err != nil {
		return nil, err
	}
	c := &ConfirmBatch{
		parent:         nil,
		children:       make([]*ConfirmBatch, 0),
		db:             stateDB,
		seq:            seq,
		seqReceipt:     nil,
		txReceipts:     nil,
		elders:         make([]types.Txi, 0),
		eldersQueryMap: make(map[ogTypes.HashKey]types.Txi),
	}
	return c, nil
}

func (c *ConfirmBatch) stop() {
	c.db.Stop()
}

//func (c *ConfirmBatch) construct(elders map[ogTypes.HashKey]types.Txi) error {
//	for _, txi := range elders {
//		// return error if a sequencer confirm a tx that has same nonce as itself.
//		if txi.Sender() == c.seq.Sender() && txi.GetNonce() == c.seq.GetNonce() {
//			return fmt.Errorf("seq's nonce is the same as a tx it confirmed, nonce: %d, tx hash: %s",
//				c.seq.GetNonce(), txi.GetTxHash())
//		}
//
//		switch tx := txi.(type) {
//		case *types.Sequencer:
//			break
//		case *types.Tx:
//			//c.processTx(tx)
//			// TODO
//		default:
//			c.addTx(tx)
//		}
//	}
//	return nil
//}

//func (c *ConfirmBatch) isValid() error {
//	// verify balance and nonce
//	for _, accountDetail := range c.getDetails() {
//		err := accountDetail.isValid()
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

func (c *ConfirmBatch) isSame(cb *ConfirmBatch) bool {
	return c.seq.GetTxHash().Cmp(cb.seq.GetTxHash()) == 0
}

//func (c *ConfirmBatch) getOrCreateDetail(addr ogTypes.Address) *accountDetail {
//	detailCost := c.getDetail(addr)
//	if detailCost == nil {
//		detailCost = c.createDetail(addr)
//	}
//	return detailCost
//}
//
//func (c *ConfirmBatch) getDetail(addr ogTypes.Address) *accountDetail {
//	return c.db[addr.AddressKey()]
//}
//
//func (c *ConfirmBatch) getDetails() map[ogTypes.AddressKey]*accountDetail {
//	return c.db
//}
//
//func (c *ConfirmBatch) createDetail(addr ogTypes.Address) *accountDetail {
//	c.db[addr.AddressKey()] = newBatchDetail(addr, c)
//	return c.db[addr.AddressKey()]
//}
//
//func (c *ConfirmBatch) setDetail(detail *accountDetail) {
//	c.db[detail.address.AddressKey()] = detail
//}

//func (c *ConfirmBatch) processTx(txi types.Txi) {
//	if txi.GetType() != types.TxBaseTypeNormal {
//		return
//	}
//	tx := txi.(*types.Tx)
//
//	detailCost := c.getOrCreateDetail(tx.From)
//	detailCost.addCost(tx.TokenId, tx.Value)
//	detailCost.addTx(tx)
//	c.setDetail(detailCost)
//
//	detailEarn := c.getOrCreateDetail(tx.To)
//	detailEarn.addEarn(tx.TokenId, tx.Value)
//	c.setDetail(detailEarn)
//
//	c.addTxToElders(tx)
//}

// addTx adds tx only without any processing.
func (c *ConfirmBatch) addTx(tx types.Txi) {
	//detailSender := c.getOrCreateDetail(tx.Sender())
	//detailSender.addTx(tx)

	c.addTxToElders(tx)
}

func (c *ConfirmBatch) addTxToElders(tx types.Txi) {
	c.elders = append(c.elders, tx)
	c.eldersQueryMap[tx.GetTxHash().HashKey()] = tx

	txList := c.eldersQueryNonce[tx.Sender().AddressKey()]
	if txList == nil {
		txList = NewTxList()
	}
	txList.Put(tx)
	c.eldersQueryNonce[tx.Sender().AddressKey()] = txList
}

func (c *ConfirmBatch) existCurrentTx(hash ogTypes.Hash) bool {
	return c.eldersQueryMap[hash.HashKey()] != nil
}

// existTx checks if input tx exists in current confirm batch. If not
// exists, then check parents confirm batch and check DAG ledger at last.
func (c *ConfirmBatch) existTx(hash ogTypes.Hash) bool {
	exists := c.existCurrentTx(hash)
	if exists {
		return exists
	}
	if c.parent != nil {
		return c.parent.existTx(hash)
	}
	return false
}

func (c *ConfirmBatch) getCurrentTx(hash ogTypes.Hash) types.Txi {
	return c.eldersQueryMap[hash.HashKey()]
}

func (c *ConfirmBatch) getCurrentTxByNonce(addr ogTypes.Address, nonce uint64) types.Txi {
	txList := c.eldersQueryNonce[addr.AddressKey()]
	if txList == nil {
		return nil
	}
	return txList.Get(nonce)
}

func (c *ConfirmBatch) existSeq(seqHash ogTypes.Hash) bool {
	if c.seq.GetTxHash().Cmp(seqHash) == 0 {
		return true
	}
	if c.parent != nil {
		return c.parent.existSeq(seqHash)
	}
	//return c.ledger.GetTx(seqHash) != nil
	return false
}

func (c *ConfirmBatch) getCurrentReceipt(hash ogTypes.Hash) *Receipt {
	if c.seq.GetTxHash().Cmp(hash) == 0 {
		return c.seqReceipt
	}
	return c.txReceipts[hash.HashKey()]
}

//func (c *ConfirmBatch) getCurrentBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
//	detail := c.getDetail(addr)
//	if detail == nil {
//		return nil
//	}
//	return detail.getBalance(tokenID)
//}

// getBalance get balance from its own StateDB
func (c *ConfirmBatch) getBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
	return c.db.GetTokenBalance(addr, tokenID)

	//blc := c.getCurrentBalance(addr, tokenID)
	//if blc != nil {
	//	return blc
	//}
	//return c.getConfirmedBalance(addr, tokenID)
}

//func (c *ConfirmBatch) getConfirmedBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
//	if c.parent != nil {
//		return c.parent.getBalance(addr, tokenID)
//	}
//	return c.ledger.GetBalance(addr, tokenID)
//}

//func (c *ConfirmBatch) getCurrentLatestNonce(addr ogTypes.Address) (uint64, error) {
//	detail := c.getDetail(addr)
//	if detail == nil {
//		return 0, fmt.Errorf("can't find latest nonce for addr: %s", addr.Hex())
//	}
//	return detail.getNonce()
//}

// getLatestNonce get latest nonce from its own StateDB
func (c *ConfirmBatch) getLatestNonce(addr ogTypes.Address) uint64 {
	return c.db.GetNonce(addr)

	//nonce, err := c.getCurrentLatestNonce(addr)
	//if err == nil {
	//	return nonce, nil
	//}
	//return c.getConfirmedLatestNonce(addr)
}

//// getConfirmedLatestNonce get latest nonce from parents and DAG ledger. Note
//// that this confirm batch itself is not included in this nonce search.
//func (c *ConfirmBatch) getConfirmedLatestNonce(addr ogTypes.Address) (uint64, error) {
//	if c.parent != nil {
//		return c.parent.getLatestNonce(addr)
//	}
//	return c.ledger.GetLatestNonce(addr)
//}

func (c *ConfirmBatch) bindParent(parent *ConfirmBatch) {
	c.parent = parent
}

func (c *ConfirmBatch) bindChildren(child *ConfirmBatch) {
	c.children = append(c.children, child)
}

func (c *ConfirmBatch) confirmParent(confirmed *ConfirmBatch) {
	if c.parent.seq.Hash.Cmp(confirmed.seq.Hash) != 0 {
		return
	}
	c.parent = nil
}
