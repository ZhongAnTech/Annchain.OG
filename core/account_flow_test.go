package core_test

import (
	"testing"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
)

func newTestAccountFlowTx(nonce uint64, value *math.BigInt) *types.Tx {
	txCreator := &og.TxCreator{
		Signer: &crypto.SignerSecp256k1{},
	}
	pk, _ := crypto.PrivateKeyFromString(testPk0)
	addr := types.HexToAddress(testAddr0)

	tx := txCreator.NewSignedTx(addr, addr, value, nonce, pk)
	tx.SetHash(tx.CalcTxHash())

	return tx.(*types.Tx)
}

func TestTxList(t *testing.T) {
	t.Parallel()

	testNonces := []uint64{208, 505, 910, 157, 771, 718, 98, 897, 538, 38}

	tl := core.NewTxList()
	for _, nonce := range testNonces {
		tx := newTestAccountFlowTx(nonce, math.NewBigInt(0))
		tl.Put(tx)
	}

	if tl.Len() != len(testNonces) {
		t.Fatalf("txlist's length not equal the number of inserted txs")
	}
	for _, nonce := range testNonces {
		tx := tl.Get(nonce)
		if tx == nil {
			t.Fatalf("can't get tx from txlist, nonce: %d", nonce)
		}
		if tx.GetNonce() != nonce {
			t.Fatalf("nonce not same, expect %d but get %d", nonce, tx.GetNonce())
		}
	}
	for _, nonce := range testNonces {
		if !tl.Remove(nonce) {
			t.Fatalf("remove tx from txlist failed, nonce %d", nonce)
		}
		if tl.Get(nonce) != nil {
			t.Fatalf("still get tx from txlist after removed, nonce %d", nonce)
		}
	}
}

func TestBalanceState(t *testing.T) {
	t.Parallel()

	var err error
	var spent int64

	originBalance := math.NewBigInt(100000)
	bs := core.NewBalanceState(originBalance)

	fstValue := int64(10000)
	subValue := math.NewBigInt(fstValue)
	err = bs.TrySubBalance(subValue)
	if err != nil {
		t.Fatalf("TrySubBalance err: %v", err)
	}
	spent = bs.Spent().GetInt64()
	if spent != fstValue {
		t.Fatalf("the value of spent is not correct, expect %d, get %d", fstValue, spent)
	}

	tx0value := int64(1000)
	tx1value := int64(2000)
	tx2value := int64(3000)
	tx0 := newTestAccountFlowTx(0, math.NewBigInt(tx0value))
	tx1 := newTestAccountFlowTx(1, math.NewBigInt(tx1value))
	tx2 := newTestAccountFlowTx(2, math.NewBigInt(tx2value))

	err = bs.TryRemoveTx(tx0)
	if err != nil {
		t.Fatalf("TryRemoveTx tx0 error: %v", err)
	}
	spent = bs.Spent().GetInt64()
	if spent != (fstValue - tx0value) {
		t.Fatalf("the value of spent is not correct, expect %d, get %d", fstValue-tx0value, spent)
	}
	err = bs.TryRemoveTx(tx1)
	if err != nil {
		t.Fatalf("TryRemoveTx tx1 error: %v", err)
	}
	spent = bs.Spent().GetInt64()
	if spent != (fstValue - tx0value - tx1value) {
		t.Fatalf("the value of spent is not correct, expect %d, get %d", fstValue-tx0value-tx1value, spent)
	}
	err = bs.TryRemoveTx(tx2)
	if err != nil {
		t.Fatalf("TryRemoveTx tx2 error: %v", err)
	}
	spent = bs.Spent().GetInt64()
	if spent != (fstValue - tx0value - tx1value - tx2value) {
		t.Fatalf("the value of spent is not correct, expect %d, get %d", fstValue-tx0value-tx1value-tx2value, spent)
	}

}

// func TestNonceHeap(t *testing.T) {
// 	testList := []uint64{ 208, 505, 910, 157, 771, 718, 98, 897, 538, 38 }

// 	nh := make(nonceHeap)
// 	for _, v := range testList {

// 	}

// }
