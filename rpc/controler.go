package rpc

import (
	"net/http"
	"strconv"

	"github.com/annchain/OG/og"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	"github.com/gin-gonic/gin"
	"github.com/annchain/OG/og/syncer"
	"github.com/annchain/OG/ffchan"
)

type RpcController struct {
	P2pServer      *p2p.Server
	Og             *og.Og
	TxBuffer       *og.TxBuffer
	SyncerManager  *syncer.SyncManager
	NewRequestChan chan types.TxBaseType
}

//NodeStatus
type NodeStatus struct {
	NodeInfo  *p2p.NodeInfo   `json:"node_info"`
	PeersInfo []*p2p.PeerInfo `json:"peers_info"`
}

//TxRequester
type TxRequester interface {
	GenerateRequest(from int, to int)
}

//SequenceRequester
type SequenceRequester interface {
	GenerateRequest()
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
	var tx types.Tx
	err := c.ShouldBindJSON(&tx)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}
	r.TxBuffer.AddLocalTx(&tx)
	//todo add transaction
	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
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
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

func (r *RpcController) OgPeersInfo(c *gin.Context) {
	info := r.Og.Manager.Hub.PeersInfo()
	c.JSON(http.StatusOK, info)
}

func (r *RpcController) Debug(c *gin.Context) {
	p := c.Request.URL.Query().Get("f")
	switch p {
	case "1":
		<- ffchan.NewTimeoutSender(r.NewRequestChan, types.TxBaseTypeNormal, "manualRequest", 1000).C
		//r.NewRequestChan <- types.TxBaseTypeNormal
	case "2":
		<- ffchan.NewTimeoutSender(r.NewRequestChan, types.TxBaseTypeSequencer, "manualRequest", 1000).C
		//r.NewRequestChan <- types.TxBaseTypeSequencer
	}
}
