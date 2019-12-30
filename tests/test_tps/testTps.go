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
package main

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/rpc"
	"time"
)

func generateTxrequests(N int) []rpc.NewTxRequest {
	var requests []rpc.NewTxRequest
	oldpub, _ := crypto.Signer.RandomKeyPair()
	to := oldpub.Address().Hex()
	toAdd := oldpub.Address()
	pub, priv := crypto.Signer.RandomKeyPair()
	for i := 1; i < N; i++ {
		from := pub.Address()
		tx := types.Tx{
			TxBase: types.TxBase{
				Type:         types.TxBaseTypeTx,
				AccountNonce: uint64(i),
				PublicKey:    pub.KeyBytes[:],
			},
			From:  &from,
			To:    toAdd,
			Value: math.NewBigInt(0),
		}
		tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).SignatureBytes[:]
		//v:=  og.TxFormatVerifier{}
		//ok:= v.VerifySignature(&tx)
		//target := tx.SignatureTargets()
		//fmt.Println(hexutil.Encode(target))
		//if !ok {
		//	panic("not ok")
		//}
		request := rpc.NewTxRequest{
			Nonce:     "1",
			From:      tx.From.Hex(),
			To:        to,
			Value:     tx.Value.String(),
			Signature: tx.Signature.String(),
			Pubkey:    pub.String(),
		}
		requests = append(requests, request)
	}
	return requests
}

func testTPs() {
	var N = 100000
	var M = 99
	debug = false
	fmt.Println("started ", time.Now())

	var apps []app
	for i := 0; i < M; i++ {
		apps = append(apps, newApp())
	}
	requests := generateTxrequests(N)
	fmt.Println("gen txs ", time.Now(), len(requests))
	for i := 0; i < M; i++ {
		go apps[i].ConsumeQueue()
	}
	for i := range requests {
		j := i % M
		apps[j].requestChan <- &requests[i]
	}
	time.Sleep(time.Second * 100)
	for i := 0; i < M; i++ {
		close(apps[i].quit)
	}
	return
}
