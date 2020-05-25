package core

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	log "github.com/sirupsen/logrus"
	"sync"
)

type TxStatus uint8

const (
	TxStatusNotExist TxStatus = iota
	TxStatusQueue
	TxStatusTip
	TxStatusBadTx
	TxStatusPending
	TxStatusSeqPreConfirm
	TxStatusSeqPreConfirmByPass

	//TxStatusPreConfirm
)

func (ts *TxStatus) String() string {
	switch *ts {
	case TxStatusBadTx:
		return "BadTx"
	case TxStatusNotExist:
		return "NotExist"
	case TxStatusPending:
		return "Pending"
	case TxStatusQueue:
		return "Queueing"
	case TxStatusTip:
		return "Tip"
	case TxStatusSeqPreConfirm:
		return "SeqPreConfirm"
	case TxStatusSeqPreConfirmByPass:
		return "SeqPreConfirmByPass"
	default:
		return "UnknownStatus"
	}
}

type txPoolStorage struct {
	ledger Ledger

	tips *TxMap
	//badtxs   *TxMap
	//pendings *TxMap
	flows    *AccountFlowSet
	txLookup *txLookUp // txLookUp stores all the txs for external query
}

func newTxPoolStorage(ledger Ledger) *txPoolStorage {
	storage := &txPoolStorage{}
	storage.ledger = ledger

	storage.tips = NewTxMap()
	//storage.badtxs = NewTxMap()
	//storage.pendings = NewTxMap()
	storage.flows = NewAccountFlowSet(ledger)
	storage.txLookup = newTxLookUp()

	return storage
}

func (s *txPoolStorage) init(genesis *tx_types.Sequencer) {
	genesisEnvelope := newTxEnvelope(TxTypeGenesis, TxStatusTip, genesis, 1)
	s.txLookup.Add(genesisEnvelope)
	s.tips.Add(genesis)
}

func (s *txPoolStorage) stats() (int, int, int) {
	return s.txLookup.stats()
}

func (s *txPoolStorage) getTxNum() int {
	return s.txLookup.count()
}

func (s *txPoolStorage) getTxByHash(hash common.Hash) types.Txi {
	return s.txLookup.get(hash)
}

func (s *txPoolStorage) getTxByNonce(addr common.Address, nonce uint64) types.Txi {
	return s.flows.GetTxByNonce(addr, nonce)
}

func (s *txPoolStorage) getTxHashesInOrder() common.Hashes {
	return s.txLookup.getOrder()
}

func (s *txPoolStorage) getTxEnvelope(hash common.Hash) *txEnvelope {
	return s.txLookup.GetEnvelope(hash)
}

func (s *txPoolStorage) getLatestNonce(addr common.Address) (uint64, error) {
	return s.flows.GetLatestNonce(addr)
}

func (s *txPoolStorage) getTxStatusInPool(hash common.Hash) TxStatus {
	return s.txLookup.status(hash)
}

func (s *txPoolStorage) getTipsInList() (v []types.Txi) {
	return s.tips.GetAllValues()
}

func (s *txPoolStorage) getTipsInMap() map[common.Hash]types.Txi {
	return s.tips.txs
}

func (s *txPoolStorage) remove(tx types.Txi, removeType hashOrderRemoveType) {
	hash := tx.GetTxHash()
	s.removeMember(hash, s.getTxStatusInPool(hash))
	s.flows.Remove(tx)
	s.txLookup.Remove(hash, removeType)
}

func (s *txPoolStorage) removeMember(hash common.Hash, status TxStatus) {
	switch status {
	case TxStatusBadTx:
		//s.badtxs.Remove(hash)
	case TxStatusTip:
		s.tips.Remove(hash)
	case TxStatusPending:
		//s.pendings.Remove(hash)
	default:
		log.Warnf("unknown tx status: %s", status.String())
	}

}

func (s *txPoolStorage) removeAll() {
	s.tips = NewTxMap()
	s.flows = NewAccountFlowSet(s.flows.ledger)
	s.txLookup = newTxLookUp()
}

func (s *txPoolStorage) addTxEnv(txEnv *txEnvelope) {
	s.txLookup.add(txEnv)
	s.addMember(txEnv.tx, txEnv.status)
}

func (s *txPoolStorage) addMember(tx types.Txi, status TxStatus) {
	switch status {
	case TxStatusTip:
		s.tips.Add(tx)
	case TxStatusBadTx:
		//s.badtxs.Add(tx)
	case TxStatusPending:
		//s.pendings.Add(tx)
	default:
		//log.Warnf("unknown tx status: %s", status.String())
		return
	}
}

