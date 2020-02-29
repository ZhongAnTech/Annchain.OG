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
package ogcore_test

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/og/txmaker"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/ledger"
	"github.com/annchain/OG/ogcore/pool"
	"github.com/sirupsen/logrus"
	"testing"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogcore/state"
	"github.com/annchain/OG/ogdb"
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

func newTestSeq(nonce uint64) *types.Sequencer {
	txCreator := &txmaker.OGTxCreator{}
	pk, _ := crypto.PrivateKeyFromString(testPkSecp1)
	addr := newTestAddress(pk)

	seq := txCreator.NewSignedSequencer(txmaker.SignedSequencerBuildRequest{
		UnsignedSequencerBuildRequest: txmaker.UnsignedSequencerBuildRequest{
			Issuer:       addr,
			Height:       nonce,
			AccountNonce: nonce,
		},
		PrivateKey: pk,
	})
	seq.SetHash(miner.CalcHash(seq))

	return seq.(*types.Sequencer)
}

func newTestTxPool(t *testing.T) (*pool.TxPool, *ledger.Dag, *types.Sequencer, func()) {
	txpoolconfig := pool.TxPoolConfig{
		QueueSize:     100,
		TipsSize:      100,
		ResetDuration: 5,
		TxVerifyTime:  2,
		TxValidTime:   7,
	}
	db := ogdb.NewMemDatabase()

	conf := ledger.DagConfig{GenesisGenerator: &ledger.HardcodeGenesisGenerator{}}

	dag, generatedGenesis, errnew := ledger.NewDag(conf, state.DefaultStateDBConfig(), db, nil)
	if errnew != nil {
		t.Fatalf("new a dag failed with error: %v", errnew)
	}
	pool := &pool.TxPool{
		NodeLogger: debuglog.NodeLogger{
			Logger: logrus.StandardLogger(),
		},
		EventBus: nil,
		Config:   txpoolconfig,
		Dag:      dag,
	}
	pool.InitDefault()

	pool.Init(generatedGenesis)

	//pool.Start()
	dag.Start()

	return pool, dag, generatedGenesis, func() {
		//pool.Stop()
		dag.Stop()
	}
}

func newTestPoolTx(nonce uint64) *types.Tx {
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
	tx.SetHash(miner.CalcHash(tx))

	return tx.(*types.Tx)
}

func newTestPoolBadTx() *types.Tx {
	txCreator := &txmaker.OGTxCreator{}
	pk, _ := crypto.PrivateKeyFromString(testPkSecp2)
	addr := newTestAddress(pk)

	tx := txCreator.NewSignedTx(txmaker.SignedTxBuildRequest{
		UnsignedTxBuildRequest: txmaker.UnsignedTxBuildRequest{
			From:         addr,
			To:           addr,
			Value:        math.NewBigInt(100),
			AccountNonce: 0,
			TokenId:      0,
		},
		PrivateKey: pk,
	})
	tx.SetHash(miner.CalcHash(tx))

	return tx.(*types.Tx)
}

func TestPoolInit(t *testing.T) {
	t.Parallel()

	poolTest, _, genesis, finish := newTestTxPool(t)
	defer finish()

	// check if genesis is the only tip
	tips := poolTest.GetAllTips()
	if len(tips) != 1 {
		t.Fatalf("should have only one tip")
	}
	tip := tips[genesis.GetHash()]
	if tip == nil {
		t.Fatalf("genesis not stored in tips")
	}

	// check if genesis is in txLookUp
	ge := poolTest.Get(genesis.GetHash())
	if ge == nil {
		t.Fatalf("cant get genesis from poolTest.txLookUp")
	}

	// check genesis's status
	status := poolTest.GetStatus(genesis.GetHash())
	if status != pool.TxStatusTip {
		t.Fatalf("genesis's status is not tip but %s", status.String())
	}

}

