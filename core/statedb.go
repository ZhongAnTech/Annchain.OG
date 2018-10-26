package core

import (
	"sync"
	"time"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/common/math"
)

type StateDB struct {
	db			ogdb.Database
	accessor	*Accessor
	states		map[types.Address]*State

	mu 			sync.RWMutex
}


func (da *StateDB) GetBalance(addr types.Address) (*math.BigInt, error) {
	// TODO

	return nil, nil
}

func (da *StateDB) GetNonce(addr types.Address) (uint64, error) {
	// TODO
	
	return 0, nil
}

func (sd *StateDB) GetState(addr types.Address) *State {
	state := sd.states[addr]
	if state == nil {
		state = sd.LoadState(addr)
		sd.updateState(addr, state)
	}
	return state
}

// UpdateState
func (sd *StateDB) UpdateState(addr types.Address, state *State) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.updateState(addr, state)
}
func (sd *StateDB) updateState(addr types.Address, state *State) {
	// TODO
	// Update state's beat time.
}

// Load get data from db.
func (sd *StateDB) LoadState(addr types.Address) *State {
	return sd.accessor.LoadState(addr)
}

func (sd *StateDB) Confirm() {

}


type State struct {
	Address 	types.Address
	Balance		*math.BigInt
	LastBeat	time.Time
	Nonce		uint64
	IsDirty		bool
}







