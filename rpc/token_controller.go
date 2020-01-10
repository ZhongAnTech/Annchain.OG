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
package rpc

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/status"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type NewPublicOfferingRequest struct {
	Nonce      uint64 `json:"nonce"`
	From       string `json:"from"`
	Value      string `json:"value"`
	Action     uint8  `json:"action"`
	EnableSPO  bool   `json:"enable_spo"`
	CryptoType string `json:"crypto_type"`
	Signature  string `json:"signature"`
	Pubkey     string `json:"pubkey"`
	TokenId    int32  `json:"token_id"`
	TokenName  string `json:"token_name"`
}

//todo optimize later
func (r *RpcController) TokenDestroy(c *gin.Context) {
	var (
		tx    types.Txi
		txReq NewPublicOfferingRequest
		sig   crypto.Signature
		pub   crypto.PublicKey
	)

	if status.ArchiveMode {
		Response(c, http.StatusBadRequest, fmt.Errorf("archive mode"), nil)
		return
	}

	err := c.ShouldBindJSON(&txReq)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: %v", err), nil)
		return
	}
	from, err := common.StringToAddress(txReq.From)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("from address format error: %v", err), nil)
		return
	}

	//nonce, err := strconv.ParseUint(txReq.Nonce, 10, 64)
	//if err != nil {
	//	Response(c, http.StatusBadRequest, fmt.Errorf("nonce format error"), nil)
	//	return
	//}

	signature := common.FromHex(txReq.Signature)
	if signature == nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("signature format error"), nil)
		return
	}

	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("action  format error: %v", err), nil)
		return
	}

	if txReq.CryptoType == "" {
		pub, err = crypto.PublicKeyFromString(txReq.Pubkey)
		if err != nil {
			Response(c, http.StatusBadRequest, fmt.Errorf("pubkey format error %v", err), nil)
			return
		}
	} else {

		pub, err = crypto.PublicKeyFromStringWithCryptoType(txReq.CryptoType, txReq.Pubkey)
		if err != nil {
			Response(c, http.StatusBadRequest, fmt.Errorf("pubkey format error %v", err), nil)
			return
		}
	}

	sig = crypto.SignatureFromBytes(pub.Type, signature)
	if sig.Type != crypto.Signer.GetCryptoType() || pub.Type != crypto.Signer.GetCryptoType() {
		Response(c, http.StatusOK, fmt.Errorf("crypto algorithm mismatch"), nil)
		return
	}

	fmt.Println(fmt.Sprintf("tx req action: %x", txReq.Action))
	tx, err = r.TxCreator.NewActionTxWithSeal(from, common.Address{}, math.NewBigInt(0), txReq.Action, txReq.Nonce,
		txReq.EnableSPO, txReq.TokenId, txReq.TokenName, pub, sig)
	if err != nil {
		Response(c, http.StatusInternalServerError, fmt.Errorf("new tx failed %v", err), nil)
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	if !r.SyncerManager.IncrementalSyncer.Enabled {
		Response(c, http.StatusOK, fmt.Errorf("tx is disabled when syncing"), nil)
		return
	}

	r.TxBuffer.ReceivedNewTxChan <- tx

	Response(c, http.StatusOK, nil, tx.GetTxHash().Hex())
	return
}

func (r *RpcController) NewPublicOffering(c *gin.Context) {
	var (
		tx    types.Txi
		txReq NewPublicOfferingRequest
		sig   crypto.Signature
		pub   crypto.PublicKey
	)

	if status.ArchiveMode {
		Response(c, http.StatusBadRequest, fmt.Errorf("archive mode"), nil)
		return
	}

	err := c.ShouldBindJSON(&txReq)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: %v", err), nil)
		return
	}
	from, err := common.StringToAddress(txReq.From)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("from address format error: %v", err), nil)
		return
	}

	value, ok := math.NewBigIntFromString(txReq.Value, 10)
	if !ok {
		err = fmt.Errorf("new Big Int error")
	}
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("value format error: %v", err), nil)
		return
	}

	signature := common.FromHex(txReq.Signature)
	if signature == nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("signature format error"), nil)
		return
	}

	if txReq.CryptoType == "" {
		pub, err = crypto.PublicKeyFromString(txReq.Pubkey)
		if err != nil {
			Response(c, http.StatusBadRequest, fmt.Errorf("pubkey format error %v", err), nil)
			return
		}
	} else {

		pub, err = crypto.PublicKeyFromStringWithCryptoType(txReq.CryptoType, txReq.Pubkey)
		if err != nil {
			Response(c, http.StatusBadRequest, fmt.Errorf("pubkey format error %v", err), nil)
			return
		}
	}

	sig = crypto.SignatureFromBytes(pub.Type, signature)
	if sig.Type != crypto.Signer.GetCryptoType() || pub.Type != crypto.Signer.GetCryptoType() {
		Response(c, http.StatusOK, fmt.Errorf("crypto algorithm mismatch"), nil)
		return
	}

	tx, err = r.TxCreator.NewActionTxWithSeal(from, common.Address{}, value, txReq.Action, txReq.Nonce,
		txReq.EnableSPO, 0, txReq.TokenName, pub, sig)
	if err != nil {
		Response(c, http.StatusInternalServerError, fmt.Errorf("new tx failed %v", err), nil)
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	if !r.SyncerManager.IncrementalSyncer.Enabled {
		Response(c, http.StatusOK, fmt.Errorf("tx is disabled when syncing"), nil)
		return
	}

	r.TxBuffer.ReceivedNewTxChan <- tx

	Response(c, http.StatusOK, nil, tx.GetTxHash().Hex())
	return
}

