package core_test

import (
	"testing"

	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
)

func newTestDag(t *testing.T) (*core.Dag, *types.Sequencer, func()) {
	conf := core.DagConfig{}
	db, remove := newTestLDB()
	dag := core.NewDag(conf, db)

	genesis, balance := og.DefaultGenesis()
	err := dag.Init(genesis, balance)
	if err != nil {
		t.Fatalf("init dag failed with error: %v", err)
	}
	dag.Start()

	return dag, genesis, func(){
		dag.Stop()
		remove()
	}
}

func newTestDagTx(nonce uint64) *types.Tx {
	txCreator := &og.TxCreator{
		Signer: &crypto.SignerSecp256k1{},
	}
	pk, _ := crypto.PrivateKeyFromString(testPk0)
	addr := types.HexToAddress(testAddr0)

	tx := txCreator.NewSignedTx(addr, addr, math.NewBigInt(0), nonce, pk)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*types.Tx)
}

func TestDagInit(t *testing.T) {
	t.Parallel()

	dag, genesis, finish := newTestDag(t)
	defer finish()

	if dag.GetTx(genesis.GetTxHash()) == nil {
		t.Fatalf("genesis is not stored in dag db")
	}
	ge := dag.Genesis()
	if ge == nil {
		t.Fatalf("genesis is not set in dag")
	}
	if !ge.Compare(genesis) {
		t.Fatalf("genesis setted in dag is not the genesis we want")
	}
	ls := dag.LatestSequencer()
	if ls == nil {
		t.Fatalf("latest seq is not set in dag")
	}
	if !ls.Compare(genesis) {
		t.Fatalf("latest seq in dag is not the genesis we want")
	}
}

func TestDagLoadGenesis(t *testing.T) {
	t.Parallel()

	var err error

	conf := core.DagConfig{}
	db, remove := newTestLDB()
	defer remove()
	dag := core.NewDag(conf, db)

	acc := core.NewAccessor(db)
	genesis, _ := og.DefaultGenesis()
	err = acc.WriteGenesis(genesis)
	if err != nil {
		t.Fatalf("can't write genesis into db: %v", err)
	}
	if ok := dag.LoadGenesis(); !ok {
		t.Fatalf("can't load genesis from db")
	}
	
	ge := dag.Genesis()
	if ge == nil {
		t.Fatalf("genesis is not set in dag")
	}
	if !ge.Compare(genesis) {
		t.Fatalf("genesis setted in dag is not the genesis we want")
	}
	ls := dag.LatestSequencer()
	if ls == nil {
		t.Fatalf("latest seq is not set in dag")
	}
	if !ls.Compare(genesis) {
		t.Fatalf("latest seq in dag is not the genesis we want")
	}

}

func TestDagPush(t *testing.T) {
	t.Parallel()

	dag, genesis, finish := newTestDag(t)
	defer finish()

	var err error

	tx1 := newTestDagTx(0)
	tx1.ParentsHash = []types.Hash{genesis.GetTxHash()}
	tx2 := newTestDagTx(1)
	tx2.ParentsHash = []types.Hash{genesis.GetTxHash()}

	bd := &core.BatchDetail{TxList: make(map[types.Hash]types.Txi)}
	bd.TxList[tx1.GetTxHash()] = tx1
	bd.TxList[tx2.GetTxHash()] = tx2
	bd.Pos = math.NewBigInt(0)
	bd.Neg = math.NewBigInt(0)

	batch := map[types.Address]*core.BatchDetail{}
	batch[tx1.From] = bd

	seq := newTestSeq(0)
	seq.ParentsHash = []types.Hash{
		tx1.GetTxHash(),
		tx2.GetTxHash(),
	}

	cb := &core.ConfirmBatch{}
	cb.Seq = seq
	cb.Batch = batch

	err = dag.Push(cb)
	if err != nil {
		t.Fatalf("push confirm batch to dag failed: %v", err)
	}
	// check if txs stored into db
	if dag.GetTx(tx1.GetTxHash()) == nil {
		t.Fatalf("tx1 is not stored in dag")
	}
	if dag.GetTx(tx2.GetTxHash()) == nil {
		t.Fatalf("tx2 is not stored in dag")
	}
	// check if seq stored into db
	if dag.GetTx(seq.GetTxHash()) == nil {
		t.Fatalf("seq is not stored in dag")
	}
	if dag.LatestSequencer().GetTxHash() != seq.GetTxHash() {
		t.Fatalf("latest seq is not set")
	}
	// check txs' hashs
	var hashsP *types.Hashs
	hashsP, err = dag.Accessor().ReadIndexedTxHashs(seq.Id)
	hashs := *hashsP
	if err != nil {
		t.Fatalf("read indexed tx hashs failed: %v", err)
	}
	if len(hashs) != 2 {
		t.Fatalf("hashs length not match")
	}
	if !( (hashs[0] == tx1.GetTxHash() && hashs[1] == tx2.GetTxHash()) || 
			(hashs[1] == tx1.GetTxHash() && hashs[0] == tx2.GetTxHash()) ) {
		t.Fatalf("indexed hashs are not the list of tx1 and tx2's hash")
	}


	// TODO check addr balance

}