func (s *txPoolStorage) switchTxStatus(hash common.Hash, newStatus TxStatus) {
	oldStatus := s.getTxStatusInPool(hash)
	if oldStatus == newStatus || oldStatus == TxStatusNotExist {
		return
	}
	tx := s.getTxByHash(hash)
	s.addMember(tx, newStatus)
	s.removeMember(hash, oldStatus)
	s.txLookup.switchstatus(hash, newStatus)
}

func (s *txPoolStorage) flowExists(addr common.Address) bool {
	return s.flows.Get(addr) == nil
}

func (s *txPoolStorage) flowReset(addr common.Address) {
	s.flows.ResetFlow(addr, state.NewBalanceSet())
}

func (s *txPoolStorage) flowProcess(tx types.Txi) {
	s.flows.Add(tx)
}

func (s *txPoolStorage) tryProcessTx(tx types.Txi) TxQuality {
	if tx.GetType() != types.TxBaseTypeNormal {
		log.WithField("tx", tx).Tracef("not a normal tx")
		return TxQualityIsGood
	}

	// check if the tx itself has no conflicts with local ledger
	txNormal := tx.(*tx_types.Tx)
	stateFrom := s.flows.GetBalanceState(txNormal.Sender(), txNormal.TokenId)
	if stateFrom == nil {
		originBalance := s.ledger.GetBalance(txNormal.Sender(), txNormal.TokenId)
		stateFrom = NewBalanceState(originBalance)
	}

	// if tx's value is larger than its balance, return fatal.
	if txNormal.Value.Value.Cmp(stateFrom.OriginBalance().Value) > 0 {
		log.WithField("tx", tx).Tracef("fatal tx, tx's value larger than balance")
		return TxQualityIsFatal
	}
	// if ( the value that 'from' already spent )
	// 	+ ( the value that 'from' newly spent )
	// 	> ( balance of 'from' in db )
	totalspent := math.NewBigInt(0)
	if totalspent.Value.Add(stateFrom.spent.Value, txNormal.Value.Value).Cmp(
		stateFrom.originBalance.Value) > 0 {
		log.WithField("tx", tx).Tracef("bad tx, total spent larger than balance")
		return TxQualityIsBad
	}

	return TxQualityIsGood
}

func (s *txPoolStorage) switchToConfirmBatch(batch *confirmBatch) {
	txToRejudge := make([]*txEnvelope, 0)
	newTxOrder := make([]*txEnvelope, 0)
	for _, hash := range s.getTxHashesInOrder() {
		txEnv := s.getTxEnvelope(hash)
		if txEnv.tx.GetType() == types.TxBaseTypeNormal && batch.existTx(hash) {
			newTxOrder = append(newTxOrder, txEnv)
			continue
		} else if txEnv.tx.GetType() == types.TxBaseTypeSequencer && batch.existSeq(hash) {
			newTxOrder = append(newTxOrder, txEnv)
			continue
		}
		txToRejudge = append(txToRejudge, txEnv)
	}
	s.removeAll()

	// deal confirmed txs
	for _, txenv := range newTxOrder {
		if txenv.tx.GetType() == types.TxBaseTypeNormal {
			txenv.status = TxStatusPending
		} else if txenv.tx.GetType() == types.TxBaseTypeSequencer {
			txenv.status = TxStatusSeqPreConfirm
		}
		s.txLookup.Add(txenv)
	}

}

// ----------------------------------------------------
// TxMap
type TxMap struct {
	txs map[common.Hash]types.Txi
	mu  sync.RWMutex
}

func NewTxMap() *TxMap {
	tm := &TxMap{
		txs: make(map[common.Hash]types.Txi),
	}
	return tm
}

func (tm *TxMap) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return len(tm.txs)
}

func (tm *TxMap) Get(hash common.Hash) types.Txi {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.txs[hash]
}

func (tm *TxMap) GetAllKeys() common.Hashes {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var keys common.Hashes
	// slice of keys
	for k := range tm.txs {
		keys = append(keys, k)
	}
	return keys
}

func (tm *TxMap) GetAllValues() []types.Txi {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var values []types.Txi
	// slice of keys
	for _, v := range tm.txs {
		values = append(values, v)
	}
	return values
}

func (tm *TxMap) Exists(tx types.Txi) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if _, ok := tm.txs[tx.GetTxHash()]; !ok {
		return false
	}
	return true
}
func (tm *TxMap) Remove(hash common.Hash) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.txs, hash)
}
func (tm *TxMap) Add(tx types.Txi) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, ok := tm.txs[tx.GetTxHash()]; !ok {
		tm.txs[tx.GetTxHash()] = tx
	}
}