func (r *RpcController) NewSecondOffering(c *gin.Context) {
	var (
		tx    types.Txi
		txReq NewPublicOfferingRequest
		sig   crypto.Signature
		pub   crypto.PublicKey
	)

	if status.ArchiveMode {
		Response(c, http.StatusBadRequest, fmt.Errorf("archive mode"), nil)
		return
	}

	err := c.ShouldBindJSON(&txReq)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: %v", err), nil)
		return
	}
	from, err := common.StringToAddress(txReq.From)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("from address format error: %v", err), nil)
		return
	}

	value, ok := math.NewBigIntFromString(txReq.Value, 10)
	if !ok {
		err = fmt.Errorf("new Big Int error")
	}
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("value format error: %v", err), nil)
		return
	}

	signature := common.FromHex(txReq.Signature)
	if signature == nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("signature format error"), nil)
		return
	}

	if txReq.CryptoType == "" {
		pub, err = crypto.PublicKeyFromString(txReq.Pubkey)
		if err != nil {
			Response(c, http.StatusBadRequest, fmt.Errorf("pubkey format error %v", err), nil)
			return
		}
	} else {
		pub, err = crypto.PublicKeyFromStringWithCryptoType(txReq.CryptoType, txReq.Pubkey)
		if err != nil {
			Response(c, http.StatusBadRequest, fmt.Errorf("pubkey format error %v", err), nil)
			return
		}
	}

	sig = crypto.SignatureFromBytes(pub.Type, signature)
	if sig.Type != crypto.Signer.GetCryptoType() || pub.Type != crypto.Signer.GetCryptoType() {
		Response(c, http.StatusOK, fmt.Errorf("crypto algorithm mismatch"), nil)
		return
	}

	tx, err = r.TxCreator.NewActionTxWithSeal(from, common.Address{}, value, tx_types.ActionTxActionSPO,
		txReq.Nonce, txReq.EnableSPO, txReq.TokenId, txReq.TokenName, pub, sig)
	if err != nil {
		Response(c, http.StatusInternalServerError, fmt.Errorf("new tx failed %v", err), nil)
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	if !r.SyncerManager.IncrementalSyncer.Enabled {
		Response(c, http.StatusOK, fmt.Errorf("tx is disabled when syncing"), nil)
		return
	}

	r.TxBuffer.ReceivedNewTxChan <- tx

	Response(c, http.StatusOK, nil, tx.GetTxHash().Hex())
	return
}

func (r *RpcController) LatestTokenId(c *gin.Context) {
	tokenId := r.Og.Dag.GetLatestTokenId()
	Response(c, http.StatusOK, nil, tokenId)
}

func (r *RpcController) Tokens(c *gin.Context) {
	tokens := r.Og.Dag.GetTokens()
	Response(c, http.StatusOK, nil, tokens)
}

type TokenResponse struct {
	TokenID    int32    `json:"token_id"`
	Name       string   `json:"name"`
	Symbol     string   `json:"symbol"`
	Issuer     string   `json:"issuer"`
	ReIssuable bool     `json:"re_issuable"`
	Issues     []string `json:"issues"`
	Destroyed  bool     `json:"destroyed"`
}

func newTokenRespFromTokenObj(token *state.TokenObject) TokenResponse {
	tokenResp := TokenResponse{}
	tokenResp.TokenID = token.TokenID
	tokenResp.Name = token.Name
	tokenResp.Symbol = token.Symbol
	tokenResp.Issuer = token.Issuer.Hex()
	tokenResp.ReIssuable = token.ReIssuable
	tokenResp.Destroyed = token.Destroyed

	tokenResp.Issues = make([]string, 0)
	for _, issue := range token.Issues {
		tokenResp.Issues = append(tokenResp.Issues, issue.String())
	}

	return tokenResp
}

func (r *RpcController) GetToken(c *gin.Context) {
	str := c.Query("id")
	tokenId, err := strconv.Atoi(str)
	if err != nil {
		Response(c, http.StatusBadRequest, err, nil)
		return
	}
	token := r.Og.Dag.GetToken(int32(tokenId))
	tokenResp := newTokenRespFromTokenObj(token)

	Response(c, http.StatusOK, nil, tokenResp)
}
