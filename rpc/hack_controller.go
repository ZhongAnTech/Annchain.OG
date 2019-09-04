package rpc

import (
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/gin-gonic/gin"
	"net/http"
)

type TxsResp struct {
	Sequencer    SeqResp  `json:"sequencer"`
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
}

type SeqResp struct {
	Type     types.TxBaseType `json:"type"`
	Hash     string           `json:"hash"`
	Parents  []string         `json:"parents"`
	From     string           `json:"from"`
	Nonce    uint64           `json:"nonce"`
	Treasure string           `json:"treasure"`
	Height   uint64           `json:"height"`
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
		txResp := TxResp{}
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

		return txResp

	case *tx_types.Sequencer:
		seqResp := SeqResp{}
		seqResp.Type = tx.Type
		seqResp.Hash = tx.Hash.Hex()

		for _, p := range tx.ParentsHash {
			seqResp.Parents = append(seqResp.Parents, p.Hex())
		}

		seqResp.From = tx.Sender().Hex()
		seqResp.Nonce = tx.AccountNonce
		seqResp.Treasure = tx.Treasure.String()
		seqResp.Height = tx.Height

		return seqResp

	default:
		return nil
	}
}

func txsHelper(txiList []types.Txi) TxsResp {
	var seq SeqResp
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

			seq = resp

		default:
			continue
		}
	}

	resp := TxsResp{}
	resp.Sequencer = seq
	resp.Transactions = txs
	return resp
}
