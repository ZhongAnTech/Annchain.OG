package verifier

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og/dummy"
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"

	"github.com/sirupsen/logrus"
	"testing"
)

func buildTx(from common.Address, accountNonce uint64) *archive.Tx {
	tx := archive.RandomTx()
	tx.AccountNonce = accountNonce
	tx.From = &from
	return tx
}

func buildSeq(from common.Address, accountNonce uint64, id uint64) *ogmessage.Sequencer {
	tx := ogmessage.RandomSequencer()
	tx.AccountNonce = accountNonce
	tx.Issuer = &from
	return tx
}

func setParents(tx ogmessage.Txi, parents []ogmessage.Txi) {
	tx.GetBase().ParentsHash = common.Hashes{}
	for _, parent := range parents {
		tx.GetBase().ParentsHash = append(tx.GetBase().ParentsHash, parent.GetTxHash())
	}
}

func judge(t *testing.T, actual interface{}, want interface{}, id int) {
	assert.Equal(t, actual, want)
	if actual != want {
		logrus.Warnf("Assert %d, expected %t, actual %t.", id, want, actual)
	}
}

// A3: Nodes produced by same source must be sequential (tx nonce ++).
func TestA3(t *testing.T) {
	pool := &DummyTxPoolParents{}
	pool.Init()
	dag := &dummyDag{}
	dag.init()

	addr1 := common.HexToAddress("0x0001")
	addr2 := common.HexToAddress("0x0002")

	txs := []ogmessage.Txi{
		buildSeq(addr1, 1, 1),
		buildTx(addr1, 2),
		buildTx(addr2, 0),
		buildTx(addr1, 3),
		buildTx(addr2, 1),

		buildTx(addr2, 2),
		buildTx(addr1, 4),
		buildTx(addr2, 3),
		buildTx(addr1, 2),
		buildTx(addr1, 4),

		buildTx(addr2, 4),
		buildTx(addr1, 3),
	}

	setParents(txs[0], []ogmessage.Txi{})
	setParents(txs[1], []ogmessage.Txi{txs[0]})
	setParents(txs[2], []ogmessage.Txi{txs[1], txs[7]})
	setParents(txs[3], []ogmessage.Txi{txs[2], txs[8]})
	setParents(txs[4], []ogmessage.Txi{txs[3], txs[9]})

	setParents(txs[5], []ogmessage.Txi{txs[4], txs[10]})
	setParents(txs[6], []ogmessage.Txi{txs[5]})
	setParents(txs[7], []ogmessage.Txi{txs[0]})
	setParents(txs[8], []ogmessage.Txi{txs[7]})
	setParents(txs[9], []ogmessage.Txi{txs[8]})

	setParents(txs[10], []ogmessage.Txi{txs[9]})
	setParents(txs[11], []ogmessage.Txi{txs[10]})

	for _, tx := range txs {
		pool.AddRemoteTx(tx, true)
	}

	// tx2 is bad, others are good
	verifier := GraphVerifier{
		Dag:    dag,
		TxPool: pool,
	}

	truth := []int{
		0, 1, 1, 1, 1,
		1, 1, 0, 1, 0,
		1, 1}

	for i, tx := range txs {
		judge(t, verifier.verifyA3(tx), truth[i] == 1, i)
	}

}

// A6: [My job] Node cannot reference two un-ordered nodes as its parents
func TestA6(t *testing.T) {
	pool := &dummy.DummyTxPoolParents{}
	pool.Init()
	dag := &dummyDag{}
	dag.init()

	addr1 := common.HexToAddress("0x0001")
	addr2 := common.HexToAddress("0x0002")
	addr3 := common.HexToAddress("0x0003")

	txs := []ogmessage.Txi{
		buildSeq(addr1, 1, 1),
		buildTx(addr1, 2),
		buildTx(addr1, 0),
		buildTx(addr2, 3),
		buildTx(addr3, 1),

		buildTx(addr2, 2),
		buildTx(addr1, 4),
		buildTx(addr3, 3),
		buildTx(addr2, 2),
	}

	setParents(txs[0], []ogmessage.Txi{})
	setParents(txs[1], []ogmessage.Txi{txs[0]})
	setParents(txs[2], []ogmessage.Txi{txs[1], txs[6]})
	setParents(txs[3], []ogmessage.Txi{txs[2], txs[7]})
	setParents(txs[4], []ogmessage.Txi{txs[3], txs[8]})

	setParents(txs[5], []ogmessage.Txi{txs[4]})
	setParents(txs[6], []ogmessage.Txi{txs[0]})
	setParents(txs[7], []ogmessage.Txi{txs[2]})
	setParents(txs[8], []ogmessage.Txi{txs[7]})

	for _, tx := range txs {
		pool.AddRemoteTx(tx, true)
	}

	// tx2 is bad, others are good
	verifier := GraphVerifier{
		Dag:    dag,
		TxPool: pool,
	}

	truth := []int{
		0, 1, 0, 1, 0,
		1, 1, 1, 1}

	for i, tx := range txs {
		judge(t, verifier.verifyA3(tx), truth[i] == 1, i)
	}

}
