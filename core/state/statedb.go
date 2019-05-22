package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/trie"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
	log "github.com/sirupsen/logrus"
)

var (
	// emptyStateRoot is the known hash of an empty state trie entry.
	emptyStateRoot = crypto.Keccak256Hash(nil)

	// emptyStateHash is the known hash of an empty state trie.
	emptyStateHash = types.Hash{}

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

// StateDB stores account's data. Account's data include address,
// balance, nonce, code and its contract db if it is an contract
// address. An account is stored as a StateObject, for more detail
// please check StateObject struct.
type StateDB struct {
	conf StateDBConfig

	// trie stores account's basic data, every node of the trie
	// represents a StateObject.Account.
	// db is for trie accessing.
	db   Database
	trie Trie
	root types.Hash

	refund uint64
	// journal records every action which will change statedb's data
	// and it's for VM term revert only.
	journal     *journal
	snapshotSet []shot
	snapshotID  int

	// states stores all the active state object, any changes on stateobject
	// will also update states. Active stateobject means those objects
	// that has recently been updated or queried. This "recently" is measured
	// by [beats] map.
	states   map[types.Address]*StateObject
	dirtyset map[types.Address]struct{}
	beats    map[types.Address]time.Time

	close chan struct{}

	mu sync.RWMutex
}

func NewStateDB(conf StateDBConfig, db Database, root types.Hash) (*StateDB, error) {
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

func (sd *StateDB) Root() types.Hash {
	return sd.root
}

// CreateAccount will create a new state for input address and
// return the state. If input address already exists in StateDB
// returns it.
func (sd *StateDB) CreateAccount(addr types.Address) {
	newstate := NewStateObject(addr, sd)
	oldstate := sd.getStateObject(addr)
	if oldstate != nil {
		sd.journal.append(&resetObjectChange{
			prev: oldstate,
		})
	} else {
		sd.journal.append(&createObjectChange{
			account: &addr,
		})
	}
	sd.states[addr] = newstate
	// sd.beats[addr] = time.Now()

}

func (sd *StateDB) GetBalance(addr types.Address) *math.BigInt {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	defer sd.refreshbeat(addr)
	return sd.getBalance(addr)
}
func (sd *StateDB) getBalance(addr types.Address) *math.BigInt {
	state := sd.getStateObject(addr)
	if state == nil {
		return math.NewBigInt(0)
	}
	return state.GetBalance()
}

func (sd *StateDB) GetNonce(addr types.Address) uint64 {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	defer sd.refreshbeat(addr)
	return sd.getNonce(addr)
}
func (sd *StateDB) getNonce(addr types.Address) uint64 {
	state := sd.getStateObject(addr)
	if state == nil {
		return 0
	}
	return state.GetNonce()
}

func (sd *StateDB) Exist(addr types.Address) bool {
	if stobj := sd.getStateObject(addr); stobj != nil {
		return true
	}
	return false
}

func (sd *StateDB) Empty(addr types.Address) bool {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return true
	}
	if len(stobj.code) != 0 {
		return false
	}
	if stobj.data.Nonce != uint64(0) {
		return false
	}
	if stobj.data.Balance.GetInt64() != int64(0) {
		return false
	}
	return true
}

// GetStateObject get a state from StateDB. If state not exist,
// load it from db.
func (sd *StateDB) GetStateObject(addr types.Address) *StateObject {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	return sd.getStateObject(addr)
}
func (sd *StateDB) getStateObject(addr types.Address) *StateObject {
	state, exist := sd.states[addr]
	if !exist {
		var err error
		state, err = sd.loadStateObject(addr)
		if err != nil {
			log.Errorf("load stateobject from trie error: %v", err)
			return nil
		}
		if state == nil {
			return nil
		}
		sd.states[addr] = state
	}
	return state
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
	state := sd.getStateObject(addr)
	if state == nil {
		sd.CreateAccount(addr)
		state = sd.getStateObject(addr)
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
	// delete(sd.beats, addr)
	return nil
}

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
}

func (sd *StateDB) GetRefund() uint64 {
	return sd.refund
}

func (sd *StateDB) GetCodeHash(addr types.Address) types.Hash {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return types.Hash{}
	}
	return stobj.GetCodeHash()
}

func (sd *StateDB) GetCode(addr types.Address) []byte {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return nil
	}
	return stobj.GetCode(sd.db)
}

func (sd *StateDB) GetCodeSize(addr types.Address) int {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
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
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return emptyStateHash
	}
	return stobj.GetState(sd.db, key)
}

func (sd *StateDB) GetCommittedState(addr types.Address, key types.Hash) types.Hash {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return emptyStateHash
	}
	return stobj.GetCommittedState(sd.db, key)
}

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
}

func (sd *StateDB) SetNonce(addr types.Address, nonce uint64) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	defer sd.refreshbeat(addr)
	state := sd.getOrCreateStateObject(addr)
	state.SetNonce(nonce)
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

	oldobj := sd.getStateObject(addr)
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

