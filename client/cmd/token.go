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
	"github.com/annchain/OG/client/tx_client"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/spf13/cobra"
)

var (
	tokenCmd = &cobra.Command{
		Use:   "token",
		Short: "token processing",
		Run:   tokenList,
	}
	tokenIPOCmd = &cobra.Command{
		Use:   "ipo",
		Short: "token initial  public offering",
		Run:   tokenIPO,
	}

	tokenSPOCmd = &cobra.Command{
		Use:   "spo",
		Short: "token second  public offering",
		Run:   tokenSPO,
	}
	tokenDestroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "token second  public offering",
		Run:   tokenDestroy,
	}
	tokenTransferCmd = &cobra.Command{
		Use:   "transfer",
		Short: "token transfer",
		Run:   tokenTransfer,
	}
	 tokenName =  "btc"
	 tokenId = int32(0)
	 enableSPO  bool
)

func tokenInit() {
	tokenCmd.AddCommand(tokenIPOCmd, tokenSPOCmd,tokenDestroyCmd,tokenTransferCmd)
	tokenCmd.PersistentFlags().StringVarP(&priv_key, "priv_key", "k", "", "priv_key ***")
	tokenIPOCmd.PersistentFlags().Int64VarP(&value, "value", "v", 0, "value 1")

	tokenSPOCmd.PersistentFlags().Int64VarP(&value, "value", "v", 0, "value 1")
	tokenTransferCmd.PersistentFlags().Int64VarP(&value, "value", "v", 0, "value 1")
	tokenSPOCmd.PersistentFlags().Int32VarP(&tokenId, "token_id", "i", 0, "token_id 1")
	tokenDestroyCmd.PersistentFlags().Int32VarP(&tokenId, "token_id", "i", 0, "token_id 1")
	tokenTransferCmd.PersistentFlags().Int32VarP(&tokenId, "token_id", "i", 0, "token_id 1")
	tokenCmd.PersistentFlags().Uint64VarP(&nonce, "nonce", "n", 0, "nonce 1")
	tokenTransferCmd.PersistentFlags().StringVarP(&to, "to", "t", "", "to 0x***")
	tokenIPOCmd.PersistentFlags().StringVarP(&tokenName, "toke_name", "t", "test_token", "toke_name btc")
	tokenIPOCmd.PersistentFlags().BoolVarP(&enableSPO, "enable_spo", "e", false , "enable_spo true")

}

func tokenIPO(cmd *cobra.Command, args []string) {

	if  priv_key == "" || value < 1 || tokenName =="" {
		cmd.HelpFunc()
	}
	privKey, err := crypto.PrivateKeyFromString(priv_key)
	if err != nil {
		fmt.Println(err)
		return
	}
	txClient:= tx_client.NewTxClient(Host,true)
	requester :=tx_client.NewRequestGenerator(privKey)

	if nonce <= 0 {
		nonce ,err  = txClient.GetNonce(requester.Address())
		if err!=nil {
			fmt.Println(err)
			return
		}
	}
	data :=  requester.TokenPublishing(nonce+1,enableSPO, tokenName, math.NewBigInt(value))
	resp,err := txClient.SendTokenIPO(&data)
	if err!=nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)
}

func tokenSPO(cmd *cobra.Command, args []string) {
	if  priv_key == "" || value < 1 {
		cmd.HelpFunc()
	}
	privKey, err := crypto.PrivateKeyFromString(priv_key)
	if err != nil {
		fmt.Println(err)
		return
	}
	txClient:= tx_client.NewTxClient(Host,true)
	requester :=tx_client.NewRequestGenerator(privKey)

	if nonce <= 0 {
		nonce ,err = txClient.GetNonce(requester.Address())
		if err!=nil {
			fmt.Println(err)
			return
		}
	}
	data :=  requester.SecondPublicOffering(tokenId,nonce+1,math.NewBigInt(value))
	resp,err := txClient.SendTokenSPO(&data)
	if err!=nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)
}

func tokenDestroy(cmd *cobra.Command, args []string) {
	if  priv_key == "" {
		cmd.HelpFunc()
	}
	privKey, err := crypto.PrivateKeyFromString(priv_key)
	if err != nil {
		fmt.Println(err)
		return
	}
	txClient:= tx_client.NewTxClient(Host,true)
	requester :=tx_client.NewRequestGenerator(privKey)

	if nonce <= 0 {
		nonce,err  = txClient.GetNonce(requester.Address())
		if err!=nil {
			fmt.Println(err)
			return
		}
	}
	data :=  requester.TokenDestroy(tokenId,nonce+1)
	resp,err := txClient.SendTokenDestroy(&data)
	if err!=nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)
}

func tokenTransfer(cmd *cobra.Command, args []string) {
	if to == "" || value < 1 || priv_key == "" {
		cmd.HelpFunc()
	}
	toAddr := common.HexToAddress(to)
	privKey, err := crypto.PrivateKeyFromString(priv_key)
	if err != nil {
		fmt.Println(err)
		return
	}
	txClient:= tx_client.NewTxClient(Host,true)
	requester :=tx_client.NewRequestGenerator(privKey)

	if nonce <= 0 {
		nonce ,err = txClient.GetNonce(requester.Address())
		if err!=nil {
			fmt.Println(err)
			return
		}
	}
	data :=  requester.NormalTx(tokenId,nonce+1,toAddr,math.NewBigInt(value))
	resp,err := txClient.SendNormalTx(&data)
	if err!=nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)
}


func tokenList(cmd *cobra.Command, args []string) {
	txClient:= tx_client.NewTxClient(Host,true)
	list,err:=  txClient.GetTokenList()
	if err!=nil {
		fmt.Println(err)
		return
	}
	fmt.Println(list)
}

