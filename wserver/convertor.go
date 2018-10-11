package wserver

import (
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
	Type  string `json:"type"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func (u *UIData) AddToBatch(tx types.Txi, includingEdge bool) {
	nodeData := NodeData{
		Unit:   tx.GetTxHash().Hex(),
		Unit_s: tx.String(),
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

	u.Nodes = append(u.Nodes, node)

	if includingEdge {
		for _, parent := range tx.Parents() {
			edge := Edge{
				Id:     tx.String() + "_" + parent.String(),
				Source: tx.GetTxHash().Hex(),
				Target: parent.Hex(),
			}
			u.Edges = append(u.Edges, edge)
		}
	}

}
