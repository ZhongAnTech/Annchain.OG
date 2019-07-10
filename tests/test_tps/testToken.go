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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/rpc"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
)

var txurl = "http://172.28.152.101:8000/new_transaction"
var ipoUrl = "http://172.28.152.101:8000/token/NewPublicOffering"
var spoUrl = "http://172.28.152.101:8000/token/NewSecondOffering"
var destroyUrl = "http://172.28.152.101:8000/token/TokenWithdraw"

var debug bool

func main() {
	debug = true
	a := newApp()
	nonce := 6
	priv, pub, addr := getkey()
	tokenName := "btcdh"
	_ = tokenName
	//request := generateTokenPublishing(priv,pub,addr,nonce,true,tokenName)
	//a.sendTx(&request,0,ipoUrl)
	//request := secondPublicOffering(priv, pub, addr, 2, nonce)
	//a.sendTx(&request, 0, spoUrl)
	//request := transfer(priv, pub, addr, 3, nonce)
	//a.sendTx(&request, 0, txurl)
	request := destroyRequest(priv,pub,addr,2,nonce)
	a.sendTx(&request,0,destroyUrl)
	return
}

func getkey() (priv crypto.PrivateKey, pub crypto.PublicKey, addr common.Address) {
	var err error
	pub, err = crypto.PublicKeyFromString(
		"0x0104c9a6957815922545a5711cf8a12feeb67c32c8e5fd801baf1319a4d87759321abfbf3b2fde27d337982596b108a4224293a1b52ad87bb221a24375bb8c592a70")
	if err != nil {
		panic(err)
	}
	priv, err = crypto.PrivateKeyFromString(
		"0x012afb81be217e411cfa7610cb99c4bbe6db0ea0e515cfe5fd92ecad0d61141d95")
	if err != nil {
		panic(err)
	}
	addr, err = common.StringToAddress("0x1c7de61f817b6a37c5b799190a3a29b8e1e2c781")
	if err != nil {
		panic(err)
	}
	return
}

func generateTokenPublishing(priv crypto.PrivateKey, pub crypto.PublicKey, from common.Address, nonce int, enableSPO bool, tokenName string) rpc.NewPublicOfferingRequest {
	//pub, priv := crypto.Signer.RandomKeyPair()
	//from:= pub.Address()
	fmt.Println(pub.String(), priv.String(), from.String())
	value := math.NewBigInt(8888888)

	tx := tx_types.ActionTx{
		TxBase: types.TxBase{
			Type:         types.TxBaseAction,
			AccountNonce: uint64(nonce),
			PublicKey:    pub.Bytes[:],
		},
		Action: tx_types.ActionTxActionIPO,
		From:   &from,
		ActionData: &tx_types.PublicOffering{
			Value:     value,
			EnableSPO: enableSPO,
			TokenName: tokenName,
		},
	}
	tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes[:]
	v := og.TxFormatVerifier{}
	ok := v.VerifySignature(&tx)
	if !ok {
		target := tx.SignatureTargets()
		fmt.Println(hexutil.Encode(target))
		panic("not ok")
	}
	request := rpc.NewPublicOfferingRequest{
		Nonce:     fmt.Sprintf("%d", nonce),
		From:      tx.From.Hex(),
		Value:     value.String(),
		Signature: tx.Signature.String(),
		Pubkey:    pub.String(),
		Action:    tx_types.ActionTxActionIPO,
		EnableSPO: enableSPO,
		TokenName: tokenName,
	}

	return request
}

