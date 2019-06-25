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
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/p2p/ioperformance"
	math2 "github.com/annchain/OG/vm/eth/common/math"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/annchain/OG/common"

	"github.com/annchain/OG/common/math"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

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
	ArchiveMode        bool
}

//NodeStatus
type NodeStatus struct {
	NodeInfo  *p2p.NodeInfo   `json:"node_info"`
	PeersInfo []*p2p.PeerInfo `json:"peers_info"`
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

//NewTxrequest for RPC request
type NewTxRequest struct {
	Nonce      string `json:"nonce"`
	From       string `json:"from"`
	To         string `json:"to"`
	Value      string `json:"value"`
	Data       string `json:"data"`
	CryptoType string `json:"crypto_type"`
	Signature  string `json:"signature"`
	Pubkey     string `json:"pubkey"`
}

//NewArchiveRequest for RPC request
type NewArchiveRequest struct {
	Data  []byte `json:"data"`
}

//NewAccountRequest for RPC request
type NewAccountRequest struct {
	Algorithm string `json:"algorithm"`
}

func cors(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
}

//Status node status
func (r *RpcController) Status(c *gin.Context) {
	var status NodeStatus
	status.NodeInfo = r.P2pServer.NodeInfo()
	status.PeersInfo = r.P2pServer.PeersInfo()
	cors(c)
	Response(c, http.StatusOK, nil, status)
}

//PeersInfo network information
func (r *RpcController) NetInfo(c *gin.Context) {
	info := r.P2pServer.NodeInfo()
	cors(c)
	Response(c, http.StatusOK, nil, info)
}

//PeersInfo  peers information
func (r *RpcController) PeersInfo(c *gin.Context) {
	peersInfo := r.P2pServer.PeersInfo()
	cors(c)
	Response(c, http.StatusOK, nil, peersInfo)
}

//Query query
func (r *RpcController) Query(c *gin.Context) {
	Response(c, http.StatusOK, nil, "not implemented yet")
}

//Transaction  get  transaction
func (r *RpcController) Transaction(c *gin.Context) {
	hashtr := c.Query("hash")
	hash, err := types.HexStringToHash(hashtr)
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
	switch tx := txi.(type) {
	case *types.Tx:
		Response(c, http.StatusOK, nil, tx)
		return
	case *types.Sequencer:
		Response(c, http.StatusOK, nil, tx)
		return
	case *types.Archive:
		Response(c, http.StatusOK, nil, tx)
		return
	}
	Response(c, http.StatusNotFound, fmt.Errorf("status not found"), nil)
}

//Confirm checks if tx has already been confirmed.
func (r *RpcController) Confirm(c *gin.Context) {
	hashtr := c.Query("hash")
	hash, err := types.HexStringToHash(hashtr)
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

//Transactions query Transactions
func (r *RpcController) Transactions(c *gin.Context) {
	seqId := c.Query("seq_id")
	address := c.Query("address")
	cors(c)
	if address == "" {
		id, err := strconv.Atoi(seqId)
		if err != nil || id < 0 {
			Response(c, http.StatusOK, fmt.Errorf("seq_id format error"), nil)
			return
		}
		if r.Og.Dag.GetHeight() < uint64(id) {
			Response(c, http.StatusOK, fmt.Errorf("txs not found"), nil)
			return
		}
		txs := r.Og.Dag.GetTxisByNumber(uint64(id))
		var txsResponse struct {
			Total int        `json:"total"`
			Txs   types.Txis `json:"txs"`
		}
		txsResponse.Total = len(txs)
		txsResponse.Txs = txs
		Response(c, http.StatusOK, nil, txsResponse)
		return
	} else {
		addr, err := types.StringToAddress(address)
		if err != nil {
			Response(c, http.StatusOK, fmt.Errorf("address format error"), nil)
			return
		}
		txs := r.Og.Dag.GetTxsByAddress(addr)
		var txsResponse struct {
			Total int         `json:"total"`
			Txs   []types.Txi `json:"txs"`
		}
		if len(txs) != 0 {
			txsResponse.Total = len(txs)
			txsResponse.Txs = txs
			Response(c, http.StatusOK, nil, txsResponse)
			return
		}
		Response(c, http.StatusOK, fmt.Errorf("txs not found"), nil)
		return
	}

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

type Tps struct {
	Num     int     `json:"num"`
	TxCount int     `json:"tx_num"`
	Seconds float64 `json:"duration"`
}

func (r *RpcController) getTps() (t *Tps, err error) {
	var tps Tps
	lseq := r.Og.Dag.LatestSequencer()
	if lseq == nil {
		return nil, fmt.Errorf("not found")
	}
	if lseq.Height < 3 {
		return
	}

	var cfs []types.ConfirmTime
	for id := lseq.Height; id > 0 && id > lseq.Height-5; id-- {
		cf := r.Og.Dag.GetConfirmTime(id)
		if cf == nil {
			return nil, fmt.Errorf("db error")
		}
		cfs = append(cfs, *cf)
	}
	var start, end time.Time
	for i, cf := range cfs {
		if i == 0 {
			end, err = time.Parse(time.RFC3339Nano, cf.ConfirmTime)
			if err != nil {
				return nil, err
			}
		}
		if i == len(cfs)-1 {
			start, err = time.Parse(time.RFC3339Nano, cf.ConfirmTime)
			if err != nil {
				return nil, err
			}
		} else {
			tps.TxCount += int(cf.TxNum)
		}
	}

	if !end.After(start) {
		return nil, fmt.Errorf("time server error")
	}
	sub := end.Sub(start)
	sec := sub.Seconds()
	if sec != 0 {
		num := float64(tps.TxCount) / sec
		tps.Num = int(num)
	}
	tps.Seconds = sec
	return &tps, nil
}

func (r *RpcController) Tps(c *gin.Context) {
	cors(c)
	t, err := r.getTps()
	if err != nil {
		Response(c, http.StatusBadRequest, err, nil)
		return
	}
	Response(c, http.StatusOK, nil, t)
	return
}

func (r *RpcController) Sequencer(c *gin.Context) {
	cors(c)
	var sq *types.Sequencer
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
		hash, err := types.HexStringToHash(hashtr)
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
		case *types.Sequencer:
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

func (r *RpcController) BftStatus(c *gin.Context) {
	cors(c)
	s := r.AnnSensus.GetBftStatus()
	Response(c, http.StatusOK, nil, s)
	return
}

func (r *RpcController) GetPoolHashes(c *gin.Context) {
	cors(c)
	s := r.Og.TxPool.GetOrder()
	Response(c, http.StatusOK, nil, s)
	return
}

func (r *RpcController) Validator(c *gin.Context) {
	cors(c)
	Response(c, http.StatusOK, nil, "validator")
	return
}

var archiveId uint32
func getArchiveId()uint32 {
	if archiveId > math2.MaxUint32-1000 {
		archiveId = 10
	}
	return atomic.AddUint32(&archiveId,1)
}

func (r *RpcController) NewArchive(c *gin.Context) {
	var (
		tx    types.Txi
		txReq NewArchiveRequest
	)
	now := time.Now()
	id := getArchiveId()
	if !r.ArchiveMode {
		Response(c, http.StatusBadRequest, fmt.Errorf("not archive mode"), nil)
		return
	}
	err := c.ShouldBindJSON(&txReq)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: %v", err), nil)
		return
	}
	//c.Request.Context()
	if len(txReq.Data) ==0 {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: no data "), nil)
		return
	}
	var buf bytes.Buffer
	buf.Write(txReq.Data)
	//TODO compress data
	logrus.WithField("id ",id).WithField("data  ",string(txReq.Data)).Trace("got archive request")
	tx, err = r.TxCreator.NewArchiveWithSeal(buf.Bytes())
	if err != nil {
		Response(c, http.StatusInternalServerError, fmt.Errorf("new tx failed"), nil)
		return
	}
	logrus.WithField("id ",id).WithField("tx", tx).Debugf("tx generated")
	if !r.SyncerManager.IncrementalSyncer.Enabled {
		Response(c, http.StatusOK, fmt.Errorf("tx is disabled when syncing"), nil)
		return
	}
	//TODO which one is fast
	//r.SyncerManager.IncrementalSyncer.CacheTx(tx)

	r.TxBuffer.ReceivedNewTxChan <- tx
	logrus.WithField("used time ",time.Since(now)).WithField("id ",id).WithField("tx ",tx).Trace("send ok")

	Response(c, http.StatusOK, nil, tx.GetTxHash().Hex())
	return
}

func (r *RpcController) NewTransaction(c *gin.Context) {
	var (
		tx    types.Txi
		txReq NewTxRequest
		sig   crypto.Signature
		pub   crypto.PublicKey
	)

	if r.ArchiveMode {
		Response(c, http.StatusBadRequest, fmt.Errorf("archive mode"), nil)
		return
	}

	err := c.ShouldBindJSON(&txReq)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: %v", err), nil)
		return
	}
	from, err := types.StringToAddress(txReq.From)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("from address format error: %v", err), nil)
		return
	}
	to, err := types.StringToAddress(txReq.To)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("to address format error: %v", err), nil)
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

	data := common.FromHex(txReq.Data)
	if data == nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("data not hex"), nil)
		return
	}

	nonce, err := strconv.ParseUint(txReq.Nonce, 10, 64)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("nonce format error"), nil)
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
	tx, err = r.TxCreator.NewTxWithSeal(from, to, value, data, nonce, pub, sig)
	if err != nil {
		Response(c, http.StatusInternalServerError, fmt.Errorf("new tx failed"), nil)
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
	addr, err := types.StringToAddress(address)
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
	addr, err := types.StringToAddress(address)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("address format err"), nil)
		return
	}
	b := r.Og.Dag.GetBalance(addr)
	Response(c, http.StatusOK, nil, gin.H{
		"address": address,
		"balance": b,
	})
	return
	// c.JSON(http.StatusBadRequest, gin.H{
	// 	"balance": b,
	// })
	// return
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
	Response(c, http.StatusOK, nil, r.AnnSensus.GetInfo())
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
	hash := types.BytesToHash(hashBytes)
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

	addr, err := types.StringToAddress(reqdata.Address)
	if err != nil {
		Response(c, http.StatusBadRequest, err, nil)
		return
	}
	query, err := hex.DecodeString(reqdata.Data)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("can't decode data to bytes"), nil)
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