// ----------------------------------------------------
// txLookUp
type txLookUp struct {
	order common.Hashes
	txs   map[common.Hash]*txEnvelope
	mu    sync.RWMutex
}

func newTxLookUp() *txLookUp {
	return &txLookUp{
		order: common.Hashes{},
		txs:   make(map[common.Hash]*txEnvelope),
	}
}

// Get tx from txLookUp by hash
func (t *txLookUp) Get(h common.Hash) types.Txi {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.get(h)
}
func (t *txLookUp) get(h common.Hash) types.Txi {
	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.tx
	}
	return nil
}

// GetEnvelope return the entire tx envelope from txLookUp
func (t *txLookUp) GetEnvelope(h common.Hash) *txEnvelope {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv
	}
	return nil
}

// Add tx into txLookUp
func (t *txLookUp) Add(txEnv *txEnvelope) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.add(txEnv)
}
func (t *txLookUp) add(txEnv *txEnvelope) {
	if _, ok := t.txs[txEnv.tx.GetTxHash()]; ok {
		return
	}

	t.order = append(t.order, txEnv.tx.GetTxHash())
	t.txs[txEnv.tx.GetTxHash()] = txEnv
}

type hashOrderRemoveType byte

const (
	noRemove hashOrderRemoveType = iota
	removeFromFront
	removeFromEnd
)

// Remove tx from txLookUp
func (t *txLookUp) Remove(h common.Hash, removeType hashOrderRemoveType) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.remove(h, removeType)
}

func (t *txLookUp) remove(h common.Hash, removeType hashOrderRemoveType) {
	switch removeType {
	case noRemove:
		return
	case removeFromFront:
		for i, hash := range t.order {
			if hash.Cmp(h) == 0 {
				t.order = append(t.order[:i], t.order[i+1:]...)
				break
			}
		}

	case removeFromEnd:
		for i := len(t.order) - 1; i >= 0; i-- {
			hash := t.order[i]
			if hash.Cmp(h) == 0 {
				t.order = append(t.order[:i], t.order[i+1:]...)
				break
			}
		}
	default:
		panic("unknown remove type")
	}
	delete(t.txs, h)
}

// RemoveTx removes tx from txLookUp.txs only, ignore the order.
func (t *txLookUp) RemoveTxFromMapOnly(h common.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.removeTxFromMapOnly(h)
}
func (t *txLookUp) removeTxFromMapOnly(h common.Hash) {
	delete(t.txs, h)
}

// RemoveByIndex removes a tx by its order index
func (t *txLookUp) RemoveByIndex(i int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.removeByIndex(i)
}

func (t *txLookUp) removeByIndex(i int) {
	if (i < 0) || (i >= len(t.order)) {
		return
	}
	hash := t.order[i]
	t.order = append(t.order[:i], t.order[i+1:]...)
	delete(t.txs, hash)
}

// Order returns hash list of txs in pool, ordered by the time
// it added into pool.
func (t *txLookUp) GetOrder() common.Hashes {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.getOrder()
}

func (t *txLookUp) getOrder() common.Hashes {
	return t.order
}

// Order returns hash list of txs in pool, ordered by the time
// it added into pool.
func (t *txLookUp) ResetOrder() {
	t.mu.RLock()
	defer t.mu.RUnlock()
	t.resetOrder()
}

func (t *txLookUp) resetOrder() {
	t.order = nil
}

// Count returns the total number of txs in txLookUp
func (t *txLookUp) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.count()
}

func (t *txLookUp) count() int {
	return len(t.txs)
}

// Stats returns the count of tips, bad txs, pending txs in txlookup
func (t *txLookUp) Stats() (int, int, int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.stats()
}

func (t *txLookUp) stats() (int, int, int) {
	tips, badtx, pending := 0, 0, 0
	for _, v := range t.txs {
		if v.status == TxStatusTip {
			tips += 1
		} else if v.status == TxStatusBadTx {
			badtx += 1
		} else if v.status == TxStatusPending {
			pending += 1
		}
	}
	return tips, badtx, pending
}

// Status returns the status of a tx
func (t *txLookUp) Status(h common.Hash) TxStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.status(h)
}

func (t *txLookUp) status(h common.Hash) TxStatus {
	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.status
	}
	return TxStatusNotExist
}

// SwitchStatus switches the tx status
func (t *txLookUp) SwitchStatus(h common.Hash, status TxStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.switchstatus(h, status)
}

func (t *txLookUp) switchstatus(h common.Hash, status TxStatus) {
	if txEnv := t.txs[h]; txEnv != nil {
		txEnv.status = status
	}
}
