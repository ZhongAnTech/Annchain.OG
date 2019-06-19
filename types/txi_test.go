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

	var txms []RawTxMarshaler
	for i := 0; i < 100; i++ {
		var txm RawTxMarshaler
		if i%2 == 0 {
			txm.RawTxi = RandomSequencer().RawTxi()
		} else {
			txm.RawTxi = RandomTx().RawTxi()
		}
		data, _ = txm.MarshalMsg(data)
		txms = append(txms, txm)
	}
	fmt.Println(len(data))
	for i := 0; i < 100; i++ {
		var txm RawTxMarshaler
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