func (r *RpcController) OgPeersInfo(c *gin.Context) {
	info := r.Og.Manager.Hub.PeersInfo()

	Response(c, http.StatusOK, nil, info)
	return
}

type Monitor struct {
	Port    string     `json:"port"`
	ShortId string     `json:"short_id"`
	Peers   []Peer     `json:"peers,omitempty"`
	SeqId   uint64     `json:"seq_id"`
	Tps     *Tps       `json:"tps"`
	Status  SyncStatus `json:"status"`
}

type Peer struct {
	Addr    string `json:"addr"`
	ShortId string `json:"short_id"`
	Link    bool   `json:"link"`
}

func (r *RpcController) Monitor(c *gin.Context) {
	var m Monitor
	seq := r.Og.Dag.LatestSequencer()
	if seq != nil {
		m.SeqId = seq.Height
	}
	peersinfo := r.Og.Manager.Hub.PeersInfo()
	for _, p := range peersinfo {
		/*
			if p.Network.Inbound {
				addr = p.Network.LocalAddress
			}else {
				addr = p.Network.RemoteAddress
			}
				ipPort :=strings.Split(addr,":")
				if len(ipPort) ==2 {
					m.Peers = append(m.Peers ,ipPort[1])
				}
		*/
		var peer Peer
		peer.Addr = p.Addrs
		peer.ShortId = p.ShortId
		peer.Link = p.Link
		m.Peers = append(m.Peers, peer)
	}
	m.Port = viper.GetString("p2p.port")
	m.ShortId = r.P2pServer.NodeInfo().ShortId
	m.Tps, _ = r.getTps()
	m.Status = r.syncStatus()

	Response(c, http.StatusOK, nil, m)
	return
}

