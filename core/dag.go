package core

import (
	"github.com/annchain/OG/ogdb"
	// "github.com/annchain/OG/og/types"
)

type Dag struct {
	db ogdb.Database

	// genesisBlock *types.Block
	// latestBlock  *types.Block

	// chain	[]*types.Block

}

func (dag *Dag) Start() {

	go dag.loop()
}

// func (dag *Dag) GetBlock(i int) *types.Block {
// 	if i == -1 {
// 		return dag.chain[len(dag.chain)-1]
// 	}
// 	if i < 0 || i > len(dag.chain) {
// 		return nil
// 	}
// 	return dag.chain[i]
// }

// func (dag *Dag) TryAppendBlock(block *types.Block) {
// 	if block.Index == dag.latestBlock.Index + 1 {
// 		block.PrevBlock = dag.latestBlock
// 		block.NextBlock = nil
// 		dag.latestBlock.NextBlock = block
// 		dag.latestBlock = block
// 		dag.chain = append(dag.chain, block)
// 	}
// 	return
// }

func (dag *Dag) Bytes() []byte {
	return []byte(dag.String())
}
func (dag *Dag) String() string {
	result := ""
	// for _, block := range dag.chain{
	// 	result += block.String()
	// }
	return result
}

// func (dag *Dag) Len() int { return int(dag.latestBlock.Index) }

func (dag *Dag) loop() {
	for {
		select {
		// TODO
		default:
			break
		}
	}
}
