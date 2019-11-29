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
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/types/token"
	"github.com/annchain/OG/types/tx_types"

	"github.com/annchain/OG/common"
	"net/http"
	"strconv"
	"strings"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/og/syncer"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/performance"
	"github.com/annchain/OG/types"
	"github.com/gin-gonic/gin"
)

type RpcController struct {
	P2pServer          *p2p.Server
	Og                 *og.Og
	TxBuffer           *og.TxBuffer
	TxCreator          *og.TxCreator
	SyncerManager      *syncer.SyncManager
	PerformanceMonitor *performance.PerformanceMonitor
	AutoTxCli          AutoTxClient
	NewRequestChan     chan types.TxBaseType
	AnnSensus          *annsensus.AnnSensus
	FormatVerifier     *og.TxFormatVerifier
}

type AutoTxClient interface {
	SetTxIntervalUs(i int)
}

//TxRequester
type TxRequester interface {
	GenerateRequest(from int, to int)
}

//SequenceRequester
type SequenceRequester interface {
	GenerateRequest()
}

//NewArchiveRequest for RPC request
type NewArchiveRequest struct {
	Data []byte `json:"data"`
}

//NewAccountRequest for RPC request
type NewAccountRequest struct {
	Algorithm string `json:"algorithm"`
}

func cors(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
}

//Query query
func (r *RpcController) Query(c *gin.Context) {
	Response(c, http.StatusOK, nil, "not implemented yet")
}

type TransactionResp struct {
	Type        uint8                    `json:"type"`
	Transaction *tx_types.TransactionMsg `json:"transaction"`
	Sequencer   *tx_types.SequencerMsg   `json:"sequencer"`
}

//Transaction  get  transaction
func (r *RpcController) Transaction(c *gin.Context) {
	hashtr := c.Query("hash")
	hash, err := common.HexStringToHash(hashtr)
	cors(c)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("hash format error"), nil)
		return
	}
	txi := r.Og.Dag.GetTx(hash)
	if txi == nil {
		txi = r.Og.TxPool.Get(hash)
	}
	if txi == nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("tx not found"), nil)
		return
	}

	txResp := TransactionResp{}
	txResp.Type = uint8(txi.GetType())
	switch tx := txi.(type) {
	case *tx_types.Tx:
		txMsg := tx.ToJsonMsg()
		txResp.Transaction = &txMsg
		Response(c, http.StatusOK, nil, txResp)
		return
	case *tx_types.Sequencer:
		seqMsg := tx.ToJsonMsg()
		txResp.Sequencer = &seqMsg
		Response(c, http.StatusOK, nil, seqMsg)
		return
		//case *tx_types.Archive:
		//	Response(c, http.StatusOK, nil, tx)
		//	return
		//case *tx_types.Campaign:
		//	Response(c, http.StatusOK, nil, tx)
		//	return
		//case *tx_types.TermChange:
		//	Response(c, http.StatusOK, nil, tx)
		//	return
		//case *tx_types.ActionTx:
		//	Response(c, http.StatusOK, nil, tx)
		//	return
	}

	Response(c, http.StatusNotFound, fmt.Errorf("status not found, only support transaction and sequencer"), nil)
}

//Confirm checks if tx has already been confirmed.
func (r *RpcController) Confirm(c *gin.Context) {
	hashtr := c.Query("hash")
	hash, err := common.HexStringToHash(hashtr)
	cors(c)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("hash format error"), nil)
		return
	}
	txiDag := r.Og.Dag.GetTx(hash)
	txiTxpool := r.Og.TxPool.Get(hash)

	if txiDag == nil && txiTxpool == nil {
		Response(c, http.StatusNotFound, fmt.Errorf("tx not found"), nil)
		return
	}
	if txiTxpool != nil {
		Response(c, http.StatusOK, nil, false)
		return
		// c.JSON(http.StatusNotFound, gin.H{
		// 	"confirm": false,
		// })
	} else {
		Response(c, http.StatusOK, nil, true)
		return
		// c.JSON(http.StatusNotFound, gin.H{
		// 	"confirm": true,
		// })
	}

}

type TxsResponse struct {
	Total int               `json:"total"`
	Txs   []TransactionResp `json:"txs"`
}

