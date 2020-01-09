package state_test

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types/token"
	"testing"
)

func TestPreloadDBWorkFlows(t *testing.T) {

	db := ogdb.NewMemDatabase()

	initRoot := common.Hash{}
	stdb, err := state.NewStateDB(state.DefaultStateDBConfig(), state.NewDatabase(db), initRoot)
	if err != nil {
		t.Errorf("create StateDB error: %v", err)
	}

	addr := common.HexToAddress(testAddress)
	stdb.CreateAccount(addr)

	testnonce := uint64(123456)
	testblc := int64(666)

	stobj := stdb.GetStateObject(addr)
	stobj.SetNonce(testnonce)
	stobj.SetBalance(token.OGTokenID, math.NewBigInt(testblc))

	stdb.SetState(addr, storageKey1, storageValue1)
	stdb.SetState(addr, storageKey2, storageValue2)
	stdb.SetState(addr, storageKey3, storageValue3)

	rootHash, err := stdb.Commit()
	if err != nil {
		t.Errorf("statedb commit error")
	}
	fmt.Println("root hash: ", rootHash)

	pdb := state.NewPreloadDB(stdb.Database(), stdb)
	pdb.SetState(addr, storageKey1, storageValue3)

	// commit state db.
	stRoot, err := stdb.Commit()
	if err != nil {
		t.Errorf("statedb commit error: %v", err)
	}
	if stRoot.Cmp(rootHash) != 0 {
		t.Errorf("statedb root been changed")
	}

	// commit preload db.
	pRoot, err := pdb.Commit()
	if err != nil {
		t.Errorf("pdb commit error: %v", err)
	}
	if pRoot.Cmp(rootHash) == 0 {
		t.Errorf("preload db root should change after state setting")
	}

	// check the value in state db and preload db.
	sdbValue := stdb.GetState(addr, storageKey1)
	if sdbValue.Cmp(storageValue1) != 0 {
		t.Errorf("value in statedb should not be changed. should be: %v, get: %v", storageValue1, sdbValue)
	}
	pdbValue := pdb.GetState(addr, storageKey1)
	if pdbValue.Cmp(storageValue3) != 0 {
		t.Errorf("value in statedb should be changed. should be: %v, get: %v", storageValue3, pdbValue)
	}

	stdb.SetState(addr, storageKey1, storageValue3)
	stRoot, err = stdb.Commit()
	if err != nil {
		t.Errorf("statedb commit error: %v", err)
	}
	if stRoot.Cmp(pRoot) != 0 {
		t.Errorf("statedb root not equal to preload db root. stRoot: %v, pRoot: %v", stRoot, pRoot)
	}

}
