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
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/og/txmaker"
	"testing"

	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/core/state"
)

var (
	testAddress01 = "0x0b5d53f433b7e4a4f853a01e987f977497dda261"
	testAddress02 = "0x0b5d53f433b7e4a4f853a01e987f977497dda262"
)

func newTestDag(t *testing.T, dbDirPrefix string) (*core.Dag, *ogmessage.Sequencer, func()) {
	conf := core.DagConfig{}
	db, remove := newTestLDB(dbDirPrefix)
	stdbconf := state.DefaultStateDBConfig()
	dag, errnew := core.NewDag(conf, stdbconf, db, nil)
	if errnew != nil {
		t.Fatalf("new dag failed with error: %v", errnew)
	}

	genesis, balance := core.DefaultGenesis("genesis.json")
	err := dag.Init(genesis, balance)
	if err != nil {
		t.Fatalf("init dag failed with error: %v", err)
	}
	dag.Start()

	return dag, genesis, func() {
		dag.Stop()
		remove()
	}
}

func newTestDagTx(nonce uint64) *ogmessage.Tx {
	txCreator := &txmaker.OGTxCreator{}
	pk, _ := crypto.PrivateKeyFromString(testPkSecp0)
	addr := newTestAddress(pk)

	tx := txCreator.NewSignedTx(txmaker.SignedTxBuildRequest{
		UnsignedTxBuildRequest: txmaker.UnsignedTxBuildRequest{
			From:         addr,
			To:           addr,
			Value:        math.NewBigInt(0),
			AccountNonce: nonce,
			TokenId:      0,
		},
		PrivateKey: pk,
	})
	tx.SetHash(tx.CalcTxHash())

	return tx.(*ogmessage.Tx)
}

