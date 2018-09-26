package og

import (
	"testing"
	"github.com/annchain/OG/types"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
)

// A3: Nodes produced by same source must be sequential (tx nonce ++).
func TestA3(t *testing.T) {
	pool := &dummyTxPoolParents{}
	pool.Init()

	addr1 := types.HexToAddress("0x0001")
	addr2 := types.HexToAddress("0x0002")

	root := types.RandomSequencer()
	root.ParentsHash = []types.Hash{}
	root.AccountNonce = 1
	pool.AddRemoteTx(root)

	tx1 := types.RandomTx()
	tx2 := types.RandomTx()
	tx3 := types.RandomTx()
	tx4 := types.RandomTx()
	tx5 := types.RandomTx()
	tx6 := types.RandomTx()
	tx7 := types.RandomTx()
	tx8 := types.RandomTx()
	tx9 := types.RandomTx()
	tx10 := types.RandomTx()
	tx11 := types.RandomTx()

	tx1.ParentsHash = []types.Hash{root.Hash}
	tx1.AccountNonce = 2
	tx1.From = addr1

	tx2.ParentsHash = []types.Hash{tx1.Hash, tx7.Hash}
	tx2.AccountNonce = 0
	tx2.From = addr2

	tx3.ParentsHash = []types.Hash{tx2.Hash, tx8.Hash}
	tx3.AccountNonce = 3
	tx3.From = addr1

	tx4.ParentsHash = []types.Hash{tx3.Hash, tx9.Hash}
	tx4.AccountNonce = 1
	tx4.From = addr2

	tx5.ParentsHash = []types.Hash{tx4.Hash, tx10.Hash}
	tx5.AccountNonce = 2
	tx5.From = addr2

	tx6.ParentsHash = []types.Hash{tx5.Hash}
	tx6.AccountNonce = 4
	tx6.From = addr1

	tx7.ParentsHash = []types.Hash{root.Hash}
	tx7.AccountNonce = 3
	tx7.From = addr2

	tx8.ParentsHash = []types.Hash{tx7.Hash}
	tx8.AccountNonce = 2
	tx8.From = addr1

	tx9.ParentsHash = []types.Hash{tx8.Hash}
	tx9.AccountNonce = 4
	tx9.From = addr1

	tx10.ParentsHash = []types.Hash{tx9.Hash}
	tx10.AccountNonce = 4
	tx10.From = addr2

	tx11.ParentsHash = []types.Hash{tx10.Hash}
	tx11.AccountNonce = 3
	tx11.From = addr1

	pool.AddRemoteTx(root)
	pool.AddRemoteTx(tx1)
	pool.AddRemoteTx(tx2)
	pool.AddRemoteTx(tx3)
	pool.AddRemoteTx(tx4)
	pool.AddRemoteTx(tx5)
	pool.AddRemoteTx(tx6)
	pool.AddRemoteTx(tx7)
	pool.AddRemoteTx(tx8)
	pool.AddRemoteTx(tx9)
	pool.AddRemoteTx(tx10)
	pool.AddRemoteTx(tx11)

	// tx2 is bad, others are good
	verifier := Verifier{
		dag:    nil,
		txPool: pool,
	}

	judge(t, verifier.VerifyGraphStructure(root), false, 0) // have not checked genesis
	judge(t, verifier.VerifyGraphStructure(tx1), false, 1)
	judge(t, verifier.VerifyGraphStructure(tx2), true, 2)
	judge(t, verifier.VerifyGraphStructure(tx3), true, 3)
	judge(t, verifier.VerifyGraphStructure(tx4), true, 4)
	judge(t, verifier.VerifyGraphStructure(tx5), true, 5)
	judge(t, verifier.VerifyGraphStructure(tx6), true, 6)
	judge(t, verifier.VerifyGraphStructure(tx7), false, 7)
	judge(t, verifier.VerifyGraphStructure(tx8), false, 8)
	judge(t, verifier.VerifyGraphStructure(tx9), false, 9)
	judge(t, verifier.VerifyGraphStructure(tx10), true, 10)
	judge(t, verifier.VerifyGraphStructure(tx11), true, 11)

}

func judge(t *testing.T, actual interface{}, want interface{}, id int) {
	assert.Equal(t, actual, want)
	if actual != want {
		logrus.Warnf("Assert %d, expected %t, actual %t.", id, want, actual)
	}
}
