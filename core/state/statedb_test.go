package state_test

import (
	"testing"

	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/ogdb"
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