//Transactions query Transactions
func (r *RpcController) Transactions(c *gin.Context) {
	heightStr := c.Query("height")
	address := c.Query("address")
	cors(c)

	var txs types.Txis
	if address == "" {
		height, err := strconv.Atoi(heightStr)
		if err != nil || height < 0 {
			Response(c, http.StatusOK, fmt.Errorf("seq_id format error"), nil)
			return
		}
		if r.Og.Dag.GetHeight() < uint64(height) {
			Response(c, http.StatusOK, fmt.Errorf("txs not found"), nil)
			return
		}
		txs = r.Og.Dag.GetTxisByNumber(uint64(height))
	} else {
		addr, err := common.StringToAddress(address)
		if err != nil {
			Response(c, http.StatusOK, fmt.Errorf("address format error"), nil)
			return
		}
		txs = r.Og.Dag.GetTxsByAddress(addr)
	}

	var txsResp TxsResponse
	txsResp.Total = len(txs)
	txsResp.Txs = make([]TransactionResp, 0)
	for _, txi := range txs {
		txResp := TransactionResp{}
		switch tx := txi.(type) {
		case *tx_types.Tx:
			txMsg := tx.ToJsonMsg()

			txResp.Type = uint8(types.TxBaseTypeNormal)
			txResp.Transaction = &txMsg
			break
		}
		txsResp.Txs = append(txsResp.Txs, txResp)
	}

	Response(c, http.StatusOK, nil, txsResp)
}

func (r *RpcController) Genesis(c *gin.Context) {
	cors(c)
	sq := r.Og.Dag.Genesis()
	if sq != nil {
		Response(c, http.StatusOK, nil, sq)
	} else {
		Response(c, http.StatusNotFound, fmt.Errorf("genesis not found"), nil)
	}
	return
}

func (r *RpcController) Sequencer(c *gin.Context) {
	cors(c)
	var sq *tx_types.Sequencer
	hashtr := c.Query("hash")
	seqId := c.Query("seq_id")
	if seqId == "" {
		seqId = c.Query("id")
	}
	if seqId != "" {
		id, err := strconv.Atoi(seqId)
		if err != nil || id < 0 {
			Response(c, http.StatusBadRequest, fmt.Errorf("id format error"), nil)
			return
		}
		sq = r.Og.Dag.GetSequencerByHeight(uint64(id))
		if sq != nil {
			Response(c, http.StatusOK, nil, sq)
			return
		} else {
			Response(c, http.StatusNotFound, fmt.Errorf("sequencer not found"), nil)
			return
		}
	}
	if hashtr == "" {
		sq = r.Og.Dag.LatestSequencer()
		if sq != nil {
			Response(c, http.StatusOK, nil, sq)
			return
		} else {
			Response(c, http.StatusNotFound, fmt.Errorf("sequencer not found"), nil)
			return
		}
	} else {
		hash, err := common.HexStringToHash(hashtr)
		if err != nil {
			Response(c, http.StatusBadRequest, fmt.Errorf("hash format error"), nil)
			return
		}
		txi := r.Og.Dag.GetTx(hash)
		if txi == nil {
			txi = r.Og.TxPool.Get(hash)
		}
		if txi == nil {
			Response(c, http.StatusNotFound, fmt.Errorf("tx not found"), nil)
			return
		}
		switch t := txi.(type) {
		case *tx_types.Sequencer:
			Response(c, http.StatusOK, nil, t)
			return
		default:
			Response(c, http.StatusNotFound, fmt.Errorf("tx not sequencer"), nil)
			return
		}
	}
	Response(c, http.StatusNotFound, fmt.Errorf("not found"), nil)
	return
}

func (r *RpcController) NewAccount(c *gin.Context) {
	var (
		txReq  NewAccountRequest
		signer crypto.ISigner
		err    error
	)
	err = c.ShouldBindJSON(&txReq)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error"), nil)
		return
	}
	algorithm := strings.ToLower(txReq.Algorithm)
	switch algorithm {
	case "ed25519":
		signer = &crypto.SignerEd25519{}
	case "secp256k1":
		signer = &crypto.SignerSecp256k1{}
	}
	pub, priv := signer.RandomKeyPair()
	if err != nil {
		Response(c, http.StatusInternalServerError, fmt.Errorf("Generate account error"), nil)
		return
	}
	Response(c, http.StatusInternalServerError, nil, gin.H{
		"pubkey":  pub.String(),
		"privkey": priv.String(),
	})
	return
}

func (r *RpcController) AutoTx(c *gin.Context) {
	intervalStr := c.Query("interval_us")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 0 {
		Response(c, http.StatusBadRequest, fmt.Errorf("interval format err"), nil)
		return
	}
	r.AutoTxCli.SetTxIntervalUs(interval)

	Response(c, http.StatusOK, nil, nil)
	return
}

