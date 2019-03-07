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
package cmd

import (
	"fmt"
	"github.com/annchain/OG/client/httplib"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/spf13/cobra"
)

var (
	txCmd = &cobra.Command{
		Use:   "tx ",
		Short: "send new transaction",
		Run:   newTx,
	}

	payload string
	to      string
	nonce   uint64
	value   int64
)

func txInit() {
	txCmd.PersistentFlags().StringVarP(&payload, "payload", "p", "", "payload value")
	txCmd.PersistentFlags().StringVarP(&to, "to", "t", "", "to 0x***")
	txCmd.PersistentFlags().StringVarP(&priv_key, "priv_key", "k", "", "priv_key ***")
	txCmd.PersistentFlags().Int64VarP(&value, "value", "v", 0, "value 1")
	txCmd.PersistentFlags().Uint64VarP(&nonce, "nonce", "n", 0, "nonce 1")
}

//NewTxrequest for RPC request
type NewTxRequest struct {
	Nonce     string `json:"nonce"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Signature string `json:"signature"`
	Pubkey    string `json:"pubkey"`
}

func newTx(cmd *cobra.Command, args []string) {
	if to == "" || value < 1 || priv_key == "" {
		cmd.HelpFunc()
	}
	toAddr := types.HexToAddress(to)
	key, err := crypto.PrivateKeyFromString(priv_key)
	if err != nil {
		fmt.Println(err)
		return
	}
	//todo smart contracts
	//data := common.Hex2Bytes(payload)
	// do sign work
	signer := crypto.NewSigner(key.Type)
	pub := signer.PubKey(key)
	from := signer.Address(pub)
	if nonce <= 0 {
		nonce = getNonce(from)
	}
	tx := types.Tx{
		Value: math.NewBigInt(value),
		To:    toAddr,
		From:  from,
		TxBase: types.TxBase{
			AccountNonce: nonce,
			Type:         types.TxBaseTypeNormal,
		},
	}
	signature := signer.Sign(key, tx.SignatureTargets())
	pubKey := signer.PubKey(key)
	txReq := &NewTxRequest{
		Nonce:     fmt.Sprintf("%d", tx.AccountNonce),
		From:      tx.From.Hex(),
		To:        to,
		Value:     tx.Value.String(),
		Signature: hexutil.Encode(signature.Bytes),
		Pubkey:    pubKey.String(),
	}
	req := httplib.Post(Host + "/new_transaction")
	_, err = req.JSONBody(&txReq)
	if err != nil {
		panic(fmt.Errorf("encode tx errror %v", err))
	}
	str, err := req.String()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(str)
}

func getNonce(addr types.Address) (nonce uint64) {
	uri := fmt.Sprintf("query_nonce?address=%s", addr.Hex())
	req := httplib.Get(Host + "/" + uri)
	var nonceResp struct {
		Nonce uint64 `json:"nonce"`
	}
	_, err := req.JSONBody(&nonceResp)
	if err != nil {
		fmt.Println("encode nonce errror ", err)
	}
	return nonceResp.Nonce
}
