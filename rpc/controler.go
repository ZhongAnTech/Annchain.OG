package rpc

import (
	"fmt"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"

	"github.com/annchain/OG/og"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/crypto"
	"github.com/gin-gonic/gin"
)

type RpcController struct {
	P2pServer      *p2p.Server
	Og             *og.Og
	TxBuffer       *og.TxBuffer
	TxCreator      *og.TxCreator
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

//NewTxrequest for RPC request
type NewTxRequest struct {
	Nonce     string `json:"nonce"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
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
		txi = r.Og.Txpool.Get(hash)
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
			txi = r.Og.Txpool.Get(hash)
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
		err = fmt.Errorf("New Big Int error")
	}
	if !checkError(err, c, http.StatusBadRequest, "value format error") {
		return
	}

	nonce, err := strconv.ParseUint(txReq.Nonce, 10, 64)
	if !checkError(err, c, http.StatusBadRequest, "nonce format error") {
		return
	}

	signature, err := hexutil.Decode(txReq.Signature)
	if !checkError(err, c, http.StatusBadRequest, "signature format error") {
		return
	}

	pub ,err = crypto.PublicKeyFromString(txReq.Pubkey)
	if !checkError(err,c,http.StatusBadRequest,"pubkey format error"){
		return
	}

	sig = crypto.SignatureFromBytes(pub.Type,signature)

	tx, err = r.TxCreator.NewTxWithSeal(from, to, value, nonce, pub, sig)
	if !checkError(err, c, http.StatusInternalServerError, "new tx failed") {
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	if !r.Og.Manager.AcceptTxs() {
		c.JSON(http.StatusOK, gin.H{
			"error": "tx is disabled when syncing",
		})
		return
	}
	r.TxBuffer.AddLocal(tx)
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
		"pubkey":  pub.PublicKeyToString(),
		"privkey": priv.PrivateKeyToString(),
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
	noncePool, errPool := r.Og.Txpool.GetLatestNonce(addr)
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
		r.NewRequestChan <- types.TxBaseTypeNormal
	case "2":
		r.NewRequestChan <- types.TxBaseTypeSequencer
	}
}

func checkError(err error, c *gin.Context, status int, message string) bool {
	if err != nil {
		c.JSON(status, gin.H{
			"error": fmt.Sprintf("%s:%s",err.Error() , message),
		})
		return false
	}
	return true
}
