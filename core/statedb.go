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

func (sd *StateDB) GetBalance(addr types.Address) *math.BigInt {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getBalance(addr)
}
func (sd *StateDB) getBalance(addr types.Address) *math.BigInt {
	state := sd.getOrCreateState(addr)
	return state.Balance
}

func (sd *StateDB) GetNonce(addr types.Address) (uint64, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getNonce(addr)
}
func (sd *StateDB) getNonce(addr types.Address) (uint64, error) {
	state, err := sd.getState(addr)
	if err != nil {
		return 0, types.ErrNonceNotExist
	}
	return state.Nonce, nil
}

// GetState get a state from statedb. If state not exist, 
// load it from db. 
func (sd *StateDB) GetState(addr types.Address) (*State, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.getState(addr)
}
func (sd *StateDB) getState(addr types.Address) (*State, error) {
	state, exist := sd.states[addr]
	if !exist {
		var err error
		state, err = sd.loadState(addr)
		if err != nil {
			return nil, err
		}
		sd.updateState(addr, state)
	}
	return state, nil
}

// GetOrCreateState
func (sd *StateDB) GetOrCreateState(addr types.Address) *State {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.getOrCreateState(addr)
}
func (sd *StateDB) getOrCreateState(addr types.Address) *State {
	state, _ := sd.getState(addr)
	if state == nil {
		state = NewState(addr)
	}
	sd.updateState(addr, state)
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
	
	state := sd.getOrCreateState(addr)
	state.AddBalance(increment)
	
	sd.updateState(addr, state)
}

func (sd *StateDB) SubBalance(addr types.Address, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	state := sd.getOrCreateState(addr)
	state.SubBalance(decrement)
	
	sd.updateState(addr, state)
}

func (sd *StateDB) SetBalance(addr types.Address, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	state := sd.getOrCreateState(addr)
	state.SetBalance(balance)
	
	sd.updateState(addr, state)
}

func (sd *StateDB) SetNonce(addr types.Address, nonce uint64) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	state := sd.getOrCreateState(addr)
	state.SetNonce(nonce)
	
	sd.updateState(addr, state)
}

// Load data from db.
func (sd *StateDB) loadState(addr types.Address) (*State, error) {
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
