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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/types/tx_types"
	"testing"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/ogdb"
)

func newTestTxPool(t *testing.T) (*core.TxPool, *core.Dag, *tx_types.Sequencer, func()) {
	txpoolconfig := core.TxPoolConfig{
		QueueSize:     100,
		TipsSize:      100,
		ResetDuration: 5,
		TxVerifyTime:  2,
		TxValidTime:   7,
	}
	db := ogdb.NewMemDatabase()
	dag, errnew := core.NewDag(core.DagConfig{}, state.DefaultStateDBConfig(), db, nil)
	if errnew != nil {
		t.Fatalf("new a dag failed with error: %v", errnew)
	}
	pool := core.NewTxPool(txpoolconfig, dag)

	genesis, balance := core.DefaultGenesis("genesis.json")
	err := dag.Init(genesis, balance)
	if err != nil {
		t.Fatalf("init dag failed with error: %v", err)
	}
	pool.Init(genesis)

	pool.Start()
	dag.Start()

	return pool, dag, genesis, func() {
		pool.Stop()
		dag.Stop()
	}
}

func newTestPoolTx(nonce uint64) *tx_types.Tx {
	txCreator := &og.TxCreator{}
	pk, _ := crypto.PrivateKeyFromString(testPkSecp0)
	addr := newTestAddress(pk)

	tx := txCreator.NewSignedTx(addr, addr, math.NewBigInt(0), nonce, pk, 0)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*tx_types.Tx)
}

func newTestPoolBadTx() *tx_types.Tx {
	txCreator := &og.TxCreator{}
	pk, _ := crypto.PrivateKeyFromString(testPkSecp2)
	addr := newTestAddress(pk)

	tx := txCreator.NewSignedTx(addr, addr, math.NewBigInt(100), 0, pk, 0)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*tx_types.Tx)
}

func TestPoolInit(t *testing.T) {
	t.Parallel()

	pool, _, genesis, finish := newTestTxPool(t)
	defer finish()

	// check if genesis is in txLookUp
	ge := pool.Get(genesis.GetTxHash())
	if ge == nil {
		t.Fatalf("cant get genesis from pool.txLookUp")
	}

	// check genesis's status
	status := pool.GetStatus(genesis.GetTxHash())
	if status != core.TxStatusTip {
		t.Fatalf("genesis's status is not tip but %s", status.String())
	}

}

func TestPoolCommit(t *testing.T) {
	t.Parallel()

	pool, _, genesis, finish := newTestTxPool(t)
	defer finish()

	var err error

	// tx0's parent is genesis
	tx0 := newTestPoolTx(0)
	tx0.ParentsHash = common.Hashes{genesis.GetTxHash()}
	err = pool.AddLocalTx(tx0, true)
	if err != nil {
		t.Fatalf("add tx0 to pool failed: %v", err)
	}
	if pool.Get(tx0.GetTxHash()) == nil {
		t.Fatalf("tx0 is not added into pool")
	}
	if status := pool.GetStatus(tx0.GetTxHash()); status != core.TxStatusTip {
		t.Fatalf("tx0's status is not tip but %s after commit, addr %s", status.String(), tx0.Sender())
	}
	geInPool := pool.Get(genesis.GetTxHash())
	if geInPool != nil {
		t.Fatalf("parent genesis is not removed from pool.")
	}

	// tx1's parent is tx0
	tx1 := newTestPoolTx(1)
	tx1.ParentsHash = common.Hashes{tx0.GetTxHash()}
	err = pool.AddLocalTx(tx1, true)
	if err != nil {
		t.Fatalf("add tx1 to pool failed: %v", err)
	}
	if pool.Get(tx1.GetTxHash()) == nil {
		t.Fatalf("tx1 is not added into pool")
	}
	if status := pool.GetStatus(tx1.GetTxHash()); status != core.TxStatusTip {
		t.Fatalf("tx1's status is not tip but %s after commit", status.String())
	}
	if pool.Get(tx0.GetTxHash()) == nil {
		t.Fatalf("tx0 is not in pool after added tx1")
	}
	if status := pool.GetStatus(tx0.GetTxHash()); status != core.TxStatusPending {
		t.Fatalf("tx0's status is not pending but %s after tx1 added", status.String())
	}

	// tx2's parent is genesis which is not in pool yet
	tx2 := newTestPoolTx(2)
	tx2.ParentsHash = common.Hashes{genesis.GetTxHash()}
	err = pool.AddLocalTx(tx2, true)
	if err != nil {
		t.Fatalf("add tx2 to pool failed: %v", err)
	}
	if pool.Get(tx2.GetTxHash()) == nil {
		t.Fatalf("tx2 is not added into pool")
	}
	if status := pool.GetStatus(tx2.GetTxHash()); status != core.TxStatusTip {
		t.Fatalf("tx2's status is not tip but %s after commit", status.String())
	}

	// TODO bad tx unit test
	// // test bad tx
	// badtx := newTestPoolBadTx()
	// badtx.ParentsHash = common.Hashes{genesis.GetTxHash()}
	// err = pool.AddLocalTx(badtx)
	// if err != nil {
	// 	t.Fatalf("add badtx to pool failed: %v", err)
	// }
	// if pool.Get(badtx.GetTxHash()) == nil {
	// 	t.Fatalf("badtx is not added into pool")
	// }
	// if status := pool.GetStatus(badtx.GetTxHash()); status != core.TxStatusBadTx {
	// 	t.Fatalf("badtx's status is not badtx but %s after commit", status.String())
	// }

}

