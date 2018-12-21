package state_test

import (
	"testing"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
)

func newTestStateDB(t *testing.T) *state.StateDB {
	db := ogdb.NewMemDatabase()
	stdb, err := state.NewStateDB(state.DefaultStateDBConfig(), state.NewDatabase(db))
	if err != nil {
		t.Errorf("create StateDB error: %v", err)
	}
	return stdb
}

func TestStateDB(t *testing.T) {

}

var (
	storageKey1   = crypto.Keccak256Hash([]byte("key1"))
	storageValue1 = crypto.Keccak256Hash([]byte("value1"))
	storageKey2   = crypto.Keccak256Hash([]byte("key2"))
	storageValue2 = crypto.Keccak256Hash([]byte("value2"))
	storageKey3   = crypto.Keccak256Hash([]byte("key3"))
	storageValue3 = crypto.Keccak256Hash([]byte("value3"))
)

func TestStateStorage(t *testing.T) {
	t.Parallel()

	stdb := newTestStateDB(t)
	addr := types.HexToAddress(testAddress)

	stdb.SetState(addr, storageKey1, storageValue1)
	stdb.SetState(addr, storageKey2, storageValue2)
	stdb.SetState(addr, storageKey3, storageValue3)

	_, err := stdb.Commit()
	if err != nil {
		t.Fatalf("commit storage error: %v", err)
	}
	// general test
	st1 := stdb.GetState(addr, storageKey1)
	if st1.Hex() != storageValue1.Hex() {
		t.Fatalf("value1 is not committed, should be %s, get %s", st1.Hex(), storageValue1.Hex())
	}

	// check if is successfully committed into trie db.
	stobj, geterr := stdb.GetStateObject(addr)
	if geterr != nil {
		t.Fatalf("get stateobject error: %v", geterr)
	}
	stobj.Uncache()
	st2 := stdb.GetState(addr, storageKey2)
	if st2.Hex() != storageValue2.Hex() {
		t.Fatalf("value2 is not committed, should be %s, get %s", st2.Hex(), storageValue2.Hex())
	}

}
