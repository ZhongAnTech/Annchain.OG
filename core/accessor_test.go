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
	testPk = "6f6720697320746865206265737420636861696e000000000000000000000000"
	testAddr = "c621b18aa1263ee747b1af41a4eb27647dc8662c"
)

func newTestLDB() (*ogdb.LevelDB, func()) {
	dirname, err := ioutil.TempDir(os.TempDir(), "ethdb_test_")
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

func compareTxi(tx1, tx2 types.Txi) bool {
	return tx1.Compare(tx2)
}

func TestReadTransaction(t *testing.T) {
	t.Parallel()

	db, remove := newTestLDB()
	defer remove()

	var err error
	acc := core.NewAccessor(db)
	txCreator := &og.TxCreator{
		Signer: &crypto.SignerSecp256k1{},
	}
	pk, _ := crypto.PrivateKeyFromString(testPk)
	addr := types.HexToAddress(testAddr)

	tx := txCreator.NewSignedTx(addr, addr, math.NewBigInt(0), 0, pk)
	tx.SetHash(tx.CalcTxHash())
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

	seq := txCreator.NewSignedSequencer(0, []types.Hash{}, 0, pk)
	seq.SetHash(seq.CalcTxHash())
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

// TODO
