package rpc

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type RpcControler struct {
	P2pServer     *p2p.Server
	Og            *og.Og
	TxBuffer      *og.TxBuffer
	AutoSequencer SequenceRequester
	AutoTx        TxRequester
}

type NodeStatus struct {
	NodeInfo  *p2p.NodeInfo   `json:"node_info"`
	PeersInfo []*p2p.PeerInfo `json:"peers_info"`
}

type TxRequester interface {
	GenerateRequest(from int, to int)
}

type SequenceRequester interface {
	GenerateRequest()
}

func (r *RpcControler) Status(c *gin.Context) {
	var status NodeStatus
	status.NodeInfo = r.P2pServer.NodeInfo()
	status.PeersInfo = r.P2pServer.PeersInfo()
	c.JSON(http.StatusOK, status)
}

func (r *RpcControler) NetInfo(c *gin.Context) {
	info := r.P2pServer.NodeInfo()
	c.JSON(http.StatusOK, info)
}

func (r *RpcControler) PeersInfo(c *gin.Context) {
	peersInfo := r.P2pServer.PeersInfo()
	c.JSON(http.StatusOK, peersInfo)
}
func (r *RpcControler) Query(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}
func (r *RpcControler) Transaction(c *gin.Context) {
	hashtr := c.Query("hash")
	hash, err := types.HexStringToHash(hashtr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "hash format error",
		})
		return
	}
	txi := r.Og.Dag.GetTx(hash)
	if txi == nil {
		txi = r.Og.Txpool.Get(hash)
	}
	if txi == nil {
		c.JSON(http.StatusOK, gin.H{
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
	c.JSON(http.StatusOK, gin.H{
		"message": "not found",
	})

}

func (r *RpcControler) Transactions(c *gin.Context) {
	seqId := c.Query("seq_id")
	id ,err:= strconv.Atoi(seqId)
	if err!=nil || id <0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "seq_id format error",
		})
		return
	}
	txs:=  r.Og.Dag.GetTxsByNumber(uint64(id))
	var txsREsponse struct {
		Total   int        `json:"total"`
		Txs []*types.Tx `json:"txs"`
	}
	if len(txs)!=0 {
		txsREsponse.Total = len(txs)
		txsREsponse.Txs = txs
		c.JSON(http.StatusOK, txsREsponse)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "not found",
	})

}

func (r *RpcControler) Genesis(c *gin.Context) {
	sq := r.Og.Dag.Genesis()
	if sq != nil {
		c.JSON(http.StatusOK, sq)
	} else {
		c.JSON(http.StatusOK, gin.H{
			"error": "not found",
		})
	}
	return
}

func (r *RpcControler) Sequencer(c *gin.Context) {
	var sq *types.Sequencer
	hashtr := c.Query("hash")
	if hashtr == "" {
		sq = r.Og.Dag.LatestSequencer()
		if sq != nil {
			c.JSON(http.StatusOK, sq)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"error": "not found",
			})
		}
		return
	} else {
		hash, err := types.HexStringToHash(hashtr)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "hash format error",
			})
			return
		}
		txi := r.Og.Dag.GetTx(hash)
		if txi == nil {
			txi = r.Og.Txpool.Get(hash)
		}
		if txi == nil {
			c.JSON(http.StatusOK, gin.H{
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
	c.JSON(http.StatusOK, gin.H{
		"error": "not found",
	})
}
func (r *RpcControler) Validator(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "validator",
	})
}
func (r *RpcControler) NewTransaction(c *gin.Context) {
	var tx types.Tx
	err := c.ShouldBindJSON(&tx)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}
	r.TxBuffer.AddTx(&tx)
	//todo add transaction
	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}
func (r *RpcControler) QueryNonce(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}
func (r *RpcControler) QueryBalance(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

func (r *RpcControler) QueryShare(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

func (r *RpcControler) ConstructPayload(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

func (r *RpcControler) QueryReceipt(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}
func (r *RpcControler) QueryContract(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}

func (r *RpcControler) OgPeersInfo(c *gin.Context) {
	info := r.Og.Manager.Hub.PeersInfo()
	c.JSON(http.StatusOK, info)
}

func (r *RpcControler) Debug(c *gin.Context) {
	p := c.Request.URL.Query().Get("f")
	switch p {
	case "1":
		r.AutoTx.GenerateRequest(0, 1)
	case "2":
		r.AutoSequencer.GenerateRequest()

	}
}
