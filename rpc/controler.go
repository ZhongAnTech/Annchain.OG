package rpc

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/p2p"
	"github.com/gin-gonic/gin"
	"net/http"
)

type RpcControler struct {
	P2pServer *p2p.Server
	Og        *og.Og
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
	c.JSON(http.StatusOK, gin.H{
		"message": "hello",
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
