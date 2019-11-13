// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package wserver

import (
	"github.com/annchain/OG/types"
)

type NodeData struct {
	Unit   string `json:"unit"`
	Unit_s string `json:"unit_s"`
	//Tx    types.TxiSmallCaseMarshal `json:"tx"`
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

type BlockDbUIData struct {
	Nodes []types.TxiSmallCaseMarshal `json:"nodes"`
	//Type  string `json:"type"`
	//Edges []Edge `json:"edges"`
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
	default:
		node.Type = ""
		//default:
		//	node.Type = "Unknown"
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