func TestDagInit(t *testing.T) {
	t.Parallel()

	dag, genesis, finish := newTestDag(t, "TestDagInit")
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

	conf := core.DagConfig{}
	db, remove := newTestLDB("TestDagLoadGenesis")
	defer remove()
	dag, errnew := core.NewDag(conf, state.DefaultStateDBConfig(), db, nil)
	if errnew != nil {
		t.Fatalf("can't new a dag: %v", errnew)
	}

	acc := core.NewAccessor(db)
	genesis, _ := core.DefaultGenesis("genesis.json")
	err := acc.WriteGenesis(genesis)
	if err != nil {
		t.Fatalf("can't write genesis into db: %v", err)
	}
	if ok, _ := dag.LoadLastState(); !ok {
		t.Fatalf("can't load last state from db")
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

	dag, genesis, finish := newTestDag(t, "TestDagPush")
	defer finish()

	var err error

	tx1 := newTestDagTx(0)
	tx1.ParentsHash = common.Hashes{genesis.GetTxHash()}
	tx2 := newTestDagTx(1)
	tx2.ParentsHash = common.Hashes{genesis.GetTxHash()}

	bd := &core.BatchDetail{TxList: core.NewTxList(), Neg: make(map[int32]*math.BigInt)}
	bd.TxList.Put(tx1)
	bd.TxList.Put(tx2)
	//bd.Pos = math.NewBigInt(0)
	bd.Neg[0] = math.NewBigInt(0)

	batch := map[common.Address]*core.BatchDetail{}
	batch[tx1.Sender()] = bd

	seq := newTestSeq(1)
	seq.ParentsHash = common.Hashes{
		tx1.GetTxHash(),
		tx2.GetTxHash(),
	}

	//hashes := &common.Hashes{tx1.GetTxHash(), tx2.GetTxHash()}

	cb := &core.ConfirmBatch{}
	cb.Seq = seq
	//cb.Batch = batch
	//cb.TxHashes = hashes

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
	var hashsP *common.Hashes
	hashsP, err = dag.Accessor().ReadIndexedTxHashs(seq.Height)
	hashs := *hashsP
	if err != nil {
		t.Fatalf("read indexed tx hashs failed: %v", err)
	}
	if len(hashs) != 2 {
		t.Fatalf("hashs length not match")
	}
	if !((hashs[0] == tx1.GetTxHash() && hashs[1] == tx2.GetTxHash()) ||
		(hashs[1] == tx1.GetTxHash() && hashs[0] == tx2.GetTxHash())) {
		t.Fatalf("indexed hashs are not the list of tx1 and tx2's hash")
	}

	txs := dag.GetTxisByNumber(seq.Height)
	fmt.Println("txs", txs)

	// TODO check addr balance
}

func TestDagProcess(t *testing.T) {
	t.Parallel()

	var ret []byte
	var err error

	dag, _, finish := newTestDag(t, "TestDagProcess")
	stdb := dag.StateDatabase()
	defer finish()

	pk, _ := crypto.PrivateKeyFromString(testPkSecp0)
	addr := newTestAddress(pk)

	// evm contract bytecode, for source code detail please check:
	// github.com/annchain/OG/vm/vm_test/contracts/setter.sol
	contractCode := "6060604052341561000f57600080fd5b600a60008190555060006001819055506102078061002e6000396000f300606060405260043610610062576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680631c0f72e11461006b57806360fe47b114610094578063c605f76c146100b7578063e5aa3d5814610145575b34600181905550005b341561007657600080fd5b61007e61016e565b6040518082815260200191505060405180910390f35b341561009f57600080fd5b6100b56004808035906020019091905050610174565b005b34156100c257600080fd5b6100ca61017e565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010a5780820151818401526020810190506100ef565b50505050905090810190601f1680156101375780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015057600080fd5b6101586101c1565b6040518082815260200191505060405180910390f35b60015481565b8060008190555050565b6101866101c7565b6040805190810160405280600a81526020017f68656c6c6f576f726c6400000000000000000000000000000000000000000000815250905090565b60005481565b6020604051908101604052806000815250905600a165627a7a723058208e1bdbeee227900e60082cfcc0e44d400385e8811ae77ac6d7f3b72f630f04170029"

	createTx := &ogmessage.Tx{}
	createTx.SetSender(addr)
	createTx.Value = math.NewBigInt(0)
	createTx.Data, err = hex.DecodeString(contractCode)
	if err != nil {
		t.Fatalf("decode hex string to bytes error: %v", err)
	}
	_, _, err = dag.ProcessTransaction(createTx, false)
	if err != nil {
		t.Fatalf("error during contract creation: %v", err)
	}
	contractAddr := crypto.CreateAddress(addr, uint64(0))

	cObj := stdb.GetStateObject(contractAddr)
	if cObj == nil {
		t.Fatalf("contract object not initiated in statedb")
	}
	codeInDB := stdb.GetCode(contractAddr)
	if codeInDB == nil {
		t.Fatalf("code not saved in statedb")
	}

	// get i from setter contract
	calldata := "e5aa3d58"
	callTx := &ogmessage.Tx{}
	callTx.SetSender(addr)
	callTx.Value = math.NewBigInt(0)
	callTx.To = contractAddr
	callTx.Data, _ = hex.DecodeString(calldata)
	ret, _, err = dag.ProcessTransaction(callTx, false)
	if err != nil {
		t.Fatalf("error during contract calling: %v", err)
	}
	targetstr := "000000000000000000000000000000000000000000000000000000000000000a"
	retstr := fmt.Sprintf("%x", ret)
	if retstr != targetstr {
		t.Fatalf("the [i] in contract is not 10, should be %s, get %s", targetstr, retstr)
	}

	// set i to be 100
	setdata := "60fe47b10000000000000000000000000000000000000000000000000000000000000064"
	setTx := &ogmessage.Tx{}
	setTx.SetSender(addr)
	setTx.Value = math.NewBigInt(0)
	setTx.To = contractAddr
	setTx.Data, _ = hex.DecodeString(setdata)
	ret, _, err = dag.ProcessTransaction(setTx, false)
	if err != nil {
		t.Fatalf("error during contract setting: %v", err)
	}
	// get i and check if it is changed
	ret, _, err = dag.ProcessTransaction(callTx, false)
	if err != nil {
		t.Fatalf("error during contract calling: %v", err)
	}
	targetstr = "0000000000000000000000000000000000000000000000000000000000000064"
	retstr = fmt.Sprintf("%x", ret)
	if retstr != targetstr {
		t.Fatalf("the [i] in contract is not 100, should be %s, get %s", targetstr, retstr)
	}

	// pay a 10 bill to contract
	transferValue := int64(10)
	payTx := &ogmessage.Tx{}
	setTx.SetSender(addr)
	payTx.Value = math.NewBigInt(transferValue)
	payTx.To = contractAddr
	ret, _, err = dag.ProcessTransaction(payTx, false)
	if err != nil {
		t.Fatalf("error during contract setting: %v", err)
	}
	blc := stdb.GetBalance(contractAddr)
	if blc.GetInt64() != transferValue {
		t.Fatalf("the value is not tranferred to contract, should be: %d, get: %d", transferValue, blc.GetInt64())
	}
}

// Check if the root of pre push and actual push is the same.
func TestDag_PrePush(t *testing.T) {
	//dag, _, finish := newTestMemDag(t)
	//defer finish()
	//
	//addr1 := types.HexToAddress(testAddress01)
	//addr2 := types.HexToAddress(testAddress02)
	//
	//dag.StateDatabase().AddBalance(addr1, math.NewBigInt(10000000))
	//root, err := dag.StateDatabase().Commit()
	//if err != nil {
	//	t.Errorf("statedb commit error: %v", err)
	//}
	//fmt.Println("root 1: ", root)
	//
	//tx1 := types.Tx{}
	//tx1.From = &addr1
	//tx1.To = addr2
	//tx1.AccountNonce = 1
	//tx1.Value = math.NewBigInt(10)
	//tx1.TokenId = token.OGTokenID
	//tx1.Weight = 100
	//tx1.Hash = types.HexToHash("0x010101")
	//
	//seq := types.Sequencer{}
	//seq.AccountNonce = 2
	//seq.ParentsHash = types.Hashes{tx1.Hash}
	//
	//batch := &core.ConfirmBatch{}
	//batch.Seq = &seq
	//batch.Txs = types.Txis{&tx1}
	//
	//if err = dag.Push(batch); err != nil {
	//	t.Errorf("dag push error: %v", err)
	//}

}
