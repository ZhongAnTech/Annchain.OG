package core

import (
	"fmt"
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
	dirtyset	map[types.Address]struct{}
	beats		map[types.Address]time.Time

	mu 			sync.RWMutex
}
func NewStateDB(db ogdb.Database, acc *Accessor) *StateDB {
	sd := &StateDB{
		db:			db,
		accessor:	acc,
		states:		make(map[types.Address]*State),
		dirtyset:	make(map[types.Address]struct{}),
		beats:		make(map[types.Address]time.Time),
	}
	return sd
}

// CreateNewState will create a new state for input address and 
// return the state. If input address already exists in StateDB 
// it returns an error.
func (sd *StateDB) CreateNewState(addr types.Address) (*State, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	state := sd.states[addr]
	if state != nil {
		return nil, fmt.Errorf("state already exists, addr: %s", addr.String())
	}
	
	state = NewState(addr)
	sd.states[addr] = state
	sd.beats[addr] = time.Now()

	return state, nil
}

func (sd *StateDB) GetBalance(addr types.Address) (*math.BigInt, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getBalance(addr)
}
func (sd *StateDB) getBalance(addr types.Address) (*math.BigInt, error) {
	state := sd.GetState(addr)
	if state == nil {
		return math.NewBigInt(0), nil
	}
	return state.Balance, nil
}

func (sd *StateDB) GetNonce(addr types.Address) (uint64, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getNonce(addr)
}
func (sd *StateDB) getNonce(addr types.Address) (uint64, error) {
	state := sd.GetState(addr)
	if state == nil {
		return 0, types.ErrNonceNotExist
	}
	return state.Nonce, nil
}

// GetState get a state from statedb. If state not exist, 
// load it from db. 
func (sd *StateDB) GetState(addr types.Address) *State {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getState(addr)
}
func (sd *StateDB) getState(addr types.Address) *State {
	state := sd.states[addr]
	if state == nil {
		state = sd.loadState(addr)
		sd.updateState(addr, state)
	}
	return state
}

// UpdateState set addr's state in StateDB. 
// 
// Note that this setting will force updating the StateDB without 
// any verification. Call this UpdateState carefully.
func (sd *StateDB) UpdateState(addr types.Address, state *State) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.updateState(addr, state)
}
func (sd *StateDB) updateState(addr types.Address, state *State) {
	sd.states[addr] = state
	sd.dirtyset[addr] = struct{}{}
	sd.refreshbeat(addr)
}

func (sd *StateDB) AddBalance(addr types.Address, increment *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	state := sd.GetState(addr)
	if state == nil {
		state = NewState(addr)
	}
	state.AddBalance(increment)
	
	sd.updateState(addr, state)
}

func (sd *StateDB) SubBalance(addr types.Address, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	state := sd.GetState(addr)
	if state == nil {
		state = NewState(addr)
	}
	state.SubBalance(decrement)
	
	sd.updateState(addr, state)
}

func (sd *StateDB) SetBalance(addr types.Address, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	state := sd.GetState(addr)
	if state == nil {
		state = NewState(addr)
	}
	state.SetBalance(balance)
	
	sd.updateState(addr, state)
}

func (sd *StateDB) SetNonce(addr types.Address, nonce uint64) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	state := sd.GetState(addr)
	if state == nil {
		state = NewState(addr)
	}
	state.SetNonce(nonce)
	
	sd.updateState(addr, state)
}

// Load get data from db.
func (sd *StateDB) loadState(addr types.Address) *State {
	return sd.accessor.LoadState(addr)
}

// Confirm
func (sd *StateDB) Confirm() {
	// TODO
}

func (sd *StateDB) refreshbeat(addr types.Address) {
	state := sd.states[addr]
	if state == nil {
		return
	}
	sd.beats[addr] = time.Now()
}

type State struct {
	Address 	types.Address
	Balance		*math.BigInt
	Nonce		uint64
}
func NewState(addr types.Address) *State {
	s := &State{}
	s.Address = addr
	s.Balance = math.NewBigInt(0)
	s.Nonce = 0
	return s
}

func (s *State) AddBalance(increment *math.BigInt) {
	// check if increment is zero
	if increment.Sign() == 0 {
		return
	}
	s.SetBalance(s.Balance.Add(increment))
}

func (s *State) SubBalance(decrement *math.BigInt) {
	// check if decrement is zero
	if decrement.Sign() == 0 {
		return
	}
	s.SetBalance(s.Balance.Sub(decrement))
}

func (s *State) SetBalance(balance *math.BigInt) {
	s.Balance = balance
}

func (s *State) SetNonce(nonce uint64) {
	s.Nonce = nonce
}

