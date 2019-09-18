package rpc

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/status"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"math/big"
	"net/http"
)

type TxsResp struct {
	Sequencer    *SeqResp `json:"sequencer"`
	Transactions []TxResp `json:"transactions"`
}

type TxResp struct {
	Type      types.TxBaseType `json:"type"`
	Hash      string           `json:"hash"`
	Parents   []string         `json:"parents"`
	From      string           `json:"from"`
	To        string           `json:"to"`
	Nonce     uint64           `json:"nonce"`
	Guarantee string           `json:"guarantee"`
	Value     string           `json:"value"`
	Weight    uint64           `json:"weight"`
}

type SeqResp struct {
	Type     types.TxBaseType `json:"type"`
	Hash     string           `json:"hash"`
	Parents  []string         `json:"parents"`
	From     string           `json:"from"`
	Nonce    uint64           `json:"nonce"`
	Treasure string           `json:"treasure"`
	Height   uint64           `json:"height"`
	Weight   uint64           `json:"weight"`
}

func (r *RpcController) QueryAllTips(c *gin.Context) {
	respSet := txsHelper(r.Og.TxPool.GetAllTips())
	Response(c, http.StatusOK, nil, respSet)
}

func (r *RpcController) QueryAllTxsInPool(c *gin.Context) {
	respSet := txsHelper(r.Og.TxPool.GetAllTxs())
	Response(c, http.StatusOK, nil, respSet)
}

func txHelper(txi types.Txi) interface{} {
	switch tx := txi.(type) {
	case *tx_types.Tx:
		txResp := &TxResp{}
		txResp.Type = tx.Type
		txResp.Hash = tx.Hash.Hex()

		for _, p := range tx.ParentsHash {
			txResp.Parents = append(txResp.Parents, p.Hex())
		}

		txResp.From = tx.From.Hex()
		txResp.To = tx.To.Hex()
		txResp.Nonce = tx.AccountNonce
		txResp.Guarantee = tx.Guarantee.String()
		txResp.Value = tx.Value.String()
		txResp.Weight = tx.Weight

		return txResp

	case *tx_types.Sequencer:
		seqResp := &SeqResp{}
		seqResp.Type = tx.Type
		seqResp.Hash = tx.Hash.Hex()

		for _, p := range tx.ParentsHash {
			seqResp.Parents = append(seqResp.Parents, p.Hex())
		}

		seqResp.From = tx.Sender().Hex()
		seqResp.Nonce = tx.AccountNonce
		seqResp.Treasure = tx.Treasure.String()
		seqResp.Height = tx.Height
		seqResp.Weight = tx.Weight

		return seqResp

	default:
		return nil
	}
}

func txsHelper(txiList []types.Txi) TxsResp {
	var seq *SeqResp
	var txs = make([]TxResp, 0)
	for _, txi := range txiList {
		switch tx := txi.(type) {
		case *tx_types.Tx:
			resp := TxResp{}
			resp.Type = tx.Type
			resp.Hash = tx.Hash.Hex()

			for _, p := range tx.ParentsHash {
				resp.Parents = append(resp.Parents, p.Hex())
			}

			resp.From = tx.From.Hex()
			resp.To = tx.To.Hex()
			resp.Nonce = tx.AccountNonce
			resp.Guarantee = tx.Guarantee.String()
			resp.Value = tx.Value.String()

			txs = append(txs, resp)

		case *tx_types.Sequencer:
			resp := SeqResp{}
			resp.Type = tx.Type
			resp.Hash = tx.Hash.Hex()

			for _, p := range tx.ParentsHash {
				resp.Parents = append(resp.Parents, p.Hex())
			}

			resp.From = tx.Sender().Hex()
			resp.Nonce = tx.AccountNonce
			resp.Treasure = tx.Treasure.String()
			resp.Height = tx.Height

			seq = &resp

		default:
			continue
		}
	}

	resp := TxsResp{}
	resp.Sequencer = seq
	resp.Transactions = txs
	return resp
}

func (r *RpcController) SecretTransaction(c *gin.Context) {
	var (
		tx    types.Txi
		txReq NewSecretTxRequest
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

	var parentsHash []common.Hash
	for _, hashStr := range txReq.Parents {
		hash := common.HexToHash(hashStr)
		if hash.Empty() {
			Response(c, http.StatusBadRequest, fmt.Errorf("invalid parent hash: %b", hash.Bytes), nil)
			return
		}
		parentsHash = append(parentsHash, hash)
	}

	from, err := common.StringToAddress(txReq.From)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("from address format error: %v", err), nil)
		return
	}
	to, err := common.StringToAddress(txReq.To)
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

	guarantee, ok := math.NewBigIntFromString(txReq.Guarantee, 10)
	if !ok {
		err = fmt.Errorf("convert guarantee to big int error")
	}
	if err != nil {
		Response(c, http.StatusBadRequest, err, nil)
		return
	}
	if guarantee.Value.Cmp(big.NewInt(100)) < 0 {
		Response(c, http.StatusBadRequest, fmt.Errorf("guarantee should be larger than 100"), nil)
		return
	}

	nonce := txReq.Nonce

	signature := common.FromHex(txReq.Signature)
	if signature == nil || txReq.Signature == "" {
		Response(c, http.StatusBadRequest, fmt.Errorf("signature format error"), nil)
		return
	}

	pub, err = crypto.Secp256k1PublicKeyFromString(txReq.Pubkey)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("pubkey format error %v", err), nil)
		return
	}

	sig = crypto.SignatureFromBytes(pub.Type, signature)
	if sig.Type != crypto.Signer.GetCryptoType() || pub.Type != crypto.Signer.GetCryptoType() {
		Response(c, http.StatusOK, fmt.Errorf("crypto algorithm mismatch"), nil)
		return
	}

	tx, err = r.TxCreator.NewTxForHackathon(from, to, value, guarantee, []byte{}, nonce, parentsHash, pub, sig, BaseTokenID)
	if err != nil {
		Response(c, http.StatusInternalServerError, fmt.Errorf("new tx failed: %v", err), nil)
		return
	}

	ok = r.FormatVerifier.VerifySignature(tx)
	if !ok {
		logrus.WithField("request ", txReq).WithField("tx ", tx).Warn("signature invalid")
		Response(c, http.StatusInternalServerError, fmt.Errorf("signature invalid"), nil)
		return
	}
	ok = r.FormatVerifier.VerifySourceAddress(tx)
	if !ok {
		logrus.WithField("request ", txReq).WithField("tx ", tx).Warn("source address invalid")
		Response(c, http.StatusInternalServerError, fmt.Errorf("source address invalid"), nil)
		return
	}
	tx.SetVerified(types.VerifiedFormat)
	logrus.WithField("tx", tx).Debugf("tx generated")
	if !r.SyncerManager.IncrementalSyncer.Enabled {
		Response(c, http.StatusOK, fmt.Errorf("tx is disabled when syncing"), nil)
		return
	}

	r.TxBuffer.ReceivedNewTxChan <- tx

	Response(c, http.StatusOK, nil, tx.GetTxHash().Hex())
	return
}

//NewTxrequest for RPC request
//msgp:tuple NewTxRequest
type NewSecretTxRequest struct {
	Parents   []string `json:"parents"`
	Nonce     uint64   `json:"nonce"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Value     string   `json:"value"`
	Guarantee string   `json:"guarantee"`
	//Data      string `json:"data"`
	//CryptoType string `json:"crypto_type"`
	Signature string `json:"signature"`
	Pubkey    string `json:"pubkey"`
	//TokenId   int32  `json:"token_id"`
}
