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
	"fmt"
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
	// Secp256k1 Address "0x7349f7a6f622378d5fb0e2c16b9d4a3e5237c187"
	testPkSecp0 = "0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F00"

	// Secp256k1 Address "0x96f4ac2f3215b80ea3a6466ebc1f268f6f1d5406"
	testPkSecp1 = "0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F01"

	// Secp256k1 Address "0xa70c8a9485441f6fa2141d550c26d793107d3dea"
	testPkSecp2 = "0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F02"
)

func newTestAddress(priv crypto.PrivateKey) common.Address {
	signer := crypto.NewSigner(priv.Type)
	pubkey := signer.PubKey(priv)
	return signer.Address(pubkey)
}

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
	txCreator := &og.TxCreator{}
	pk, _ := crypto.PrivateKeyFromString(testPkSecp0)
	addr := newTestAddress(pk)

	tx := txCreator.NewSignedTx(addr, addr, math.NewBigInt(0), nonce, pk)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*tx_types.Tx)
}

func newTestSeq(nonce uint64) *tx_types.Sequencer {
	txCreator := &og.TxCreator{}
	pk, _ := crypto.PrivateKeyFromString(testPkSecp1)
	addr := newTestAddress(pk)

	seq := txCreator.NewSignedSequencer(addr, nonce, nonce, pk)
	seq.SetHash(seq.CalcTxHash())

	return seq.(*tx_types.Sequencer)
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
	err = acc.WriteTransaction(nil, tx)
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
	err = acc.WriteTransaction(nil, seq)
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
	err = acc.WriteLatestSequencer(nil, latestSeq)
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
	pk, _ := crypto.PrivateKeyFromString(testPkSecp1)
	addr := newTestAddress(pk)

	balance := acc.ReadBalance(addr)
	if balance == nil {
		t.Fatalf("read balance failed")
	}
	if balance.Value.Cmp(math.NewBigInt(0).Value) != 0 {
		t.Fatalf("the balance of addr %s is not 0", addr.String())
	}

	newBalance := math.NewBigInt(int64(rand.Intn(10000) + 10000))
	err = acc.SetBalance(nil, addr, newBalance)
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
	err = acc.AddBalance(nil, addr, addAmount)
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
	err = acc.SubBalance(nil, addr, subAmount)
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

// Note that nonce is now managed by statedb, so no need to test
// latest nonce in accessor part.

func TestDag_Start(t *testing.T) {
	db, remove := newTestLDB("TestTransactionStorage")
	defer remove()

	acc := core.NewAccessor(db)
	seq := tx_types.RandomSequencer()
	height := seq.Height
	//acc.WriteLatestSequencer(nil,seq)
	batch := acc.NewBatch()
	acc.WriteSequencerByHeight(batch, seq)
	fmt.Println(seq)
	err := batch.Write()
	if err != nil {
		t.Fatal(err)
	}
	readSeq, err := acc.ReadSequencerByHeight(height)
	if err != nil {
		fmt.Println(batch.ValueSize())
		t.Fatal(err)
	}
	if readSeq.Height != height {
		t.Fatal(readSeq, seq)
	}
	fmt.Println(seq, readSeq)
}