func (r *RpcController) NetIo(c *gin.Context) {
	cors(c)
	type transportData struct {
		*ioperformance.IoDataInfo `json:"transport_data"`
	}
	data := transportData{ioperformance.GetNetPerformance()}
	Response(c, http.StatusOK, nil, data)
	return
}

func (r *RpcController) Debug(c *gin.Context) {
	p := c.Request.URL.Query().Get("f")
	switch p {
	case "1":
		r.NewRequestChan <- types.TxBaseTypeNormal
	case "2":
		r.NewRequestChan <- types.TxBaseTypeSequencer
	case "cc":
		err := r.DebugCreateContract()
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("new contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, nil)
			return
		}
	case "qc":
		ret, err := r.DebugQueryContract()
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("query contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, fmt.Sprintf("%x", ret))
			return
		}
	case "sc":
		param := c.Request.URL.Query().Get("param")
		err := r.DebugSetContract(param)
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("set contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, nil)
			return
		}
	case "callerCreate":
		err := r.DebugCallerCreate()
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("create caller contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, nil)
			return
		}
	case "callerCall":
		value := c.Request.URL.Query().Get("value")
		err := r.DebugCallerCall(value)
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("call caller contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, nil)
			return
		}
	}
}

func (r *RpcController) DebugCreateContract() error {
	from := types.HexToAddress("0x60ce04e6a1cc8887fa5dcd43f87c38be1d41827e")
	to := types.BytesToAddress(nil)
	value := math.NewBigInt(0)
	nonce := uint64(1)

	contractCode := "6060604052341561000f57600080fd5b600a60008190555060006001819055506102078061002e6000396000f300606060405260043610610062576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680631c0f72e11461006b57806360fe47b114610094578063c605f76c146100b7578063e5aa3d5814610145575b34600181905550005b341561007657600080fd5b61007e61016e565b6040518082815260200191505060405180910390f35b341561009f57600080fd5b6100b56004808035906020019091905050610174565b005b34156100c257600080fd5b6100ca61017e565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010a5780820151818401526020810190506100ef565b50505050905090810190601f1680156101375780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015057600080fd5b6101586101c1565b6040518082815260200191505060405180910390f35b60015481565b8060008190555050565b6101866101c7565b6040805190810160405280600a81526020017f68656c6c6f576f726c6400000000000000000000000000000000000000000000815250905090565b60005481565b6020604051908101604052806000815250905600a165627a7a723058208e1bdbeee227900e60082cfcc0e44d400385e8811ae77ac6d7f3b72f630f04170029"
	data, _ := hex.DecodeString(contractCode)

	return r.DebugCreateTxAndSendToBuffer(from, to, value, data, nonce)
}