func (r *RpcController) QueryNonce(c *gin.Context) {
	address := c.Query("address")
	addr, err := common.StringToAddress(address)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("address format err"), nil)
		return
	}
	noncePool, errPool := r.Og.TxPool.GetLatestNonce(addr)
	nonceDag, errDag := r.Og.Dag.GetLatestNonce(addr)
	var nonce int64
	if errPool != nil && errDag != nil {
		nonce = 0
	} else {
		nonce = int64(noncePool)
		if noncePool < nonceDag {
			nonce = int64(nonceDag)
		}
	}
	Response(c, http.StatusOK, nil, nonce)
	return
}

func (r *RpcController) QueryBalance(c *gin.Context) {
	address := c.Query("address")
	tokenIDStr := c.Query("token_id")
	all := c.Query("all")
	addr, err := common.StringToAddress(address)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("address format err: %v", err), nil)
		return
	}
	var tokenID int32
	if tokenIDStr == "" {
		tokenID = token.OGTokenID
	} else {
		t, err := strconv.Atoi(tokenIDStr)
		if err != nil {
			Response(c, http.StatusBadRequest, fmt.Errorf("tokenID format err: %v", err), nil)
			return
		}
		tokenID = int32(t)
	}
	if all == "true" {
		b := r.Og.Dag.GetAllTokenBalance(addr)
		Response(c, http.StatusOK, nil, gin.H{
			"address": address,
			"balance": b,
		})
		return
	}

	b := r.Og.Dag.GetBalance(addr, tokenID)
	Response(c, http.StatusOK, nil, gin.H{
		"address": address,
		"balance": b,
	})
	return
}

func (r *RpcController) QueryShare(c *gin.Context) {
	Response(c, http.StatusOK, nil, "not implemented yet")
	return
}

func (r *RpcController) ContractPayload(c *gin.Context) {
	Response(c, http.StatusOK, nil, "not implemented yet")
	return
}

func (r *RpcController) ConStatus(c *gin.Context) {
	cors(c)
	if r.AnnSensus != nil {
		Response(c, http.StatusOK, nil, r.AnnSensus.GetInfo())
	} else {
		Response(c, http.StatusOK, nil, nil)
	}

}

func (r *RpcController) ConfirmStatus(c *gin.Context) {
	cors(c)
	Response(c, http.StatusOK, nil, r.Og.TxPool.GetConfirmStatus())
}

type ReceiptResponse struct {
	TxHash          string `json:"tx_hash"`
	Status          int    `json:"status"`
	Result          string `json:"result"`
	ContractAddress string `json:"contract_address"`
}

func (r *RpcController) QueryReceipt(c *gin.Context) {
	hashHex := c.Query("hash")

	hashBytes := common.FromHex(hashHex)
	if hashBytes == nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("hash not hex"), nil)
		return
	}
	hash := common.BytesToHash(hashBytes)
	receipt := r.Og.Dag.GetReceipt(hash)
	if receipt == nil {
		Response(c, http.StatusNotFound, fmt.Errorf("can't find receipt"), nil)
		return
	}

	rr := ReceiptResponse{}
	rr.TxHash = receipt.TxHash.Hex()
	rr.Status = int(receipt.Status)
	rr.Result = receipt.ProcessResult
	rr.ContractAddress = receipt.ContractAddress.Hex()

	Response(c, http.StatusOK, nil, rr)
	return
}

type NewQueryContractReq struct {
	Address string `json:"address"`
	Data    string `json:"data"`
}

func (r *RpcController) QueryContract(c *gin.Context) {
	var (
		reqdata NewQueryContractReq
	)

	err := c.ShouldBindJSON(&reqdata)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: %v", err), nil)
		return
	}

	addr, err := common.StringToAddress(reqdata.Address)
	if err != nil {
		Response(c, http.StatusBadRequest, err, nil)
		return
	}
	query := common.FromHex(reqdata.Data)
	if query == nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("data not hex"), nil)
		return
	}

	ret, err := r.Og.Dag.CallContract(addr, query)
	if err != nil {
		Response(c, http.StatusNotFound, fmt.Errorf("query contract error: %v", err), nil)
		return
	}

	Response(c, http.StatusOK, nil, hex.EncodeToString(ret))
	return
}

func checkError(err error, c *gin.Context, status int, message string) bool {
	if err != nil {
		c.JSON(status, gin.H{
			"error": fmt.Sprintf("%s:%s", err.Error(), message),
		})
		return false
	}
	return true
}

func Response(c *gin.Context, status int, err error, data interface{}) {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	c.JSON(status, gin.H{
		"err":  msg,
		"data": data,
	})
}
