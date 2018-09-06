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
	Hash   string `json:"hash"`
	Hash_s string `json:"hash_s"`
}

type Node struct {
	Data   NodeData `json:"data"`
	Type   int      `json:"Type"`
	Height uint64   `json:"Height"`
}

type EdgeData struct{
	LinkType int `json:"link_type"`
	Data struct{
		Id string `json:"id"`
		Source string `json:"source"`
		Target string `json:"target"`
	} `json:"data"`
}

type Edge struct{
	Data EdgeData `json:"data"`
}

type UIData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func tx2UIData(tx types.Tx) string {
	uiData := UIData{}
	uiData.Nodes = make([]Node, 1)
	nodeData := NodeData{
		Hash:   tx.Hash.Hex(),
		Hash_s: tx.Hash.Hex()[:6] + "...",
	}
	node := Node{
		Data:   nodeData,
		Type:   int(tx.Type),
		Height: tx.Height,
	}
	uiData.Nodes[0] = node;

	for _,parent := range tx.ParentsHash{
		edgeData := EdgeData{
			LinkType: 0,//TODO
			Data: struct{
				Id string `json:"id"`
				Source string `json:"source"`
				Target string `json:"target"`
			}{
				Id: "",//TODO
				Source: tx.Hash.Hex(),
				Target: parent.Hex(),
			},
		}
		edge := Edge{
			Data:edgeData,
		}
		uiData.Edges = append(uiData.Edges,edge)
	}

	bs ,err := json.Marshal(&uiData)
	if err != nil{
		return ""
	}
	return string(bs)
}
