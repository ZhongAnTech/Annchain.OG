package core_test

import (
	"os"
	"testing"
	"io/ioutil"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/core"
)

var (
	testPk0 = "0000000000000000000000000000000000000000000000000000000000000000"
	testAddr0 = "188A3EB3BFD8DA1274C935946CB5765B4225503E"

	testPk1 = "1111111111111111111111111111111111111111111111111111111111111111"
	testAddr1 = "E97BB1E3813CA30F8CFC6A0E8B50047063E893B7"

	testPk2 = "2222222222222222222222222222222222222222222222222222222222222222"
	testAddr2 = "2EC79FEA2B6F64FAD50CD20CF5CC2281E141441E"
)

func newTestLDB() (*ogdb.LevelDB, func()) {
	dirname, err := ioutil.TempDir(os.TempDir(), "ogdb_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}
	db, err := ogdb.NewLevelDB(dirname, 0, 0)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
		os.RemoveAll(dirname)
	}
}

func newTestUnsealTx(nonce uint64) *types.Tx {
	txCreator := &og.TxCreator{
		Signer: &crypto.SignerSecp256k1{},
	}
	pk, _ := crypto.PrivateKeyFromString(testPk0)
	addr := types.HexToAddress(testAddr0)

	tx := txCreator.NewSignedTx(addr, addr, math.NewBigInt(0), nonce, pk)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*types.Tx)
}

func newTestSeq(nonce uint64) *types.Sequencer {
	txCreator := &og.TxCreator{
		Signer: &crypto.SignerSecp256k1{},
	}
	pk, _ := crypto.PrivateKeyFromString(testPk0)

	seq := txCreator.NewSignedSequencer(0, []types.Hash{}, nonce, pk)
	seq.SetHash(seq.CalcTxHash())

	return seq.(*types.Sequencer)
}

func compareTxi(tx1, tx2 types.Txi) bool {
	return tx1.Compare(tx2)
}

func TestTransactionStorage(t *testing.T) {
	t.Parallel()

	db, remove := newTestLDB()
	defer remove()

	var err error
	acc := core.NewAccessor(db)

	// test tx read write
	tx := newTestUnsealTx(0)
	err = acc.WriteTransaction(tx)
	if err != nil {
		t.Fatalf("write tx %s failed: %v", tx.GetTxHash().String(), err)
	}
	txRead := acc.ReadTransaction(tx.GetTxHash())
	if txRead == nil {
		t.Fatalf("cannot read tx %s from db", tx.GetTxHash().String())
	}
	if !compareTxi(tx, txRead) {
		t.Fatalf("the tx from db is not equal to the base tx")
	}
	// test tx delete
	err = acc.DeleteTransaction(tx.GetTxHash())
	if err != nil {
		t.Fatalf("delete tx %s failed: %v", tx.GetTxHash().String(), err)
	}
	txDeleted := acc.ReadTransaction(tx.GetTxHash())
	if txDeleted != nil {
		t.Fatalf("tx %s have not deleted yet", tx.GetTxHash().String())
	}

	// test sequencer read write
	seq := newTestSeq(0)
	err = acc.WriteTransaction(seq)
	if err != nil {
		t.Fatalf("write seq %s failed: %v", seq.GetTxHash().String(), err)
	}
	seqRead := acc.ReadTransaction(seq.GetTxHash())
	if seqRead == nil {
		t.Fatalf("cannot read seq %s from db", seq.GetTxHash().String())
	}
	if !compareTxi(seq, seqRead) {
		t.Fatalf("the seq from db is not equal to the base seq")
	}
}

func TestGenesisStorage(t *testing.T) {
	t.Parallel()

	db, remove := newTestLDB()
	defer remove()

	var err error
	acc := core.NewAccessor(db)

	genesis := newTestSeq(0)
	err = acc.WriteGenesis(genesis)
	if err != nil {
		t.Fatalf("write genesis error: %v", err)
	}
	genesisRead := acc.ReadGenesis()
	if genesisRead == nil {
		t.Fatalf("read genesis error")
	}
	if !compareTxi(genesis, genesisRead) {
		t.Fatalf("genesis initialized is not the same as genesis stored")
	}
}

func TestLatestSeqStorage(t *testing.T) {
	t.Parallel()

	db, remove := newTestLDB()
	defer remove()

	var err error
	acc := core.NewAccessor(db)

	latestSeq := newTestSeq(0)
	err = acc.WriteLatestSequencer(latestSeq)
	if err != nil {
		t.Fatalf("write latest sequencer error: %v", err)
	}
	latestSeqRead := acc.ReadLatestSequencer()
	if latestSeqRead == nil {
		t.Fatalf("read latest sequencer error")
	}
	if !compareTxi(latestSeq, latestSeqRead) {
		t.Fatalf("latest sequencer initialized is not the same as latest sequencer stored")
	}
}

// func TestBalanceStorage(t *testing.T) {
// 	t.Parallel()

// 	db, remove := newTestLDB()
// 	defer remove()

// 	var err error 
// 	acc := core.NewAccessor(db)

// 	addr


// }

// TODO test balance