func TestPoolCommit(t *testing.T) {
	t.Parallel()

	poolTest, _, genesis, finish := newTestTxPool(t)
	defer finish()

	var err error

	// tx0's parent is genesis
	tx0 := newTestPoolTx(0)
	tx0.ParentsHash = common.Hashes{genesis.GetHash()}
	err = poolTest.AddLocalTx(tx0, true)
	if err != nil {
		t.Fatalf("add tx0 to poolTest failed: %v", err)
	}
	if poolTest.Get(tx0.GetHash()) == nil {
		t.Fatalf("tx0 is not added into poolTest")
	}
	if status := poolTest.GetStatus(tx0.GetHash()); status != pool.TxStatusTip {
		t.Fatalf("tx0's status is not tip but %s after commit, addr %s", status.String(), tx0.Sender())
	}
	geInPool := poolTest.Get(genesis.GetHash())
	if geInPool != nil {
		t.Fatalf("parent genesis is not removed from poolTest.")
	}

	// tx1's parent is tx0
	tx1 := newTestPoolTx(1)
	tx1.ParentsHash = common.Hashes{tx0.GetHash()}
	err = poolTest.AddLocalTx(tx1, true)
	if err != nil {
		t.Fatalf("add tx1 to poolTest failed: %v", err)
	}
	if poolTest.Get(tx1.GetHash()) == nil {
		t.Fatalf("tx1 is not added into poolTest")
	}
	if status := poolTest.GetStatus(tx1.GetHash()); status != pool.TxStatusTip {
		t.Fatalf("tx1's status is not tip but %s after commit", status.String())
	}
	if poolTest.Get(tx0.GetHash()) == nil {
		t.Fatalf("tx0 is not in poolTest after added tx1")
	}
	if status := poolTest.GetStatus(tx0.GetHash()); status != pool.TxStatusPending {
		t.Fatalf("tx0's status is not pending but %s after tx1 added", status.String())
	}

	// tx2's parent is genesis which is not in poolTest yet
	tx2 := newTestPoolTx(2)
	tx2.ParentsHash = common.Hashes{genesis.GetHash()}
	err = poolTest.AddLocalTx(tx2, true)
	if err != nil {
		t.Fatalf("add tx2 to poolTest failed: %v", err)
	}
	if poolTest.Get(tx2.GetHash()) == nil {
		t.Fatalf("tx2 is not added into poolTest")
	}
	if status := poolTest.GetStatus(tx2.GetHash()); status != pool.TxStatusTip {
		t.Fatalf("tx2's status is not tip but %s after commit", status.String())
	}

	// TODO bad tx unit test
	// // test bad tx
	// badtx := newTestPoolBadTx()
	// badtx.ParentsHash = common.Hashes{genesis.GetHash()}
	// err = poolTest.AddLocalTx(badtx)
	// if err != nil {
	// 	t.Fatalf("add badtx to poolTest failed: %v", err)
	// }
	// if poolTest.Get(badtx.GetHash()) == nil {
	// 	t.Fatalf("badtx is not added into poolTest")
	// }
	// if status := poolTest.GetStatus(badtx.GetHash()); status != TxStatusBadTx {
	// 	t.Fatalf("badtx's status is not badtx but %s after commit", status.String())
	// }

}

func TestPoolConfirm(t *testing.T) {
	t.Parallel()

	poolTest, dag, genesis, finish := newTestTxPool(t)
	defer finish()

	var err error

	// sequencer's parents are normal txs
	tx0 := newTestPoolTx(0)
	tx0.ParentsHash = common.Hashes{genesis.GetHash()}
	poolTest.AddLocalTx(tx0, true)

	// TODO
	// tx3 := newTestPoolBadTx()
	// poolTest.AddLocalTx(tx3)

	tx1 := newTestPoolTx(1)
	tx1.ParentsHash = common.Hashes{genesis.GetHash()}
	poolTest.AddLocalTx(tx1, true)

	seq := newTestSeq(1)
	seq.ParentsHash = common.Hashes{
		tx0.GetHash(),
		tx1.GetHash(),
	}
	err = poolTest.AddLocalTx(seq, true)
	if err != nil {
		t.Fatalf("add seq to poolTest failed: %v", err)
	}
	if poolTest.Get(seq.GetHash()) == nil {
		t.Fatalf("sequencer is not added into poolTest")
	}
	if status := poolTest.GetStatus(seq.GetHash()); status != pool.TxStatusTip {
		t.Fatalf("sequencer's status is not tip but %s after added", status.String())
	}
	if poolTest.Get(tx0.GetHash()) != nil {
		t.Fatalf("tx0 is not removed from poolTest")
	}
	if poolTest.Get(tx1.GetHash()) != nil {
		t.Fatalf("tx1 is not removed from poolTest")
	}
	if dag.GetTx(tx0.GetHash()) == nil {
		t.Fatalf("tx0 is not stored in dag")
	}
	if dag.GetTx(tx1.GetHash()) == nil {
		t.Fatalf("tx1 is not stored in dag")
	}
	if dag.GetTx(seq.GetHash()) == nil {
		t.Fatalf("seq is not stored in dag")
	}
	if dag.LatestSequencer().GetHash().Cmp(seq.GetHash()) != 0 {
		t.Fatalf("latest seq in dag is not the seq we want")
	}

	// TODO bad tx unit test
	// // sequencer's parent is bad tx
	// badtx := newTestPoolBadTx()
	// badtx.ParentsHash = common.Hashes{seq.GetHash()}
	// poolTest.AddLocalTx(badtx)

	// addr := common.HexToAddress(testAddr2)
	// dag.Accessor().SetBalance(addr, math.NewBigInt(1000))

	// badtxseq := newTestSeq(2)
	// badtxseq.ParentsHash = common.Hashes{badtx.GetHash()}
	// badtxseq.ContractHashOrder = common.Hashes{badtx.GetHash()}
	// err = poolTest.AddLocalTx(badtxseq)
	// if err != nil {
	// 	t.Fatalf("add badtxseq to poolTest failed: %v", err)
	// }
	// if poolTest.Get(badtxseq.GetHash()) == nil {
	// 	t.Fatalf("badtxseq is not added into poolTest")
	// }
	// if status := poolTest.GetStatus(badtxseq.GetHash()); status != TxStatusTip {
	// 	t.Fatalf("badtxseq's status is not tip but %s after added", status.String())
	// }
	// if poolTest.Get(badtx.GetHash()) != nil {
	// 	t.Fatalf("badtx is not removed from poolTest")
	// }
	// if dag.GetTx(badtx.GetHash()) == nil {
	// 	t.Fatalf("badtx is not stored in dag")
	// }
	// if dag.GetTx(badtxseq.GetHash()) == nil {
	// 	t.Fatalf("battxseq is not stored in dag")
	// }
	// if dag.LatestSequencer().GetHash().Cmp(badtxseq.GetHash()) != 0 {
	// 	t.Fatalf("latest seq in dag is not the battxseq we want")
	// }

}
