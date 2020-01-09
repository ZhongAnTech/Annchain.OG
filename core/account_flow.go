// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package core

import (
	"container/heap"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/types/tx_types"
	"sort"
	"sync"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type AccountFlows struct {
	afs  map[common.Address]*AccountFlow
	pool *TxPool
	mu   sync.RWMutex
}

func NewAccountFlows(pool *TxPool) *AccountFlows {
	return &AccountFlows{
		afs:  make(map[common.Address]*AccountFlow),
		pool: pool,
	}
}

func (a *AccountFlows) Add(tx types.Txi) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if tx.GetType() == types.TxBaseTypeArchive {
		return
	}

	af := a.afs[tx.Sender()]
	if af == nil {
		af = NewAccountFlow(state.NewBalanceSet())
	}
	if tx.GetType() == types.TxBaseTypeNormal {
		txn := tx.(*tx_types.Tx)
		if af.balances[txn.TokenId] == nil {
			blc := a.pool.dag.GetBalance(txn.Sender(), txn.TokenId)
			af.balances[txn.TokenId] = NewBalanceState(blc)
		}
	}
	af.Add(tx)
	a.afs[tx.Sender()] = af
}

func (a *AccountFlows) Get(addr common.Address) *AccountFlow {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.afs[addr]
}

func (a *AccountFlows) GetBalanceState(addr common.Address, tokenID int32) *BalanceState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	af := a.Get(addr)
	if af == nil {
		return nil
	}
	bls := af.balances
	if bls == nil {
		return nil
	}
	return bls[tokenID]
}

func (a *AccountFlows) GetTxByNonce(addr common.Address, nonce uint64) types.Txi {
	a.mu.RLock()
	defer a.mu.RUnlock()

	flow := a.afs[addr]
	if flow == nil {
		return nil
	}
	return flow.GetTx(nonce)
}

func (a *AccountFlows) GetLatestNonce(addr common.Address) (uint64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	flow := a.afs[addr]
	if flow == nil {
		return 0, fmt.Errorf("no related tx in txlookup")
	}
	if !(flow.Len() > 0) {
		return 0, fmt.Errorf("flow not long enough")
	}
	return flow.LatestNonce()
}

func (a *AccountFlows) ResetFlow(addr common.Address, originBalance state.BalanceSet) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.afs[addr] = NewAccountFlow(originBalance)
}

func (a *AccountFlows) Remove(tx types.Txi) {
	a.mu.Lock()
	defer a.mu.Unlock()

	flow := a.afs[tx.Sender()]
	if flow == nil {
		log.WithField("tx", tx).Warnf("remove tx from accountflows failed")
		return
	}
	flow.Remove(tx.GetNonce())
	// remove account flow if there is no txs sent by this address in pool
	if flow.Len() == 0 {
		delete(a.afs, tx.Sender())
	}
}

// AccountFlow stores the information about an address. It includes the
// balance state of the account among the txpool,
type AccountFlow struct {
	balances map[int32]*BalanceState
	txlist   *TxList
}

func NewAccountFlow(originBalance state.BalanceSet) *AccountFlow {
	bls := map[int32]*BalanceState{}
	for k, v := range originBalance {
		bls[k] = NewBalanceState(v)
	}

	return &AccountFlow{
		balances: bls,
		txlist:   NewTxList(),
	}
}
func (af *AccountFlow) BalanceState(tokenID int32) *BalanceState {
	return af.balances[tokenID]
}
func (af *AccountFlow) TxList() *TxList {
	return af.txlist
}

// return the count of txs sent by this account.
func (af *AccountFlow) Len() int {
	return af.txlist.Len()
}

// GetTx get a tx from accountflow.
func (af *AccountFlow) GetTx(nonce uint64) types.Txi {
	return af.txlist.Get(nonce)
}

// Add new tx into account flow. This function should
// 1. update account's balance state.
// 2. add tx into nonce sorted txlist.
func (af *AccountFlow) Add(tx types.Txi) error {
	if af.txlist.get(tx.GetNonce()) != nil {
		log.WithField("tx", tx).Errorf("add tx that has same nonce")
		return fmt.Errorf("already exists")
	}
	if tx.GetType() != types.TxBaseTypeNormal {
		af.txlist.Put(tx)
		return nil
	}
	txnormal := tx.(*tx_types.Tx)
	if af.balances[txnormal.TokenId] == nil {
		af.txlist.Put(tx)
		return fmt.Errorf("accountflow not exists for addr: %s", tx.Sender().Hex())
	}
	value := txnormal.GetValue()
	err := af.balances[txnormal.TokenId].TrySubBalance(value)
	if err != nil {
		return err
	}
	af.txlist.Put(tx)
	return nil
}