func (r *RpcController) DebugQueryContract() ([]byte, error) {
	from := types.HexToAddress("0x60ce04e6a1cc8887fa5dcd43f87c38be1d41827e")
	contractAddr := crypto.CreateAddress(from, uint64(1))

	calldata := "e5aa3d58"
	callTx := &types.Tx{}
	callTx.From = from
	callTx.Value = math.NewBigInt(0)
	callTx.To = contractAddr
	callTx.Data, _ = hex.DecodeString(calldata)

	b, _, err := r.Og.Dag.ProcessTransaction(callTx)
	return b, err
}

func (r *RpcController) DebugSetContract(n string) error {
	from := types.HexToAddress("0x60ce04e6a1cc8887fa5dcd43f87c38be1d41827e")
	to := crypto.CreateAddress(from, uint64(1))
	value := math.NewBigInt(0)
	curnonce, err := r.Og.Dag.GetLatestNonce(from)
	if err != nil {
		return err
	}
	nonce := curnonce + 1
	setdata := fmt.Sprintf("60fe47b1%s", n)
	data, err := hex.DecodeString(setdata)
	if err != nil {
		return err
	}

	return r.DebugCreateTxAndSendToBuffer(from, to, value, data, nonce)
}

func (r *RpcController) DebugCallerCreate() error {
	from := types.HexToAddress("0xc18969f0b7e3d192d86e220c44be2f03abdaeda2")
	to := types.BytesToAddress(nil)
	value := math.NewBigInt(0)
	nonce := uint64(1)

	contractCode := "6060604052341561000f57600080fd5b732fd82147682e011063adb8534b2d1d8831f529696000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff16600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610247806100d46000396000f300606060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063671ac9c51461005c578063ceee2e201461007f578063f1850c1b146100d4575b600080fd5b341561006757600080fd5b61007d6004808035906020019091905050610129565b005b341561008a57600080fd5b6100926101d0565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34156100df57600080fd5b6100e76101f5565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166360fe47b1826040518263ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040180828152602001915050600060405180830381600087803b15156101b957600080fd5b6102c65a03f115156101ca57600080fd5b50505050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16815600a165627a7a72305820a1651c4d2b88805e349363d97e5141614ba060f6fd871434b856d5cc689006eb0029"
	data, _ := hex.DecodeString(contractCode)

	return r.DebugCreateTxAndSendToBuffer(from, to, value, data, nonce)
}

