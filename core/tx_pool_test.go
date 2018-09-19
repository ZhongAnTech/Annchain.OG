package core_test

import (
	"testing"

	"github.com/annchain/OG/core"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
)

func newTestTxPool(t *testing.T) (*core.TxPool, *core.Dag, *types.Sequencer, func()) {
	txpoolconfig := core.TxPoolConfig{
		QueueSize: 100,
		TipsSize: 100,
		ResetDuration: 5,
		TxVerifyTime: 2,
		TxValidTime: 7,
	}
	db := ogdb.NewMemDatabase()
	dag := core.NewDag(core.DagConfig{}, db)
	pool := core.NewTxPool(txpoolconfig, dag)

	genesis, balance := og.DefaultGenesis()
	err := dag.Init(genesis, balance)
	if err != nil {
		t.Fatalf("init dag failed with error: %v", err)
	}
	pool.Init(genesis)

	pool.Start()
	dag.Start()

	return pool, dag, genesis, func(){
		pool.Stop()
		dag.Stop()
	}
}

func newTestPoolTx(nonce uint64) *types.Tx {
	txCreator := &og.TxCreator{
		Signer: &crypto.SignerSecp256k1{},
	}
	pk, _ := crypto.PrivateKeyFromString(testPk0)
	addr := types.HexToAddress(testAddr0)

	tx := txCreator.NewSignedTx(addr, addr, math.NewBigInt(0), nonce, pk)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*types.Tx)
}

func newTestPoolBadTx() *types.Tx {
	txCreator := &og.TxCreator{
		Signer: &crypto.SignerSecp256k1{},
	}
	pk, _ := crypto.PrivateKeyFromString(testPk0)
	addr := types.HexToAddress(testAddr0)

	tx := txCreator.NewSignedTx(addr, addr, math.NewBigInt(100), 0, pk)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*types.Tx)
}

func TestPoolInit(t *testing.T) {
	t.Parallel()

	pool, _, genesis, finish := newTestTxPool(t)
	defer finish()

	// check if genesis is the only tip
	tips := pool.GetAllTips()
	if len(tips) != 1 {
		t.Fatalf("should have only one tip")
	}
	tip := tips[genesis.GetTxHash()]
	if tip == nil {
		t.Fatalf("genesis not stored in tips")
	}

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

	// tx1's parent is genesis
	tx1 := newTestPoolTx(1)
	tx1.ParentsHash = []types.Hash{genesis.GetTxHash()}
	err = pool.AddLocalTx(tx1)
	if err != nil {
		t.Fatalf("add tx1 to pool failed: %v", err)
	}
	if pool.Get(tx1.GetTxHash()) == nil {
		t.Fatalf("tx1 is not added into pool")
	}
	if status := pool.GetStatus(tx1.GetTxHash()); status != core.TxStatusTip {
		t.Fatalf("tx1's status is not tip but %s after commit", status.String())
	}
	geInPool := pool.Get(genesis.GetTxHash())
	if geInPool != nil {
		t.Fatalf("parent genesis is not removed from pool.")
	}

	// tx2's parent is tx1
	tx2 := newTestPoolTx(2)
	tx2.ParentsHash = []types.Hash{tx1.GetTxHash()}
	err = pool.AddLocalTx(tx2)
	if err != nil {
		t.Fatalf("add tx2 to pool failed: %v", err)
	}
	if pool.Get(tx2.GetTxHash()) == nil {
		t.Fatalf("tx2 is not added into pool")
	}
	if status := pool.GetStatus(tx2.GetTxHash()); status != core.TxStatusTip {
		t.Fatalf("tx2's status is not tip but %s after commit", status.String())
	}
	if pool.Get(tx1.GetTxHash()) == nil {
		t.Fatalf("tx1 is not in pool after added tx2")
	}
	if status := pool.GetStatus(tx1.GetTxHash()); status != core.TxStatusPending {
		t.Fatalf("tx1's status is not pending but %s after tx2 added", status.String())
	}

	// tx3's parent is genesis which is not in pool yet
	tx3 := newTestPoolTx(3)
	tx3.ParentsHash = []types.Hash{genesis.GetTxHash()}
	err = pool.AddLocalTx(tx3)
	if err != nil {
		t.Fatalf("add tx3 to pool failed: %v", err)
	}
	if pool.Get(tx3.GetTxHash()) == nil {
		t.Fatalf("tx3 is not added into pool")
	}
	if status := pool.GetStatus(tx3.GetTxHash()); status != core.TxStatusTip {
		t.Fatalf("tx3's status is not tip but %s after commit", status.String())
	}

	// test bad tx
	badtx := newTestPoolBadTx()
	badtx.ParentsHash = []types.Hash{genesis.GetTxHash()}
	err = pool.AddLocalTx(badtx)
	if err != nil {
		t.Fatalf("add badtx to pool failed: %v", err)
	}
	if pool.Get(badtx.GetTxHash()) == nil {
		t.Fatalf("badtx is not added into pool")
	}
	if status := pool.GetStatus(badtx.GetTxHash()); status != core.TxStatusBadTx {
		t.Fatalf("badtx's status is not tip but %s after commit", status.String())
	}

}

func TestPoolConfirm(t *testing.T) {
	t.Parallel()

	pool, dag, genesis, finish := newTestTxPool(t)
	defer finish()

	var err error

	// sequencer's parents are normal txs
	tx1 := newTestPoolTx(1)
	tx1.ParentsHash = []types.Hash{genesis.GetTxHash()}
	pool.AddLocalTx(tx1)

	tx2 := newTestPoolTx(2)
	tx2.ParentsHash = []types.Hash{genesis.GetTxHash()}
	pool.AddLocalTx(tx2)

	seq := newTestSeq(0)
	seq.ParentsHash = []types.Hash{
		tx1.GetTxHash(),
		tx2.GetTxHash(),
	}
	err = pool.AddLocalTx(seq)
	if err != nil {
		t.Fatalf("add seq to pool failed: %v", err)
	}
	if pool.Get(seq.GetTxHash()) == nil {
		t.Fatalf("sequencer is not added into pool")
	}
	if status := pool.GetStatus(seq.GetTxHash()); status != core.TxStatusTip {
		t.Fatalf("sequencer's status is not tip but %s after added", status.String())
	}
	if pool.Get(tx1.GetTxHash()) != nil {
		t.Fatalf("tx1 is not removed from pool")
	}
	if pool.Get(tx2.GetTxHash()) != nil {
		t.Fatalf("tx2 is not removed from pool")
	}
	if dag.GetTx(tx1.GetTxHash()) == nil {
		t.Fatalf("tx1 is not stored in dag")
	}
	if dag.GetTx(tx2.GetTxHash()) == nil {
		t.Fatalf("tx2 is not stored in dag")
	}
	if dag.GetTx(seq.GetTxHash()) == nil {
		t.Fatalf("seq is not stored in dag")
	}
	if dag.LatestSequencer().GetTxHash().Cmp(seq.GetTxHash()) != 0 {
		t.Fatalf("latest seq in dag is not the seq we want")
	}

	// // sequencer's parent is bad tx
	// badtx := newTestPoolBadTx()
	// badtx.ParentsHash = []types.Hash{seq.GetTxHash()}
	// pool.AddLocalTx(badtx)

	// badtxseq := newTestSeq(1)
	// badtxseq.ParentsHash = []types.Hash{badtx.GetTxHash()}
	// badtxseq.ContractHashOrder = []types.Hash{badtx.GetTxHash()}
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






