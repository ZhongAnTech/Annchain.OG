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
package crypto

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/types/archive"
	"testing"
	"time"
)

func TestRawTx_Tx(t *testing.T) {
	signer := crypto.NewSigner(crypto.CryptoTypeEd25519)
	var num = 10000
	var txs types.Txs
	var rawtxs archive.RawTxs
	for i := 0; i < num; i++ {
		tx := archive.RandomTx()
		pub, _ := signer.RandomKeyPair()
		tx.PublicKey = pub.KeyBytes[:]
		rawtxs = append(rawtxs, tx.RawTx())
	}
	start := time.Now()
	for i := 0; i < num; i++ {
		txs = append(txs, rawtxs[i].Tx())
	}
	fmt.Println("used time ", time.Now().Sub(start))

}

func TestRawTx_encode(t *testing.T) {
	signer := crypto.NewSigner(crypto.CryptoTypeEd25519)
	crypto.Signer = signer
	var num = 10000
	var txs types.Txs
	type bytes struct {
		archive.RawData
	}
	var rawtxs []bytes
	for i := 0; i < num; i++ {
		tx := archive.RandomTx()
		pub, _ := signer.RandomKeyPair()
		tx.PublicKey = pub.KeyBytes[:]
		data, _ := tx.MarshalMsg(nil)
		rawtxs = append(rawtxs, bytes{data})
	}
	start := time.Now()
	for i := 0; i < num; i++ {
		var tx types.Tx
		_, err := tx.UnmarshalMsg(rawtxs[i].RawData)
		if err != nil {
			panic(err)
		}
		txs = append(txs, &tx)
	}
	fmt.Println("used time ", time.Now().Sub(start))

}