func (r *RpcController) DebugCallerCall(setvalue string) error {
	from := types.HexToAddress("0xc18969f0b7e3d192d86e220c44be2f03abdaeda2")
	to := crypto.CreateAddress(from, uint64(1))
	value := math.NewBigInt(0)
	curnonce, err := r.Og.Dag.GetLatestNonce(from)
	if err != nil {
		return err
	}
	nonce := curnonce + 1
	datastr := fmt.Sprintf("671ac9c5%s", setvalue)
	data, err := hex.DecodeString(datastr)
	if err != nil {
		return err
	}

	return r.DebugCreateTxAndSendToBuffer(from, to, value, data, nonce)
}

func (r *RpcController) DebugCreateTxAndSendToBuffer(from, to types.Address, value *math.BigInt, data []byte, nonce uint64) error {
	pubstr := "0x0104a391a55b84e45858748324534a187c60046266b17a5c77161104c4a6c4b1511789de692b898768ff6ee731e2fe068d6234a19a1e20246c968df9c0ca797498e7"
	pub, _ := crypto.PublicKeyFromString(pubstr)

	sigstr := "6060604052341561000f57600080fd5b600a60008190555060006001819055506102078061002e6000396000f300606060405260043610610062576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680631c0f72e11461006b57806360fe47b114610094578063c605f76c146100b7578063e5aa3d5814610145575b34600181905550005b341561007657600080fd5b61007e61016e565b6040518082815260200191505060405180910390f35b341561009f57600080fd5b6100b56004808035906020019091905050610174565b005b34156100c257600080fd5b6100ca61017e565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010a5780820151818401526020810190506100ef565b50505050905090810190601f1680156101375780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015057600080fd5b6101586101c1565b6040518082815260200191505060405180910390f35b60015481565b8060008190555050565b6101866101c7565b6040805190810160405280600a81526020017f68656c6c6f576f726c6400000000000000000000000000000000000000000000815250905090565b60005481565b6020604051908101604052806000815250905600a165627a7a723058208e1bdbeee227900e60082cfcc0e44d400385e8811ae77ac6d7f3b72f630f04170029"
	sigb, _ := hex.DecodeString(sigstr)
	sig := crypto.SignatureFromBytes(crypto.CryptoTypeSecp256k1, sigb)

	tx, err := r.TxCreator.NewTxWithSeal(from, to, value, data, nonce, pub, sig)
	if err != nil {
		return err
	}
	r.TxBuffer.ReceivedNewTxChan <- tx
	return nil
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
		"message": msg,
		"data":    data,
	})
}
