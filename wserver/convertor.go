package wserver

import (
	"encoding/json"
	"github.com/annchain/OG/types"
)

//type Tx struct {
//	TxBase
//	From  Address
//	To    Address
//	Value *math.BigInt
//}
//type TxBase struct {
//	Type         TxBaseType
//	Hash         Hash
//	ParentsHash  []Hash
//	AccountNonce uint64
//	Height       uint64
//	PublicKey    []byte
//	Signature    []byte
//}

type NodeData struct {
	Unit   string `json:"unit"`
	Unit_s string `json:"unit_s"`
}

type Node struct {
	Data NodeData `json:"data"`
	Type string   `json:"type"`
}

type Edge struct {
	Id     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type UIData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func tx2UIData(tx types.Txi) string {
	uiData := UIData{}
	uiData.Nodes = make([]Node, 1)
	nodeData := NodeData{
		Unit:   tx.GetTxHash().Hex(),
		Unit_s: tx.GetTxHash().Hex(), //[:6] + "...",
	}
	node := Node{
		Data: nodeData,
	}

	switch tx.GetBase().Type {
	case types.TxBaseTypeSequencer:
		node.Type = "sequencer_unit"
	case types.TxBaseTypeNormal:
		node.Type = ""
	default:
		node.Type = "Unknown"
	}

	uiData.Nodes[0] = node
	for _, parent := range tx.Parents() {
		edge := Edge{
			Id:     tx.GetTxHash().Hex() + "_" + parent.Hex(),
			Source: tx.GetTxHash().Hex(),
			Target: parent.Hex(),
		}
		uiData.Edges = append(uiData.Edges, edge)
	}

	bs, err := json.Marshal(&uiData)
	if err != nil {
		return ""
	}
	return string(bs)
}
