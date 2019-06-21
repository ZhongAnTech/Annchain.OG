package core

import (
	"fmt"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/types"
	"testing"
)

func TestDag_Stop(t *testing.T) {
	dag := Dag{
		statedb: &state.StateDB{},
	}
	oldHash := dag.statedb.Root()
	newHash, _ := types.HexStringToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	fmt.Println(oldHash != newHash)
	fmt.Println(newHash.Empty(), oldHash.Empty())
	dag.SaveStateRoot()
}