func TestPoolConfirm(t *testing.T) {
	t.Parallel()

	pool, dag, genesis, finish := newTestTxPool(t)
	defer finish()

	var err error

	// sequencer's parents are normal txs
	tx0 := newTestPoolTx(0)
	tx0.ParentsHash = common.Hashes{genesis.GetTxHash()}
	pool.AddLocalTx(tx0, true)

	// TODO
	// tx3 := newTestPoolBadTx()
	// pool.AddLocalTx(tx3)

	tx1 := newTestPoolTx(1)
	tx1.ParentsHash = common.Hashes{genesis.GetTxHash()}
	pool.AddLocalTx(tx1, true)

	seq := newTestSeq(1)
	seq.ParentsHash = common.Hashes{
		tx0.GetTxHash(),
		tx1.GetTxHash(),
	}
	err = pool.AddLocalTx(seq, true)
	if err != nil {
		t.Fatalf("add seq to pool failed: %v", err)
	}
	if pool.Get(seq.GetTxHash()) == nil {
		t.Fatalf("sequencer is not added into pool")
	}
	if status := pool.GetStatus(seq.GetTxHash()); status != core.TxStatusTip {
		t.Fatalf("sequencer's status is not tip but %s after added", status.String())
	}
	if pool.Get(tx0.GetTxHash()) != nil {
		t.Fatalf("tx0 is not removed from pool")
	}
	if pool.Get(tx1.GetTxHash()) != nil {
		t.Fatalf("tx1 is not removed from pool")
	}
	if dag.GetTx(tx0.GetTxHash()) == nil {
		t.Fatalf("tx0 is not stored in dag")
	}
	if dag.GetTx(tx1.GetTxHash()) == nil {
		t.Fatalf("tx1 is not stored in dag")
	}
	if dag.GetTx(seq.GetTxHash()) == nil {
		t.Fatalf("seq is not stored in dag")
	}
	if dag.LatestSequencer().GetTxHash().Cmp(seq.GetTxHash()) != 0 {
		t.Fatalf("latest seq in dag is not the seq we want")
	}

	// TODO bad tx unit test
	// // sequencer's parent is bad tx
	// badtx := newTestPoolBadTx()
	// badtx.ParentsHash = common.Hashes{seq.GetTxHash()}
	// pool.AddLocalTx(badtx)

	// addr := common.HexToAddress(testAddr2)
	// dag.Accessor().SetBalance(addr, math.NewBigInt(1000))

	// badtxseq := newTestSeq(2)
	// badtxseq.ParentsHash = common.Hashes{badtx.GetTxHash()}
	// badtxseq.ContractHashOrder = common.Hashes{badtx.GetTxHash()}
	// err = pool.AddLocalTx(badtxseq)
	// if err != nil {
	// 	t.Fatalf("add badtxseq to pool failed: %v", err)
	// }
	// if pool.Get(badtxseq.GetTxHash()) == nil {
	// 	t.Fatalf("badtxseq is not added into pool")
	// }
	// if status := pool.GetStatus(badtxseq.GetTxHash()); status != core.TxStatusTip {
	// 	t.Fatalf("badtxseq's status is not tip but %s after added", status.String())
	// }
	// if pool.Get(badtx.GetTxHash()) != nil {
	// 	t.Fatalf("badtx is not removed from pool")
	// }
	// if dag.GetTx(badtx.GetTxHash()) == nil {
	// 	t.Fatalf("badtx is not stored in dag")
	// }
	// if dag.GetTx(badtxseq.GetTxHash()) == nil {
	// 	t.Fatalf("battxseq is not stored in dag")
	// }
	// if dag.LatestSequencer().GetTxHash().Cmp(badtxseq.GetTxHash()) != 0 {
	// 	t.Fatalf("latest seq in dag is not the battxseq we want")
	// }

}
