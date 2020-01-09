package core

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/core/state"
	"testing"
)

func TestDag_Stop(t *testing.T) {
	dag := Dag{
		statedb: &state.StateDB{},
	}
	oldHash := dag.statedb.Root()
	newHash, _ := common.HexStringToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	fmt.Println(oldHash != newHash)
	fmt.Println(newHash.Empty(), oldHash.Empty())
	dag.SaveStateRoot()
}
