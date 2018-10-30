package core

import (
	"fmt"
	"sync"
	"time"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/common/math"
	log "github.com/sirupsen/logrus"
)

type StateDBConfig struct {
	FlushTimer		time.Duration
	PurgeTimer		time.Duration
	BeatExpireTime	time.Duration
}
func DefaultStateDBConfig() StateDBConfig {
	return StateDBConfig{
		FlushTimer: time.Duration(5000),
		PurgeTimer: time.Duration(10000),
		BeatExpireTime: time.Duration(10 * time.Minute),
	}
}

type StateDB struct {
	conf		StateDBConfig

	db			ogdb.Database
	accessor	*Accessor

	states		map[types.Address]*State
	dirtyset	map[types.Address]struct{}
	beats		map[types.Address]time.Time

	close		chan struct{}

	mu 			sync.RWMutex
}
func NewStateDB(conf StateDBConfig, db ogdb.Database, acc *Accessor) *StateDB {
	sd := &StateDB{
		conf:		conf,
		db:			db,
		accessor:	acc,
		states:		make(map[types.Address]*State),
		dirtyset:	make(map[types.Address]struct{}),
		beats:		make(map[types.Address]time.Time),
		close:		make(chan struct{}),
	}

	go sd.loop()
	return sd
}

func (sd *StateDB) Stop() {
	close(sd.close)
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
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.getState(addr)
}
func (sd *StateDB) getState(addr types.Address) *State {
	state, exist := sd.states[addr]
	if !exist {
		state = sd.loadState(addr)
		sd.updateState(addr, state)
	}
	return state
}

// DeleteState remove a state from StateDB. Return error 
// if it fails.
func (sd *StateDB) DeleteState(addr types.Address) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.deleteState(addr)
}
func (sd *StateDB) deleteState(addr types.Address) error {
	_, exist := sd.states[addr]
	if !exist {
		return nil
	}
	delete(sd.states, addr)
	return nil
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

// Load data from db.
func (sd *StateDB) loadState(addr types.Address) *State {
	return sd.accessor.LoadState(addr)
}

// flush tries to save those dirty data to disk db.
func (sd *StateDB) flush() {
	for addr, _ := range sd.dirtyset {
		state, exist := sd.states[addr]
		if !exist {
			log.Warnf("can't find dirty state in StateDB, addr: %s", addr.String())
			continue
		}
		sd.accessor.SaveState(addr, state)
	}
	sd.dirtyset = make(map[types.Address]struct{})
}

// purge tries to remove all the state that haven't sent any beat 
// for a long time. 
// 
// Note that dirty states will not be removed.
func (sd *StateDB) purge() {
	for addr, lastbeat := range sd.beats {
		// skip dirty states
		if _, in := sd.dirtyset[addr]; in {
			continue
		}
		if time.Since(lastbeat) > sd.conf.BeatExpireTime {
			sd.deleteState(addr)
		}
	}
}

// refreshbeat update the beat time of an address.
func (sd *StateDB) refreshbeat(addr types.Address) {
	state := sd.states[addr]
	if state == nil {
		return
	}
	sd.beats[addr] = time.Now()
}

func (sd *StateDB) loop() {
	flushTimer := time.NewTicker(sd.conf.FlushTimer)
	purgeTimer := time.NewTicker(sd.conf.PurgeTimer)

	for {
		select {
		case <-sd.close:
			flushTimer.Stop()
			purgeTimer.Stop()
			return

		case <-flushTimer.C:
			sd.mu.Lock()
			sd.flush()
			sd.mu.Unlock()

		case <-purgeTimer.C:
			sd.mu.Lock()
			sd.purge()
			sd.mu.Unlock()

		}
	}
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