// Remove a tx from account flow, find tx by nonce first, then
// rolls back the balance and remove tx from txlist.
func (af *AccountFlow) Remove(nonce uint64) error {
	tx := af.txlist.Get(nonce)
	if tx == nil {
		return nil
	}
	if tx.GetType() != types.TxBaseTypeNormal {
		af.txlist.Remove(nonce)
		return nil
	}
	txnormal := tx.(*tx_types.Tx)
	if af.balances[txnormal.TokenId] == nil {
		af.txlist.Remove(nonce)
		return fmt.Errorf("accountflow not exists for addr: %s", tx.Sender().Hex())
	}
	value := txnormal.GetValue()
	err := af.balances[txnormal.TokenId].TryRemoveValue(value)
	if err != nil {
		return err
	}
	af.txlist.Remove(nonce)
	return nil
}

// LatestNonce returns the largest nonce stored in txlist.
func (af *AccountFlow) LatestNonce() (uint64, error) {
	tl := af.txlist
	if tl == nil {
		return 0, fmt.Errorf("txlist is nil")
	}
	if !(tl.Len() > 0) {
		return 0, fmt.Errorf("txlist is empty")
	}
	keys := tl.keys
	if keys == nil {
		return 0, fmt.Errorf("txlist's keys field is nil")
	}
	return keys.Tail(), nil
}

type BalanceState struct {
	spent         *math.BigInt
	originBalance *math.BigInt
}

func NewBalanceState(balance *math.BigInt) *BalanceState {
	return &BalanceState{
		spent:         math.NewBigInt(0),
		originBalance: balance,
	}
}
func (bs *BalanceState) Spent() *math.BigInt {
	return bs.spent
}
func (bs *BalanceState) OriginBalance() *math.BigInt {
	return bs.originBalance
}

// TrySubBalance checks if origin balance is enough for total spent of
// txs in pool. It trys to add new spent "value" into total spent and
// compare total spent with origin balance.
func (bs *BalanceState) TrySubBalance(value *math.BigInt) error {
	totalspent := math.NewBigInt(0)
	totalspent.Value.Add(bs.spent.Value, value.Value)
	// check if (already spent + new spent) > confirmed balance
	if totalspent.Value.Cmp(bs.originBalance.Value) > 0 {
		return fmt.Errorf("balance not enough")
	}
	bs.spent.Value = totalspent.Value
	return nil
}

// TryRemoveValue is called when remove a tx from pool. It reduce the total spent
// by the value of removed tx.
func (bs *BalanceState) TryRemoveValue(txValue *math.BigInt) error {
	// check if (already spent < tx's value)
	if bs.spent.Value.Cmp(txValue.Value) < 0 {
		return fmt.Errorf("tx's value is too much to remove, spent: %s, tx value: %s", bs.spent.String(), txValue.String())
	}
	bs.spent.Value.Sub(bs.spent.Value, txValue.Value)
	bs.originBalance.Value.Sub(bs.originBalance.Value, txValue.Value)
	return nil
}

type nonceHeap []uint64

func (n nonceHeap) Tail() uint64 {
	sort.Sort(n)
	return n[n.Len()-1]
}

// for sort
func (n nonceHeap) Len() int           { return len(n) }
func (n nonceHeap) Less(i, j int) bool { return n[i] < n[j] }
func (n nonceHeap) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

// for heap
func (n *nonceHeap) Push(x interface{}) {
	*n = append(*n, x.(uint64))
}
func (n *nonceHeap) Pop() interface{} {
	old := *n
	length := len(old)
	last := old[length-1]
	*n = old[0 : length-1]
	return last
}

type TxList struct {
	keys   *nonceHeap
	txflow map[uint64]types.Txi
}

func NewTxList() *TxList {
	return &TxList{
		keys:   new(nonceHeap),
		txflow: make(map[uint64]types.Txi),
	}
}

func (t *TxList) Len() int {
	return t.keys.Len()
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
		}
	}
	delete(t.txflow, nonce)
	return true
}
