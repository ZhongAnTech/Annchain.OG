package core

import (
	"sort"
	"fmt"
	"container/heap"
	"sync"

	"github.com/annchain/OG/types"
)

type nonceHeap []uint64
// for sort
func (n nonceHeap) Len() int 			{ return len(n) }
func (n nonceHeap) Less(i, j int) bool	{ return n[i] < n[j] }
func (n nonceHeap) Swap(i, j int)		{ n[i], n[j] = n[j], n[i] }
// for heap
func (n *nonceHeap) Push(x interface{}) {
	*n = append(*n, x.(uint64))
}
func (n *nonceHeap) Pop() interface{} {
	old := *n
	length := len(old)
	last := old[length-1]
	*n = old[0:length-1]
	return last
}

type TxList struct {
	keys	*nonceHeap
	txflow	map[uint64]types.Txi
}
func NewTxList() *TxList {
	return &TxList{
		keys: new(nonceHeap),
		txflow: make(map[uint64]types.Txi),
	}
}

func (t *TxList) Get(nonce uint64) types.Txi {
	return t.get(nonce)
}
func (t *TxList) get(nonce uint64) types.Txi {
	return t.txflow[nonce]
}

func (t *TxList) Put(txi types.Txi) {
	t.put(txi)
}
func (t *TxList) put(txi types.Txi) {
	nonce := txi.GetNonce()
	if _, ok := t.txflow[nonce]; !ok {
		heap.Push(t.keys, nonce)
		t.txflow[nonce] = txi
	}
}

func (t *TxList) Remove(nonce uint64) bool {
	return t.remove(nonce)
}
func (t *TxList) remove(nonce uint64) bool {
	_, ok := t.txflow[nonce]
	if !ok {
		return false
	}
	for i := 0; i < t.keys.Len(); i++ {
		if (*t.keys)[i] == nonce {
			heap.Remove(t.keys, i)	
			break
		}
	}
	delete(t.txflow, nonce)
	return true
}

type txLookUp struct {
	txs 		map[types.Hash]*txEnvelope
	txflow		map[types.Address]*TxList
	mu  		sync.RWMutex
}
func newTxLookUp() *txLookUp {
	return &txLookUp{
		txs: make(map[types.Hash]*txEnvelope),
		txflow: make(map[types.Address]*TxList),
	}
}

// Get tx from txLookUp by hash
func (t *txLookUp) Get(h types.Hash) types.Txi {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.get(h)
}
func (t *txLookUp) get(h types.Hash) types.Txi {
	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.tx
	}
	return nil
}

// Get tx by address and nonce.
func (t *txLookUp) GetByNonce(addr types.Address, nonce uint64) types.Txi {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.getByNonce(addr, nonce)
}
func (t *txLookUp) getByNonce(addr types.Address, nonce uint64) types.Txi {
	txlist := t.txflow[addr]
	if txlist == nil {
		return nil
	}
	return txlist.get(nonce)
}

// GetLastestNonce returns the latest nonce of an address
func (t *txLookUp) GetLatestNonce(addr types.Address) (uint64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.getLatestNonce(addr)
}
func (t *txLookUp) getLatestNonce(addr types.Address) (uint64, error) {
	txlist := t.txflow[addr]
	if txlist == nil {
		return 0, fmt.Errorf("no related tx in txlookup")
	}
	if !(txlist.keys.Len() > 0) {
		return 0, fmt.Errorf("txlist not long enough")
	}
	sort.Sort(txlist.keys)
	// WARN: keys stored in txlookup should not be changed!
	// TODO: need unit test here.
	keys := *txlist.keys
	return keys.Pop().(uint64), nil
}

// Add tx into txLookUp
func (t *txLookUp) Add(txEnv *txEnvelope) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.add(txEnv)
}
func (t *txLookUp) add(txEnv *txEnvelope) {
	tx := txEnv.tx

	txlist := t.txflow[tx.Sender()]
	if txlist == nil {
		txlist = NewTxList()
		t.txflow[tx.Sender()] = txlist
	}
	txlist.put(tx)

	t.txs[txEnv.tx.GetTxHash()] = txEnv
}

// Remove tx from txLookUp
func (t *txLookUp) Remove(h types.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.remove(h)
}
func (t *txLookUp) remove(h types.Hash) {
	tx := t.get(h)
	if tx == nil {
		return
	}
	txlist := t.txflow[tx.Sender()]
	if txlist != nil {
		txlist.remove(tx.GetNonce())
	}
	delete(t.txs, h)
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
func (t *txLookUp) Status(h types.Hash) TxStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.status(h)
}
func (t *txLookUp) status(h types.Hash) TxStatus {
	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.status
	}
	return TxStatusNotExist
}

// SwitchStatus switches the tx status
func (t *txLookUp) SwitchStatus(h types.Hash, status TxStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.switchstatus(h, status)
}
func (t *txLookUp) switchstatus(h types.Hash, status TxStatus) {
	if txEnv := t.txs[h]; txEnv != nil {
		txEnv.status = status
	}
}


