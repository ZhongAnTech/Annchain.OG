package core_test

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
)

var (
	testPk0   = "0000000000000000000000000000000000000000000000000000000000000000"
	testAddr0 = "188A3EB3BFD8DA1274C935946CB5765B4225503E"

	testPk1   = "1111111111111111111111111111111111111111111111111111111111111111"
	testAddr1 = "E97BB1E3813CA30F8CFC6A0E8B50047063E893B7"

	testPk2   = "2222222222222222222222222222222222222222222222222222222222222222"
	testAddr2 = "2EC79FEA2B6F64FAD50CD20CF5CC2281E141441E"
)

func newTestLDB(dirPrefix string) (*ogdb.LevelDB, func()) {
	dirname, err := ioutil.TempDir(os.TempDir(), "ogdb_test_"+dirPrefix+"_")
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
	pk, _ := crypto.PrivateKeyFromString(testPk1)
	addr := types.HexToAddress(testAddr1)

	seq := txCreator.NewSignedSequencer(addr, nonce, []types.Hash{}, nonce, pk)
	seq.SetHash(seq.CalcTxHash())

	return seq.(*types.Sequencer)
}

func compareTxi(tx1, tx2 types.Txi) bool {
	return tx1.Compare(tx2)
}

func TestTransactionStorage(t *testing.T) {
	t.Parallel()

	db, remove := newTestLDB("TestTransactionStorage")
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
	seq := newTestSeq(1)
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

	db, remove := newTestLDB("TestGenesisStorage")
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

	db, remove := newTestLDB("TestLatestSeqStorage")
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

func TestBalanceStorage(t *testing.T) {
	t.Parallel()

	db, remove := newTestLDB("TestBalanceStorage")
	defer remove()

	var err error
	acc := core.NewAccessor(db)
	addr := types.HexToAddress(testAddr0)

	balance := acc.ReadBalance(addr)
	if balance == nil {
		t.Fatalf("read balance failed")
	}
	if balance.Value.Cmp(math.NewBigInt(0).Value) != 0 {
		t.Fatalf("the balance of addr %s is not 0", addr.String())
	}

	newBalance := math.NewBigInt(int64(rand.Intn(10000) + 10000))
	err = acc.SetBalance(addr, newBalance)
	if err != nil {
		t.Fatalf("set new balance failed: %v", err)
	}
	bFromDb := acc.ReadBalance(addr)
	if bFromDb == nil {
		t.Fatalf("read balance after seting failed")
	}
	if bFromDb.Value.Cmp(newBalance.Value) != 0 {
		t.Fatalf("the balance in db is not equal the balance we set")
	}

	addAmount := math.NewBigInt(int64(rand.Intn(10000)))
	err = acc.AddBalance(addr, addAmount)
	if err != nil {
		t.Fatalf("add balance failed")
	}
	bFromDb = acc.ReadBalance(addr)
	if bFromDb == nil {
		t.Fatalf("read balance after adding failed")
	}
	if newBalance.Value.Add(newBalance.Value, addAmount.Value).Cmp(bFromDb.Value) != 0 {
		t.Fatalf("the balance after add is not as expected")
	}

	subAmount := math.NewBigInt(int64(rand.Intn(10000)))
	err = acc.SubBalance(addr, subAmount)
	if err != nil {
		t.Fatalf("sub balance failed")
	}
	bFromDb = acc.ReadBalance(addr)
	if bFromDb == nil {
		t.Fatalf("read balance after sub failed")
	}
	if newBalance.Value.Sub(newBalance.Value, subAmount.Value).Cmp(bFromDb.Value) != 0 {
		t.Fatalf("the balance after sub is not as expected")
	}
}

func TestLatestNonce(t *testing.T) {
	t.Parallel()

	db, remove := newTestLDB("TestLatestNonce")
	defer remove()

	var err error
	acc := core.NewAccessor(db)

	var nonce uint64

	tx0 := newTestUnsealTx(0)
	err = acc.WriteTransaction(tx0)
	if err != nil {
		t.Fatalf("write tx0 %s failed: %v", tx0.GetTxHash().String(), err)
	}
	_, err = acc.ReadAddrLatestNonce(tx0.Sender())
	if (err != nil) && (err.Error() != "not exists") {
		t.Fatalf("first tx0 nonce is not empty")
	}

	tx1 := newTestUnsealTx(1)
	err = acc.WriteTransaction(tx1)
	if err != nil {
		t.Fatalf("write tx1 %s failed: %v", tx1.GetTxHash().String(), err)
	}
	nonce, err = acc.ReadAddrLatestNonce(tx1.Sender())
	if err != nil {
		t.Fatalf("read tx1 nonce failed: %v", err)
	}
	if nonce != uint64(1) {
		t.Fatalf("the nonce in db is not we expected. hope %d but get %d", 1, nonce)
	}

	tx2 := newTestUnsealTx(2)
	err = acc.WriteTransaction(tx2)
	if err != nil {
		t.Fatalf("write tx2 %s failed: %v", tx2.GetTxHash().String(), err)
	}
	nonce, err = acc.ReadAddrLatestNonce(tx2.Sender())
	if err != nil {
		t.Fatalf("read tx2 nonce failed: %v", err)
	}
	if nonce != uint64(2) {
		t.Fatalf("the nonce in db is not we expected. hope %d but get %d", 2, nonce)
	}

	badtx := newTestUnsealTx(1)
	err = acc.WriteTransaction(badtx)
	if err != nil {
		t.Fatalf("write badtx %s failed: %v", badtx.GetTxHash().String(), err)
	}
	nonce, err = acc.ReadAddrLatestNonce(badtx.Sender())
	if err != nil {
		t.Fatalf("read badtx nonce failed: %v", err)
	}
	if nonce != uint64(2) {
		t.Fatalf("the nonce in db is not we expected. hope %d but get %d", 1, nonce)
	}

}
