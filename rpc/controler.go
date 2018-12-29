package rpc

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/annchain/OG/common/hexutil"
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
}

//NodeStatus
type NodeStatus struct {
	NodeInfo  *p2p.NodeInfo   `json:"node_info"`
	PeersInfo []*p2p.PeerInfo `json:"peers_info"`
}

type AutoTxClient interface {
	SetTxIntervalMs(i int)
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
	Nonce     string `json:"nonce"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Data      string `json:"data"`
	Signature string `json:"signature"`
	Pubkey    string `json:"pubkey"`
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
	c.JSON(http.StatusOK, status)
}

//PeersInfo network information
func (r *RpcController) NetInfo(c *gin.Context) {
	info := r.P2pServer.NodeInfo()
	cors(c)
	c.JSON(http.StatusOK, info)
}

//PeersInfo  peers information
func (r *RpcController) PeersInfo(c *gin.Context) {
	peersInfo := r.P2pServer.PeersInfo()
	cors(c)
	c.JSON(http.StatusOK, peersInfo)
}

//Query query
func (r *RpcController) Query(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

//Transaction  get  transaction
func (r *RpcController) Transaction(c *gin.Context) {
	hashtr := c.Query("hash")
	hash, err := types.HexStringToHash(hashtr)
	cors(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "hash format error",
		})
		return
	}
	txi := r.Og.Dag.GetTx(hash)
	if txi == nil {
		txi = r.Og.TxPool.Get(hash)
	}
	if txi == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "not found",
		})
		return
	}
	switch tx := txi.(type) {
	case *types.Tx:
		c.JSON(http.StatusOK, tx)
		return
	case *types.Sequencer:
		c.JSON(http.StatusOK, tx)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{
		"message": "not found",
	})

}

//Transaction  get  transaction
func (r *RpcController) Confirm(c *gin.Context) {
	hashtr := c.Query("hash")
	hash, err := types.HexStringToHash(hashtr)
	cors(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "hash format error",
		})
		return
	}
	txiDag := r.Og.Dag.GetTx(hash)
	txiTxpool := r.Og.TxPool.Get(hash)

	if txiDag == nil && txiTxpool == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "not found",
		})
		return
	}
	if txiTxpool != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"confirm": false,
		})
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"confirm": true,
		})
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
			c.JSON(http.StatusOK, gin.H{
				"message": "seq_id format error",
			})
			return
		}
		txs := r.Og.Dag.GetTxsByNumber(uint64(id))
		var txsREsponse struct {
			Total int         `json:"total"`
			Txs   []*types.Tx `json:"txs"`
		}
		if len(txs) != 0 {
			txsREsponse.Total = len(txs)
			txsREsponse.Txs = txs
			c.JSON(http.StatusOK, txsREsponse)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{
			"message": "not found",
		})
	} else {
		addr, err := types.StringToAddress(address)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "address format error",
			})
			return
		}
		txs := r.Og.Dag.GetTxsByAddress(addr)
		var txsREsponse struct {
			Total int         `json:"total"`
			Txs   []types.Txi `json:"txs"`
		}
		if len(txs) != 0 {
			txsREsponse.Total = len(txs)
			txsREsponse.Txs = txs
			c.JSON(http.StatusOK, txsREsponse)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{
			"message": "not found",
		})
	}

}

func (r *RpcController) Genesis(c *gin.Context) {
	cors(c)
	sq := r.Og.Dag.Genesis()
	if sq != nil {
		c.JSON(http.StatusOK, sq)
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "not found",
		})
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
	if lseq.Id < 3 {
		return
	}

	var cfs []types.ConfirmTime
	for id := lseq.Id; id > 0 && id > lseq.Id-5; id-- {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
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
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "id format error",
			})
			return
		}
		sq = r.Og.Dag.GetSequencerById(uint64(id))
		if sq != nil {
			c.JSON(http.StatusOK, sq)
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "not found",
			})
		}
		return
	}
	if hashtr == "" {
		sq = r.Og.Dag.LatestSequencer()
		if sq != nil {
			c.JSON(http.StatusOK, sq)
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "not found",
			})
		}
		return
	} else {
		hash, err := types.HexStringToHash(hashtr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "hash format error",
			})
			return
		}
		txi := r.Og.Dag.GetTx(hash)
		if txi == nil {
			txi = r.Og.TxPool.Get(hash)
		}
		if txi == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "not found",
			})
			return
		}
		sq := txi.(*types.Sequencer)
		if sq != nil {
			c.JSON(http.StatusOK, sq)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": "not found",
	})
}
func (r *RpcController) Validator(c *gin.Context) {
	cors(c)
	c.JSON(http.StatusOK, gin.H{
		"message": "validator",
	})
}
func (r *RpcController) NewTransaction(c *gin.Context) {
	var (
		tx    types.Txi
		txReq NewTxRequest
		sig   crypto.Signature
		pub   crypto.PublicKey
	)

	err := c.ShouldBindJSON(&txReq)
	if !checkError(err, c, http.StatusBadRequest, "request format error") {
		return
	}
	from, err := types.StringToAddress(txReq.From)
	if !checkError(err, c, http.StatusBadRequest, "address format error") {
		return
	}

	to, err := types.StringToAddress(txReq.To)
	if !checkError(err, c, http.StatusBadRequest, "address format error") {
		return
	}

	value, ok := math.NewBigIntFromString(txReq.Value, 10)
	if !ok {
		err = fmt.Errorf("new Big Int error")
	}
	if !checkError(err, c, http.StatusBadRequest, "value format error") {
		return
	}

	var data []byte
	if txReq.Data == "" {
		data = []byte{}
	} else {
		data, err = hex.DecodeString(txReq.Data)
		if !checkError(err, c, http.StatusBadRequest, "data format error") {
			return
		}
	}

	nonce, err := strconv.ParseUint(txReq.Nonce, 10, 64)
	if !checkError(err, c, http.StatusBadRequest, "nonce format error") {
		return
	}

	signature, err := hexutil.Decode(txReq.Signature)
	if !checkError(err, c, http.StatusBadRequest, "signature format error") {
		return
	}

	pub, err = crypto.PublicKeyFromString(txReq.Pubkey)
	if !checkError(err, c, http.StatusBadRequest, "pubkey format error") {
		return
	}

	sig = crypto.SignatureFromBytes(pub.Type, signature)
	if sig.Type != r.TxCreator.Signer.GetCryptoType() || pub.Type != r.TxCreator.Signer.GetCryptoType() {
		c.JSON(http.StatusOK, gin.H{
			"error": "crypto algorithm mismatch",
		})
		return
	}
	tx, err = r.TxCreator.NewTxWithSeal(from, to, value, data, nonce, pub, sig)
	if !checkError(err, c, http.StatusInternalServerError, "new tx failed") {
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	if !r.SyncerManager.IncrementalSyncer.Enabled {
		c.JSON(http.StatusOK, gin.H{
			"error": "tx is disabled when syncing",
		})
		return
	}

	r.TxBuffer.ReceivedNewTxChan <- tx
	// <-ffchan.NewTimeoutSenderShort(r.TxBuffer.ReceivedNewTxChan, tx, "rpcNewTx").C

	//todo add transaction
	c.JSON(http.StatusOK, gin.H{
		"hash": tx.GetTxHash().Hex(),
	})
}
func (r *RpcController) NewAccount(c *gin.Context) {
	var (
		txReq  NewAccountRequest
		signer crypto.Signer
		err    error
	)
	err = c.ShouldBindJSON(&txReq)
	if !checkError(err, c, http.StatusBadRequest, "request format error") {
		return
	}
	algorithm := strings.ToLower(txReq.Algorithm)
	switch algorithm {
	case "ed25519":
		signer = &crypto.SignerEd25519{}
	case "secp256k1":
		signer = &crypto.SignerSecp256k1{}
	}
	pub, priv, err := signer.RandomKeyPair()
	if !checkError(err, c, http.StatusInternalServerError, "Generate account error.") {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"pubkey":  pub.String(),
		"privkey": priv.String(),
	})
}

func (r *RpcController) AutoTx(c *gin.Context) {
	intervalStr := c.Query("interval_ms")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "interval format err",
		})
		return
	}
	r.AutoTxCli.SetTxIntervalMs(interval)
	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
	return
}

func (r *RpcController) QueryNonce(c *gin.Context) {
	address := c.Query("address")
	addr, err := types.StringToAddress(address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "address format err",
		})
		return
	}
	noncePool, errPool := r.Og.TxPool.GetLatestNonce(addr)
	nonceDag, errDag := r.Og.Dag.GetLatestNonce(addr)
	var nonce int64
	if errPool != nil && errDag != nil {
		nonce = -1
	} else {
		nonce = int64(noncePool)
		if noncePool < nonceDag {
			nonce = int64(nonceDag)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"nonce": nonce,
	})
}
func (r *RpcController) QueryBalance(c *gin.Context) {
	address := c.Query("address")
	addr, err := types.StringToAddress(address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "address format err",
		})
		return
	}
	b := r.Og.Dag.GetBalance(addr)
	c.JSON(http.StatusBadRequest, gin.H{
		"balance": b,
	})
	return
}

func (r *RpcController) QueryShare(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

func (r *RpcController) ConstructPayload(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

func (r *RpcController) QueryReceipt(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

func (r *RpcController) QueryContract(c *gin.Context) {
	addrstr := c.Query("contract_address")
	querystr := c.Query("query_data")

	addr := types.HexToAddress(addrstr)
	query, err := hex.DecodeString(querystr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "can't decode query_data to bytes",
		})
	}
	ret, errc := r.Og.Dag.CallContract(addr, query)
	if errc != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Errorf("query contract error: %v", errc),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": hex.EncodeToString(ret),
	})
}

func (r *RpcController) OgPeersInfo(c *gin.Context) {
	info := r.Og.Manager.Hub.PeersInfo()
	c.JSON(http.StatusOK, info)
}

type Monitor struct {
	Port    string `json:"port"`
	ShortId string `json:"short_id"`
	Peers   []Peer `json:"peers,omitempty"`
	SeqId   uint64 `json:"seq_id"`
	Tps     *Tps   `json:"tps"`
}

type Peer struct {
	Addr    string `json:"addr"`
	ShortId string `json:"short_id"`
}

func (r *RpcController) Monitor(c *gin.Context) {
	var m Monitor
	seq := r.Og.Dag.LatestSequencer()
	if seq != nil {
		m.SeqId = seq.Id
	}
	peersinfo := r.P2pServer.PeersInfo()
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
		peer.Addr = p.Network.RemoteAddress
		peer.ShortId = p.ShortId
		m.Peers = append(m.Peers, peer)
	}
	m.Port = viper.GetString("p2p.port")
	m.ShortId = r.P2pServer.NodeInfo().ShortId
	m.Tps, _ = r.getTps()
	c.JSON(http.StatusOK, m)
}

func (r *RpcController) Debug(c *gin.Context) {
	p := c.Request.URL.Query().Get("f")
	switch p {
	case "1":
		r.NewRequestChan <- types.TxBaseTypeNormal
		// <-ffchan.NewTimeoutSender(r.NewRequestChan, types.TxBaseTypeNormal, "manualRequest", 1000).C
	case "2":
		r.NewRequestChan <- types.TxBaseTypeSequencer
		// <-ffchan.NewTimeoutSender(r.NewRequestChan, types.TxBaseTypeSequencer, "manualRequest", 1000).C
	case "cc":
		err := r.DebugCreateContract()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("new contract failed, err: %v", err),
			})
		} else {
			c.JSON(http.StatusOK, "success")
		}
		return 
	case "qc":
		ret, err := r.DebugQueryContract()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("new contract failed, err: %v", err),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"ret": fmt.Sprintf("%x", ret),
			})
		}
		return
	}
}

func (r *RpcController) DebugCreateContract() error {
	from := types.HexToAddress("0x60ce04e6a1cc8887fa5dcd43f87c38be1d41827e")
	to := types.HexToAddress("0x594a35ba66101523fcd229cb51368d176c405a2f")
	value := math.NewBigInt(0)
	nonce := uint64(1)

	contractCode := "6060604052341561000f57600080fd5b600a60008190555060006001819055506102078061002e6000396000f300606060405260043610610062576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680631c0f72e11461006b57806360fe47b114610094578063c605f76c146100b7578063e5aa3d5814610145575b34600181905550005b341561007657600080fd5b61007e61016e565b6040518082815260200191505060405180910390f35b341561009f57600080fd5b6100b56004808035906020019091905050610174565b005b34156100c257600080fd5b6100ca61017e565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010a5780820151818401526020810190506100ef565b50505050905090810190601f1680156101375780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015057600080fd5b6101586101c1565b6040518082815260200191505060405180910390f35b60015481565b8060008190555050565b6101866101c7565b6040805190810160405280600a81526020017f68656c6c6f576f726c6400000000000000000000000000000000000000000000815250905090565b60005481565b6020604051908101604052806000815250905600a165627a7a723058208e1bdbeee227900e60082cfcc0e44d400385e8811ae77ac6d7f3b72f630f04170029"
	data, _ := hex.DecodeString(contractCode)

	pubstr := "0x0104a391a55b84e45858748324534a187c60046266b17a5c77161104c4a6c4b1511789de692b898768ff6ee731e2fe068d6234a19a1e20246c968df9c0ca797498e7"
	pub, err := crypto.PublicKeyFromString(pubstr)
	if err != nil {
		return err
	}
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

func (r *RpcController) DebugQueryContract() ([]byte, error) {
	from := types.HexToAddress("0x60ce04e6a1cc8887fa5dcd43f87c38be1d41827e")
	contractAddr := crypto.CreateAddress(from, uint64(1))

	calldata := "e5aa3d58"
	callTx := &types.Tx{}
	callTx.From = from
	callTx.Value = math.NewBigInt(0)
	callTx.To = contractAddr
	callTx.Data, _ = hex.DecodeString(calldata)

	return r.Og.Dag.ProcessTransaction(callTx)
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

func isContractCreateTx(tx *types.Tx) bool {
	return len(tx.Data) == 0
}
