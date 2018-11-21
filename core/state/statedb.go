package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
)

type StateDBConfig struct {
	FlushTimer     time.Duration
	PurgeTimer     time.Duration
	BeatExpireTime time.Duration
}

func DefaultStateDBConfig() StateDBConfig {
	return StateDBConfig{
		FlushTimer:     time.Duration(5000),
		PurgeTimer:     time.Duration(10000),
		BeatExpireTime: time.Duration(10 * time.Minute),
	}
}

type StateDB struct {
	conf StateDBConfig

	db   Database
	trie Trie

	states   map[types.Address]*State
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
		states:   make(map[types.Address]*State),
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

	defer sd.refreshbeat(addr)
	return sd.getBalance(addr)
}
func (sd *StateDB) getBalance(addr types.Address) *math.BigInt {
	state, err := sd.getState(addr)
	if err != nil {
		return math.NewBigInt(0)
	}
	return state.Balance
}

func (sd *StateDB) GetNonce(addr types.Address) (uint64, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	defer sd.refreshbeat(addr)
	return sd.getNonce(addr)
}
func (sd *StateDB) getNonce(addr types.Address) (uint64, error) {
	state, err := sd.getState(addr)
	if err != nil {
		return 0, types.ErrNonceNotExist
	}
	return state.Nonce, nil
}

// GetState get a state from StateDB. If state not exist,
// load it from db.
func (sd *StateDB) GetState(addr types.Address) (*State, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
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
		sd.states[addr] = state
	}
	return state, nil
}

// GetOrCreateState
func (sd *StateDB) GetOrCreateState(addr types.Address) *State {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	return sd.getOrCreateState(addr)
}
func (sd *StateDB) getOrCreateState(addr types.Address) *State {
	state, _ := sd.getState(addr)
	if state == nil {
		state = NewState(addr)
	}
	return state
}

// DeleteState remove a state from StateDB. Return error
// if it fails.
func (sd *StateDB) DeleteState(addr types.Address) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	return sd.deleteState(addr)
}
func (sd *StateDB) deleteState(addr types.Address) error {
	_, exist := sd.states[addr]
	if !exist {
		return nil
	}
	delete(sd.states, addr)
	delete(sd.dirtyset, addr)
	delete(sd.beats, addr)
	return nil
}

// UpdateState set addr's state in StateDB.
//
// Note that this setting will force updating the StateDB without
// any verification. Call this UpdateState carefully.
func (sd *StateDB) UpdateState(addr types.Address, state *State) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	sd.updateState(addr, state)
}
func (sd *StateDB) updateState(addr types.Address, state *State) {
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
	state := sd.getOrCreateState(addr)
	state.AddBalance(increment)

	sd.updateState(addr, state)
}

// SubBalance
func (sd *StateDB) SubBalance(addr types.Address, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.subBalance(addr, decrement)
}
func (sd *StateDB) subBalance(addr types.Address, decrement *math.BigInt) {
	defer sd.refreshbeat(addr)
	state := sd.getOrCreateState(addr)
	state.SubBalance(decrement)

	sd.updateState(addr, state)
}

// SetBalance
func (sd *StateDB) SetBalance(addr types.Address, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setBalance(addr, balance)
}
func (sd *StateDB) setBalance(addr types.Address, balance *math.BigInt) {
	defer sd.refreshbeat(addr)
	state := sd.getOrCreateState(addr)
	state.SetBalance(balance)

	sd.updateState(addr, state)
}

func (sd *StateDB) SetNonce(addr types.Address, nonce uint64) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	state := sd.getOrCreateState(addr)
	state.SetNonce(nonce)

	sd.updateState(addr, state)
}

// Load data from db.
func (sd *StateDB) loadState(addr types.Address) (*State, error) {
	// return sd.accessor.LoadState(addr)

	// TODO delete this function?
	return nil, nil
}

// func (sd *StateDB) Flush() {
// 	sd.mu.Lock()
// 	defer sd.mu.Unlock()

// 	sd.flush()
// }

// // flush tries to save those dirty data to disk db.
// func (sd *StateDB) flush() {
// 	for addr, _ := range sd.dirtyset {
// 		state, exist := sd.states[addr]
// 		if !exist {
// 			log.Warnf("can't find dirty state in StateDB, addr: %s", addr.String())
// 			continue
// 		}
// 		sd.accessor.SaveState(addr, state)
// 	}
// 	sd.dirtyset = make(map[types.Address]struct{})
// }

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
	sd.beats[addr] = time.Now()
}

// Commit tries to save dirty data to memory trie db.
func (sd *StateDB) Commit() {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.commit()
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
		data, _ := state.MarshalMsg(nil)
		sd.trie.TryUpdate(addr.ToBytes(), data)
	}
	// commit current trie into triedb.
	// TODO later need onleaf callback to link account trie to storage trie.
	rootHash, err := sd.trie.Commit(nil)
	return rootHash, err
}

// func (sd *StateDB) commitToTrie()

func (sd *StateDB) loop() {
	// flushTimer := time.NewTicker(sd.conf.FlushTimer)
	purgeTimer := time.NewTicker(sd.conf.PurgeTimer)

	for {
		select {
		case <-sd.close:
			purgeTimer.Stop()
			// sd.mu.Lock()
			// sd.flush()
			// sd.mu.Unlock()
			return

		case <-purgeTimer.C:
			sd.mu.Lock()
			sd.purge()
			sd.mu.Unlock()

		}
	}
}

// type CacheTxs struct {
// 	txmap  *TxMap
// 	txlist map[types.Address]*TxList
// }

// func NewCacheTxs() *CacheTxs {
// 	return &CacheTxs{
// 		txmap:  NewTxMap(),
// 		txlist: make(map[types.Address]*TxList),
// 	}
// }

// func (ctxs *CacheTxs) GetTxByHash(hash types.Hash) types.Txi {
// 	return ctxs.txmap.Get(hash)
// }

// func (ctxs *CacheTxs) GetTxByNonce(addr types.Address, nonce uint64) types.Txi {
// 	txlist, ok := ctxs.txlist[addr]
// 	if !ok {
// 		return nil
// 	}
// 	return txlist.Get(nonce)
// }

// func (ctxs *CacheTxs) AddTx(tx types.Txi) {
// 	txlist, ok := ctxs.txlist[tx.Sender()]
// 	if !ok {
// 		txlist = NewTxList()
// 		ctxs.txlist[tx.Sender()] = txlist
// 	}
// 	txlist.Put(tx)
// 	ctxs.txmap.Add(tx)
// }

// func (ctxs *CacheTxs) RemoveTxByHash(hash types.Hash) error {
// 	tx := ctxs.txmap.Get(hash)
// 	if tx == nil {
// 		return nil
// 	}
// 	ctxs.txmap.Remove(hash)

// 	txlist, ok := ctxs.txlist[tx.Sender()]
// 	if !ok {
// 		return fmt.Errorf("tx not exist in txlist.")
// 	}
// 	txlist.Remove(tx.GetNonce())

// 	return nil
// }

// func (ctxs *CacheTxs) RemoveTxByNonce(addr types.Address, nonce uint64) error {
// 	txlist, ok := ctxs.txlist[addr]
// 	if !ok {
// 		return nil
// 	}
// 	tx := txlist.Get(nonce)
// 	if tx == nil {
// 		return nil
// 	}
// 	ctxs.txmap.Remove(tx.GetTxHash())
// 	txlist.Remove(nonce)
// 	return nil
// }
