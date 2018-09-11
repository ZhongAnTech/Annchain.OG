package rpc

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/p2p"
	"github.com/gin-gonic/gin"
	"net/http"
	"github.com/annchain/OG/types"
)

type RpcControler struct {
	P2pServer *p2p.Server
	Og        *og.Og
	TxBuffer   *og.TxBuffer
}

type NodeStatus struct {
	NodeInfo  *p2p.NodeInfo   `json:"node_info"`
	PeersInfo []*p2p.PeerInfo `json:"peers_info"`
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
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}
func (r *RpcControler) Sequencer(c *gin.Context) {
	var sq *types.Sequencer
	hashtr :=  c.Param("hash")
	if hashtr == ""{
		sq = r.Og.Dag.LatestSequencer()
		if sq!=nil {
			c.JSON(http.StatusOK,sq)
		}else {
			c.JSON(http.StatusOK, gin.H{
				"error": "not found",
			})
		}
		return
	}
	//todo with hashs
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
	})
}
func (r *RpcControler) Validator(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "validator",
	})
}
func (r *RpcControler) NewTransaction(c *gin.Context) {
	var tx types.Tx
	 err :=  c.ShouldBindJSON(&tx)
	 if err!=nil {
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