func (sd *StateDB) Suicide(addr types.Address) bool {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return false
	}
	sd.journal.append(&suicideChange{
		account:     &addr,
		prev:        stobj.suicided,
		prevbalance: math.NewBigIntFromBigInt(stobj.GetBalance().Value),
	})
	stobj.suicided = true
	stobj.data.Balance = math.NewBigInt(0)
	return true
}

func (sd *StateDB) HasSuicided(addr types.Address) bool {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return false
	}
	return stobj.suicided
}

func (sd *StateDB) AddLog(l *vmtypes.Log) {
	// TODO
	// Not implemented yet
}

func (sd *StateDB) AddPreimage(h types.Hash, b []byte) {
	// TODO
	// Not implemented yet
}

func (sd *StateDB) ForEachStorage(addr types.Address, f func(key, value types.Hash) bool) {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return
	}
	it := trie.NewIterator(stobj.openTrie(sd.db).NodeIterator(nil))
	for it.Next() {
		key := types.BytesToHash(sd.trie.GetKey(it.Key))
		if value, dirty := stobj.dirtyStorage[key]; dirty {
			f(key, value)
			continue
		}
		f(key, types.BytesToHash(it.Value))
	}
}

// laodState loads a state from current trie.
func (sd *StateDB) loadStateObject(addr types.Address) (*StateObject, error) {
	data, err := sd.trie.TryGet(addr.ToBytes())
	if err != nil {
		return nil, fmt.Errorf("get state from trie err: %v", err)
	}
	if data == nil {
		return nil, nil
	}
	var state StateObject
	err = state.Decode(data, sd)
	if err != nil {
		return nil, fmt.Errorf("decode err: %v", err)
	}
	return &state, nil
}

// purge tries to remove all the state that haven't sent any beat
// for a long time.
//
// Note that dirty states will not be removed.
func (sd *StateDB) purge() {
	// TODO
	// purge will cause a panic problem, so temporaly comments the code.
	//
	// panic: [fatal error: concurrent map writes]
	// reason: the reason is that [sd.beats] is been concurrently called and
	// there is no lock handling this parameter.

	// for addr, lastbeat := range sd.beats {
	// 	// skip dirty states
	// 	if _, in := sd.journal.dirties[addr]; in {
	// 		continue
	// 	}
	// 	if _, in := sd.dirtyset[addr]; in {
	// 		continue
	// 	}
	// 	if time.Since(lastbeat) > sd.conf.BeatExpireTime {
	// 		sd.deleteStateObject(addr)
	// 	}
	// }
}

// refreshbeat update the beat time of an address.
func (sd *StateDB) refreshbeat(addr types.Address) {
	// sd.beats[addr] = time.Now()
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
		// commit state's code
		if state.code != nil && state.dirtycode {
			sd.db.TrieDB().Insert(state.GetCodeHash(), state.code)
			state.dirtycode = false
		}
		// commit state's storage
		if err := state.CommitStorage(sd.db); err != nil {
			log.Errorf("commit state's storage error: %v", err)
		}
		// update state data in current trie.
		data, _ := state.Encode()
		if err := sd.trie.TryUpdate(addr.ToBytes(), data); err != nil {
			log.Errorf("commit statedb error: %v", err)
		}
		delete(sd.dirtyset, addr)
	}
	// commit current trie into triedb.
	rootHash, err := sd.trie.Commit(func(leaf []byte, parent types.Hash) error {
		var account Account
		if _, err := account.UnmarshalMsg(leaf); err != nil {
			return nil
		}
		// log.Tracef("onleaf called with address: %s, root: %v, codehash: %v", account.Address.Hex(), account.Root.ToBytes(), account.CodeHash)
		if account.Root != emptyStateRoot {
			sd.db.TrieDB().Reference(account.Root, parent)
		}
		codehash := types.BytesToHash(account.CodeHash)
		if codehash != emptyCodeHash {
			sd.db.TrieDB().Reference(codehash, parent)
		}
		return nil
	})
	sd.root = rootHash

	sd.clearJournalAndRefund()
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

func (sd *StateDB) Snapshot() int {
	id := sd.snapshotID
	sd.snapshotID++
	sd.snapshotSet = append(sd.snapshotSet, shot{shotid: id, journalIndex: sd.journal.length()})
	return id
}

func (sd *StateDB) RevertToSnapshot(snapshotid int) {
	// TODO
	// Ethereum uses sort.Search to get the valid snapshot,
	// consider any necessary for this.
	index := 0
	s := shot{shotid: -1}
	for i, shotInMem := range sd.snapshotSet {
		if shotInMem.shotid == snapshotid {
			index = i
			s = shotInMem
			break
		}
	}
	if s.shotid == -1 {
		panic(fmt.Sprintf("can't find valid snapshot, id: %d", snapshotid))
	}
	sd.journal.revert(sd, s.journalIndex)
	sd.snapshotSet = sd.snapshotSet[:index]
}

func (sd *StateDB) clearJournalAndRefund() {
	sd.journal = newJournal()
	sd.snapshotID = 0
	sd.snapshotSet = sd.snapshotSet[:0]
	sd.refund = 0
}

func (sd *StateDB) String() string {
	return "you shall-not paaaaaaaasss!!!"
}

type shot struct {
	shotid       int
	journalIndex int
}
