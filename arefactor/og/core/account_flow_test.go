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
package core_test

import (
	"testing"

	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/types/tx_types"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og"
)

func newTestAccountFlowTx(nonce uint64, value *math.BigInt) *tx_types.Tx {
	txCreator := &og.TxCreator{}
	pk, _ := crypto.PrivateKeyFromString(testPkSecp0)
	addr := newTestAddress(pk)

	tx := txCreator.NewSignedTx(addr, addr, value, nonce, pk, 0)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*tx_types.Tx)
}

func TestTxList(t *testing.T) {
	t.Parallel()

	testNonces := []uint64{208, 505, 910, 157, 771, 718, 98, 897, 538, 38}

	tl := core.NewTxList()
	for _, nonce := range testNonces {
		tx := newTestAccountFlowTx(nonce, math.NewBigInt(0))
		tl.Put(tx)
	}

	if tl.Len() != len(testNonces) {
		t.Fatalf("txlist's length not equal the number of inserted txs")
	}
	for _, nonce := range testNonces {
		tx := tl.Get(nonce)
		if tx == nil {
			t.Fatalf("can't get tx from txlist, nonce: %d", nonce)
		}
		if tx.GetNonce() != nonce {
			t.Fatalf("nonce not same, expect %d but get %d", nonce, tx.GetNonce())
		}
	}
	for _, nonce := range testNonces {
		if !tl.Remove(nonce) {
			t.Fatalf("remove tx from txlist failed, nonce %d", nonce)
		}
		if tl.Get(nonce) != nil {
			t.Fatalf("still get tx from txlist after removed, nonce %d", nonce)
		}
	}
}

func TestBalanceState(t *testing.T) {
	t.Parallel()

	var err error
	var spent int64

	originBalance := math.NewBigInt(100000)
	bs := core.NewBalanceState(originBalance)

	// test TryProcessTx
	fstValue := int64(10000)
	subValue := math.NewBigInt(fstValue)
	err = bs.TryProcessTx(subValue)
	if err != nil {
		t.Fatalf("TryProcessTx err: %v", err)
	}
	spent = bs.Spent().GetInt64()
	if spent != fstValue {
		t.Fatalf("the value of spent is not correct, expect %d, get %d", fstValue, spent)
	}

	// test TryRemoveValue
	tx0value := int64(1000)
	tx1value := int64(2000)
	tx2value := int64(3000)
	tx0 := newTestAccountFlowTx(0, math.NewBigInt(tx0value))
	tx1 := newTestAccountFlowTx(1, math.NewBigInt(tx1value))
	tx2 := newTestAccountFlowTx(2, math.NewBigInt(tx2value))

	err = bs.TryRemoveValue(tx0.GetValue())
	if err != nil {
		t.Fatalf("TryRemoveValue tx0 error: %v", err)
	}
	spent = bs.Spent().GetInt64()
	if spent != (fstValue - tx0value) {
		t.Fatalf("the value of spent is not correct, expect %d, get %d", fstValue-tx0value, spent)
	}
	err = bs.TryRemoveValue(tx1.GetValue())
	if err != nil {
		t.Fatalf("TryRemoveValue tx1 error: %v", err)
	}
	spent = bs.Spent().GetInt64()
	if spent != (fstValue - tx0value - tx1value) {
		t.Fatalf("the value of spent is not correct, expect %d, get %d", fstValue-tx0value-tx1value, spent)
	}
	err = bs.TryRemoveValue(tx2.GetValue())
	if err != nil {
		t.Fatalf("TryRemoveValue tx2 error: %v", err)
	}
	spent = bs.Spent().GetInt64()
	if spent != (fstValue - tx0value - tx1value - tx2value) {
		t.Fatalf("the value of spent is not correct, expect %d, get %d", fstValue-tx0value-tx1value-tx2value, spent)
	}

}

func TestAccountFlow(t *testing.T) {
	t.Parallel()

	var err error

	//balancevalue := int64(100000)
	originBalance := state.NewBalanceSet()
	af := core.NewAccountFlow(originBalance)

	tx0value := int64(1000)
	tx1value := int64(2000)
	tx2value := int64(3000)
	tx0 := newTestAccountFlowTx(0, math.NewBigInt(tx0value))
	tx1 := newTestAccountFlowTx(1, math.NewBigInt(tx1value))
	tx2 := newTestAccountFlowTx(2, math.NewBigInt(tx2value))

	// test add, get
	err = af.Add(tx0)
	if err != nil {
		t.Fatalf("can't add tx0 into account flow, err: %v", err)
	}
	err = af.Add(tx1)
	if err != nil {
		t.Fatalf("can't add tx1 into account flow, err: %v", err)
	}
	err = af.Add(tx2)
	if err != nil {
		t.Fatalf("can't add tx2 into account flow, err: %v", err)
	}
	tx0inAccountFlow := af.GetTx(tx0.GetNonce())
	if tx0inAccountFlow == nil {
		t.Fatalf("can't get tx0 from account flow after add")
	}
	tx1inAccountFlow := af.GetTx(tx1.GetNonce())
	if tx1inAccountFlow == nil {
		t.Fatalf("can't get tx1 from account flow after add")
	}
	tx2inAccountFlow := af.GetTx(tx2.GetNonce())
	if tx2inAccountFlow == nil {
		t.Fatalf("can't get tx2 from account flow after add")
	}

	// test latest nonce
	latestnonce, lnerr := af.LatestNonce()
	if lnerr != nil {
		t.Fatalf("get latest nonce failed, err: %v", lnerr)
	}
	if latestnonce != uint64(2) {
		t.Fatalf("latest nonce not correct, expect %d, get %d", 2, latestnonce)
	}

	// test remove
	spent := af.BalanceState(0).Spent()

	err = af.Remove(tx0.GetNonce())
	if err != nil {
		t.Fatalf("confirm tx0 err: %v", err)
	}
	spent = af.BalanceState(0).Spent()
	if spent.Value.Cmp(math.NewBigInt(tx1value+tx2value).Value) != 0 {
		t.Fatalf("spent not correct after confirm tx0, expect: %d, get %d", tx1value+tx2value, spent.GetInt64())
	}

	err = af.Remove(tx1.GetNonce())
	if err != nil {
		t.Fatalf("confirm tx1 err: %v", err)
	}
	spent = af.BalanceState(0).Spent()
	if spent.Value.Cmp(math.NewBigInt(tx2value).Value) != 0 {
		t.Fatalf("spent not correct after confirm tx1, expect: %d, get %d", tx2value, spent.GetInt64())
	}

	err = af.Remove(tx2.GetNonce())
	if err != nil {
		t.Fatalf("confirm tx2 err: %v", err)
	}
	spent = af.BalanceState(0).Spent()
	if spent.Value.Cmp(math.NewBigInt(0).Value) != 0 {
		t.Fatalf("spent not correct after confirm tx2, expect: %d, get %d", 0, spent.GetInt64())
	}

}
