package types

import (
	"fmt"
	"testing"
)

func TestRawTx_Tx(t *testing.T) {
	var data []byte
	var txs Txs
	for i := 0; i < 100; i++ {
		tx := RandomTx()
		data, _ = tx.MarshalMsg(data)
		txs = append(txs, tx)
	}
	fmt.Println(len(data))
	for i := 0; i < 100; i++ {
		var tx Tx
		var err error
		data, err = tx.UnmarshalMsg(data)
		if err != nil {
			t.Fatal(err, i)
		}
		if tx.GetTxHash() != txs[i].GetTxHash() {
			t.Fatal(tx, *txs[i])
		}
	}
	fmt.Println(len(data))

	var txms []TxMarshaler
	for i := 0; i < 100; i++ {
		var txm TxMarshaler
		if i%2 == 0 {
			txm.Txi = RandomSequencer()
		} else {
			txm.Txi = RandomTx()
		}
		data, _ = txm.MarshalMsg(data)
		txms = append(txms, txm)
	}
	fmt.Println(len(data))
	for i := 0; i < 100; i++ {
		var txm TxMarshaler
		var err error
		data, err = txm.UnmarshalMsg(data)
		if err != nil {
			t.Fatal(err, i)
		}
		if txm.GetTxHash() != txms[i].GetTxHash() {
			t.Fatal(txm, txms[i])
		}
	}
	fmt.Println(len(data))
}