func destroyRequest(priv crypto.PrivateKey, pub crypto.PublicKey, from common.Address, tokenId int32, nonce int) rpc.NewPublicOfferingRequest {
	//pub, priv := crypto.Signer.RandomKeyPair()
	//from:= pub.Address()
	fmt.Println(pub.String(), priv.String(), from.String())
	value := math.NewBigInt(0)

	tx := tx_types.ActionTx{
		TxBase: types.TxBase{
			Type:         types.TxBaseAction,
			AccountNonce: uint64(nonce),
			PublicKey:    pub.Bytes[:],
		},
		Action: tx_types.ActionTxActionDestroy,
		From:   &from,
		ActionData: &tx_types.PublicOffering{
			Value: value,
			//EnableSPO:  enableSPO,
			//TokenName: "test_token",
			TokenId: tokenId,
		},
	}
	tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes[:]
	v := og.TxFormatVerifier{}
	ok := v.VerifySignature(&tx)
	if !ok {
		target := tx.SignatureTargets()
		fmt.Println(hexutil.Encode(target))
		panic("not ok")
	}
	request := rpc.NewPublicOfferingRequest{
		Nonce:     fmt.Sprintf("%d", nonce),
		From:      tx.From.Hex(),
		Value:     value.String(),
		Signature: tx.Signature.String(),
		Pubkey:    pub.String(),
		Action:    tx_types.ActionTxActionDestroy,
		//EnableSPO: enableSPO,
		//TokenName: "test_token",
		TokenId: tokenId,
	}

	return request
}

func secondPublicOffering(priv crypto.PrivateKey, pub crypto.PublicKey, from common.Address, tokenId int32, nonce int) rpc.NewPublicOfferingRequest {
	//pub, priv := crypto.Signer.RandomKeyPair()
	//from:= pub.Address()
	fmt.Println(pub.String(), priv.String(), from.String())
	value := math.NewBigInt(100000)

	tx := tx_types.ActionTx{
		TxBase: types.TxBase{
			Type:         types.TxBaseAction,
			AccountNonce: uint64(nonce),
			PublicKey:    pub.Bytes[:],
		},
		Action: tx_types.ActionTxActionSPO,
		From:   &from,
		ActionData: &tx_types.PublicOffering{
			Value: value,
			//EnableSPO: true,
			//TokenName: "test_token",
			TokenId: tokenId,
		},
	}
	tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes[:]
	v := og.TxFormatVerifier{}
	ok := v.VerifySignature(&tx)
	target := tx.SignatureTargets()
	fmt.Println(hexutil.Encode(target))
	if !ok {
		target := tx.SignatureTargets()
		fmt.Println(hexutil.Encode(target))
		panic("not ok")
	}
	request := rpc.NewPublicOfferingRequest{
		Nonce:     fmt.Sprintf("%d", nonce),
		From:      tx.From.Hex(),
		Value:     value.String(),
		Signature: tx.Signature.String(),
		Pubkey:    pub.String(),
		Action:    tx_types.ActionTxActionSPO,
		//EnableSPO: true,
		//TokenName: "test_token",
		TokenId: tokenId,
	}

	return request
}

func transfer(priv crypto.PrivateKey, pub crypto.PublicKey, from common.Address, tokenId int32, nonce int) rpc.NewTxRequest {
	topub, _ := crypto.Signer.RandomKeyPair()
	to := topub.Address()
	fmt.Println(pub.String(), priv.String(), from.String(), to.String())

	tx := tx_types.Tx{
		TxBase: types.TxBase{
			Type:         types.TxBaseTypeNormal,
			PublicKey:    pub.Bytes[:],
			AccountNonce: uint64(nonce),
		},
		From:    &from,
		TokenId: tokenId,
		Value:   math.NewBigInt(66),
		To:      to,
	}
	tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes[:]
	v := og.TxFormatVerifier{}
	ok := v.VerifySignature(&tx)
	target := tx.SignatureTargets()
	fmt.Println(hexutil.Encode(target))
	if !ok {
		target := tx.SignatureTargets()
		fmt.Println(hexutil.Encode(target))
		panic("not ok")
	}
	request := rpc.NewTxRequest{
		Nonce:     fmt.Sprintf("%d", nonce),
		From:      tx.From.Hex(),
		To:        to.String(),
		Value:     tx.Value.String(),
		Signature: tx.Signature.String(),
		Pubkey:    pub.String(),
		TokenId:   tokenId,
	}
	return request
}
