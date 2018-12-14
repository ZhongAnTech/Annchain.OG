package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
)

type StateDBConfig struct {
	PurgeTimer     time.Duration
	BeatExpireTime time.Duration
}

func DefaultStateDBConfig() StateDBConfig {
	return StateDBConfig{
		PurgeTimer:     time.Duration(10000),
		BeatExpireTime: time.Duration(10 * time.Minute),
	}
}

type StateDB struct {
	conf StateDBConfig

	db   Database
	trie Trie

	states   map[types.Address]*StateObject
	dirtyset map[types.Address]struct{}
	beats    map[types.Address]time.Time

	close chan struct{}

	mu sync.RWMutex
}

func NewStateDB(conf StateDBConfig, root types.Hash, db Database) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	sd := &StateDB{
		conf:     conf,
		db:       db,
		trie:     tr,
		states:   make(map[types.Address]*StateObject),
		dirtyset: make(map[types.Address]struct{}),
		beats:    make(map[types.Address]time.Time),
		close:    make(chan struct{}),
	}

	go sd.loop()
	return sd, nil
}

func (sd *StateDB) Stop() {
	close(sd.close)
}

func (sd *StateDB) Database() Database {
	return sd.db
}

// CreateNewState will create a new state for input address and
// return the state. If input address already exists in StateDB
// it returns an error.
func (sd *StateDB) CreateNewState(addr types.Address) (*StateObject, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	state := sd.states[addr]
	if state != nil {
		return nil, fmt.Errorf("state already exists, addr: %s", addr.String())
	}

	state = NewStateObject(addr)
	sd.states[addr] = state
	sd.beats[addr] = time.Now()

	return state, nil
}

func (sd *StateDB) GetBalance(addr types.Address) *math.BigInt {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	defer sd.refreshbeat(addr)
	return sd.getBalance(addr)
}
func (sd *StateDB) getBalance(addr types.Address) *math.BigInt {
	state, err := sd.getStateObject(addr)
	if err != nil {
		return math.NewBigInt(0)
	}
	return state.GetBalance()
}

func (sd *StateDB) GetNonce(addr types.Address) (uint64, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	defer sd.refreshbeat(addr)
	return sd.getNonce(addr)
}
func (sd *StateDB) getNonce(addr types.Address) (uint64, error) {
	state, err := sd.getStateObject(addr)
	if err != nil {
		return 0, types.ErrNonceNotExist
	}
	return state.GetNonce(), nil
}

// GetState get a state from StateDB. If state not exist,
// load it from db.
func (sd *StateDB) GetStateObject(addr types.Address) (*StateObject, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	return sd.getStateObject(addr)
}
func (sd *StateDB) getStateObject(addr types.Address) (*StateObject, error) {
	state, exist := sd.states[addr]
	if !exist {
		var err error
		state, err = sd.loadStateObject(addr)
		if err != nil {
			return nil, err
		}
		sd.states[addr] = state
	}
	return state, nil
}

// GetOrCreateState will find a state from memory by account address.
// If state not exists, it will load a state from db.
func (sd *StateDB) GetOrCreateStateObject(addr types.Address) *StateObject {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	return sd.getOrCreateStateObject(addr)
}
func (sd *StateDB) getOrCreateStateObject(addr types.Address) *StateObject {
	state, _ := sd.getStateObject(addr)
	if state == nil {
		state = NewStateObject(addr)
		state.db = sd
	}
	return state
}

// DeleteState remove a state from StateDB. Return error
// if it fails.
func (sd *StateDB) DeleteStateObject(addr types.Address) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	return sd.deleteStateObject(addr)
}
func (sd *StateDB) deleteStateObject(addr types.Address) error {
	_, exist := sd.states[addr]
	if !exist {
		return nil
	}
	delete(sd.states, addr)
	delete(sd.dirtyset, addr)
	delete(sd.beats, addr)
	return nil
}

// UpdateStateObject set addr's state in StateDB.
//
// Note that this setting will force updating the StateDB without
// any verification. Call this UpdateState carefully.
func (sd *StateDB) UpdateStateObject(addr types.Address, state *StateObject) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	sd.updateStateObject(addr, state)
}
func (sd *StateDB) updateStateObject(addr types.Address, state *StateObject) {
	sd.states[addr] = state
	sd.dirtyset[addr] = struct{}{}
}

// AddBalance
func (sd *StateDB) AddBalance(addr types.Address, increment *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.addBalance(addr, increment)
}
func (sd *StateDB) addBalance(addr types.Address, increment *math.BigInt) {
	defer sd.refreshbeat(addr)
	state := sd.getOrCreateStateObject(addr)
	state.AddBalance(increment)

	sd.updateStateObject(addr, state)
}

// SubBalance
func (sd *StateDB) SubBalance(addr types.Address, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.subBalance(addr, decrement)
}
func (sd *StateDB) subBalance(addr types.Address, decrement *math.BigInt) {
	defer sd.refreshbeat(addr)
	state := sd.getOrCreateStateObject(addr)
	state.SubBalance(decrement)

	sd.updateStateObject(addr, state)
}

// SetBalance
func (sd *StateDB) SetBalance(addr types.Address, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	sd.setBalance(addr, balance)
}
func (sd *StateDB) setBalance(addr types.Address, balance *math.BigInt) {
	state := sd.getOrCreateStateObject(addr)
	state.SetBalance(balance)

	sd.updateStateObject(addr, state)
}

func (sd *StateDB) SetNonce(addr types.Address, nonce uint64) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	state := sd.getOrCreateStateObject(addr)
	state.SetNonce(nonce)

	sd.updateStateObject(addr, state)
}

// laodState loads a state from current trie.
func (sd *StateDB) loadStateObject(addr types.Address) (*StateObject, error) {
	data, err := sd.trie.TryGet(addr.ToBytes())
	if err != nil {
		return nil, fmt.Errorf("get state from trie err: %v", err)
	}
	var state StateObject
	err = state.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode err: %v", err)
	}
	state.db = sd
	return &state, nil
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
			sd.deleteStateObject(addr)
		}
	}
}

// refreshbeat update the beat time of an address.
func (sd *StateDB) refreshbeat(addr types.Address) {
	sd.beats[addr] = time.Now()
}

// Commit tries to save dirty data to memory trie db.
func (sd *StateDB) Commit() (types.Hash, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.commit()
}

// commit tries to save dirty data to memory trie db.
//
// Note that commit doesn't hold any StateDB locks.
func (sd *StateDB) commit() (types.Hash, error) {
	// TODO
	// use journal to set some state to dirty.

	for addr, state := range sd.states {
		if _, isdirty := sd.dirtyset[addr]; !isdirty {
			continue
		}
		// update state data in current trie.
		data, _ := state.Encode()
		sd.trie.TryUpdate(addr.ToBytes(), data)
		delete(sd.dirtyset, addr)
	}
	// commit current trie into triedb.
	// TODO later need onleaf callback to link account trie to storage trie.
	rootHash, err := sd.trie.Commit(nil)
	return rootHash, err
}

func (sd *StateDB) loop() {
	purgeTimer := time.NewTicker(sd.conf.PurgeTimer)

	for {
		select {
		case <-sd.close:
			purgeTimer.Stop()
			return

		case <-purgeTimer.C:
			sd.mu.Lock()
			sd.purge()
			sd.mu.Unlock()

		}
	}
}

func (sd *StateDB) Snapshot() {
	// TODO
}
