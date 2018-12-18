package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

var (
	// emptyState is the known hash of an empty state trie entry.
	emptyState = crypto.Keccak256Hash(nil)

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = crypto.Keccak256Hash(nil)

	// emptyCodeHash is the known hash of the empty EVM bytecode.
	emptyCodeHash = crypto.Keccak256Hash(nil)
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

	refund uint64

	states   map[types.Address]*StateObject
	dirtyset map[types.Address]struct{}
	beats    map[types.Address]time.Time
	journal  *journal

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
		journal:  newJournal(),
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

// GetOrCreateStateObject will find a state from memory by account address.
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

// DeleteStateObject remove a state from StateDB. Return error
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

// // UpdateStateObject set addr's state in StateDB.
// //
// // Note that this setting will force updating the StateDB without
// // any verification. Call this UpdateState carefully.
// func (sd *StateDB) UpdateStateObject(addr types.Address, state *StateObject) {
// 	sd.mu.Lock()
// 	defer sd.mu.Unlock()

// 	defer sd.refreshbeat(addr)
// 	sd.updateStateObject(addr, state)
// }
// func (sd *StateDB) updateStateObject(addr types.Address, state *StateObject) {
// 	sd.states[addr] = state
// 	sd.dirtyset[addr] = struct{}{}
// }

// AddBalance
func (sd *StateDB) AddBalance(addr types.Address, increment *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.addBalance(addr, increment)
}
func (sd *StateDB) addBalance(addr types.Address, increment *math.BigInt) {
	// check if increment is zero
	if increment.Sign() == 0 {
		return
	}

	defer sd.refreshbeat(addr)
	state := sd.getOrCreateStateObject(addr)
	sd.setBalance(addr, state.data.Balance.Add(increment))

	// TODO replace updateStateObject by append journal
	// sd.updateStateObject(addr, state)
}

// SubBalance
func (sd *StateDB) SubBalance(addr types.Address, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.subBalance(addr, decrement)
}
func (sd *StateDB) subBalance(addr types.Address, decrement *math.BigInt) {
	// check if increment is zero
	if decrement.Sign() == 0 {
		return
	}

	defer sd.refreshbeat(addr)
	state := sd.getOrCreateStateObject(addr)
	sd.setBalance(addr, state.data.Balance.Sub(decrement))

	// TODO replace updateStateObject by append journal
	// sd.updateStateObject(addr, state)
}

func (sd *StateDB) GetRefund() uint64 {
	return sd.refund
}

func (sd *StateDB) GetCodeHash(addr types.Address) types.Hash {
	stobj, err := sd.getStateObject(addr)
	if err != nil {
		log.Errorf("get state object error: %v", err)
		return types.Hash{}
	}
	return stobj.GetCodeHash()
}

func (sd *StateDB) GetCode(addr types.Address) []byte {
	stobj, err := sd.getStateObject(addr)
	if err != nil {
		log.Errorf("get state object error: %v", err)
		return nil
	}
	return stobj.GetCode(sd.db)
}

func (sd *StateDB) GetCodeSize(addr types.Address) int {
	stobj, err := sd.getStateObject(addr)
	if err != nil {
		log.Errorf("get state object error: %v", err)
		return 0
	}
	l, dberr := stobj.GetCodeSize(sd.db)
	if dberr != nil {
		log.Errorf("get code size from obj error: %v, obj: %s", dberr, stobj.address.String())
		return 0
	}
	return l
}

func (sd *StateDB) GetState(addr types.Address, key types.Hash) types.Hash {
	// TODO
	// not implemented yet
	return types.BytesToHash([]byte{})
}

/**
Setters

Belows are the functions that will dirt the StateDB's data,
add the change to journal for later revert.
*/

// SetBalance
func (sd *StateDB) SetBalance(addr types.Address, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setBalance(addr, balance)
}
func (sd *StateDB) setBalance(addr types.Address, balance *math.BigInt) {
	defer sd.refreshbeat(addr)

	state := sd.getOrCreateStateObject(addr)
	state.SetBalance(balance)

	// sd.updateStateObject(addr, state)
}

func (sd *StateDB) SetNonce(addr types.Address, nonce uint64) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	state := sd.getOrCreateStateObject(addr)
	state.SetNonce(nonce)

	// sd.updateStateObject(addr, state)
}

func (sd *StateDB) AddRefund(increment uint64) {
	sd.journal.append(&refundChange{
		prev: sd.refund,
	})
	sd.refund += increment
}

func (sd *StateDB) SubRefund(decrement uint64) {
	sd.journal.append(&refundChange{
		prev: sd.refund,
	})
	if decrement > sd.refund {
		panic("Refund counter below zero")
	}
	sd.refund -= decrement
}

func (sd *StateDB) SetStateObject(addr types.Address, stobj *StateObject) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setStateObject(addr, stobj)
}
func (sd *StateDB) setStateObject(addr types.Address, stobj *StateObject) {
	defer sd.refreshbeat(addr)

	oldobj, _ := sd.getStateObject(addr)
	if oldobj == nil {
		return
	}
	sd.journal.append(&resetObjectChange{
		prev: oldobj,
	})
	sd.states[addr] = stobj
}

func (sd *StateDB) SetCode(addr types.Address, code []byte) {
	stobj := sd.getOrCreateStateObject(addr)
	stobj.SetCode(crypto.Keccak256Hash(code), code)
}

func (sd *StateDB) SetState(addr types.Address, key, value types.Hash) {
	stobj := sd.getOrCreateStateObject(addr)
	if stobj == nil {
		return
	}
	stobj.SetState(sd.db, key, value)
}

/**
Setters end
*/

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
		if _, in := sd.journal.dirties[addr]; in {
			continue
		}
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
	// update dirtyset according to journal
	for addr := range sd.journal.dirties {
		sd.dirtyset[addr] = struct{}{}
	}
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
	rootHash, err := sd.trie.Commit(func(leaf []byte, parent types.Hash) error {
		var account Account
		if _, err := account.UnmarshalMsg(leaf); err != nil {
			return nil
		}
		if account.Root != emptyState {
			sd.db.TrieDB().Reference(account.Root, parent)
		}
		code := types.BytesToHash(account.CodeHash)
		if code != emptyCode {
			sd.db.TrieDB().Reference(code, parent)
		}
		return nil
	})

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
