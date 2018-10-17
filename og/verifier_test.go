package og

import (
	"github.com/annchain/OG/types"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
	"testing"
)

func buildTx(from types.Address, accountNonce uint64) *types.Tx {
	tx := types.RandomTx()
	tx.AccountNonce = accountNonce
	tx.From = from
	return tx
}

func buildSeq(from types.Address, accountNonce uint64, id uint64) *types.Sequencer {
	tx := types.RandomSequencer()
	tx.Id = id
	tx.AccountNonce = accountNonce
	tx.Issuer = from
	return tx
}

func setParents(tx types.Txi, parents []types.Txi) {
	tx.GetBase().ParentsHash = []types.Hash{}
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
	pool := &dummyTxPoolParents{}
	pool.Init()
	dag := &dummyDag{}
	dag.init()

	addr1 := types.HexToAddress("0x0001")
	addr2 := types.HexToAddress("0x0002")

	txs := []types.Txi{
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

	setParents(txs[0], []types.Txi{})
	setParents(txs[1], []types.Txi{txs[0]})
	setParents(txs[2], []types.Txi{txs[1], txs[7]})
	setParents(txs[3], []types.Txi{txs[2], txs[8]})
	setParents(txs[4], []types.Txi{txs[3], txs[9]})

	setParents(txs[5], []types.Txi{txs[4], txs[10]})
	setParents(txs[6], []types.Txi{txs[5]})
	setParents(txs[7], []types.Txi{txs[0]})
	setParents(txs[8], []types.Txi{txs[7]})
	setParents(txs[9], []types.Txi{txs[8]})

	setParents(txs[10], []types.Txi{txs[9]})
	setParents(txs[11], []types.Txi{txs[10]})

	for _, tx := range txs {
		pool.AddRemoteTx(tx)
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
	pool := &dummyTxPoolParents{}
	pool.Init()
	dag := &dummyDag{}
	dag.init()

	addr1 := types.HexToAddress("0x0001")
	addr2 := types.HexToAddress("0x0002")
	addr3 := types.HexToAddress("0x0003")

	txs := []types.Txi{
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

	setParents(txs[0], []types.Txi{})
	setParents(txs[1], []types.Txi{txs[0]})
	setParents(txs[2], []types.Txi{txs[1], txs[6]})
	setParents(txs[3], []types.Txi{txs[2], txs[7]})
	setParents(txs[4], []types.Txi{txs[3], txs[8]})

	setParents(txs[5], []types.Txi{txs[4]})
	setParents(txs[6], []types.Txi{txs[0]})
	setParents(txs[7], []types.Txi{txs[2]})
	setParents(txs[8], []types.Txi{txs[7]})

	for _, tx := range txs {
		pool.AddRemoteTx(tx)
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
