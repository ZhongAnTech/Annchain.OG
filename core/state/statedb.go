package state

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/types/token"
	"sync"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/trie"
	vmtypes "github.com/annchain/OG/vm/types"
	log "github.com/sirupsen/logrus"
)

var (
	// emptyStateRoot is the known hash of an empty state trie entry.
	emptyStateRoot = crypto.Keccak256Hash(nil)

	// emptyStateHash is the known hash of an empty state trie.
	emptyStateHash = common.Hash{}

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
	root common.Hash

	refund uint64
	// journal records every action which will change statedb's data
	// and it's for VM term revert only.
	journal     *journal
	snapshotSet []shot
	snapshotID  int

	// states stores all the active state object, any changes on stateobject
	// will also update states.
	states   map[common.Address]*StateObject
	dirtyset map[common.Address]struct{}

	close chan struct{}

	mu sync.RWMutex
}

func NewStateDB(conf StateDBConfig, db Database, root common.Hash) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	sd := &StateDB{
		conf:     conf,
		db:       db,
		trie:     tr,
		states:   make(map[common.Address]*StateObject),
		dirtyset: make(map[common.Address]struct{}),
		journal:  newJournal(),
		close:    make(chan struct{}),
		root:     root,
	}

	return sd, nil
}

func (sd *StateDB) Stop() {
	close(sd.close)
}

func (sd *StateDB) Database() Database {
	return sd.db
}

func (sd *StateDB) Root() common.Hash {
	return sd.root
}

// CreateAccount will create a new state for input address and
// return the state. If input address already exists in StateDB
// returns it.
func (sd *StateDB) CreateAccount(addr common.Address) {
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

func (sd *StateDB) GetBalance(addr common.Address) *math.BigInt {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getBalance(addr, token.OGTokenID)
}

func (sd *StateDB) GetTokenBalance(addr common.Address, tokenID int32) *math.BigInt {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getBalance(addr, tokenID)
}

func (sd *StateDB) GetAllTokenBalance(addr common.Address) BalanceSet {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getAllBalance(addr)
}

func (sd *StateDB) getBalance(addr common.Address, tokenID int32) *math.BigInt {
	state := sd.getStateObject(addr)
	if state == nil {
		return math.NewBigInt(0)
	}
	balance := state.GetBalance(tokenID)
	if balance==nil {
		return math.NewBigInt(0)
	}
	return balance
}

func (sd *StateDB) getAllBalance(addr common.Address) BalanceSet {
	state := sd.getStateObject(addr)
	if state == nil {
		return NewBalanceSet()
	}
	balance := state.GetAllBalance()
	if balance==nil {
		return NewBalanceSet()
	}
	return balance
}


func (sd *StateDB) GetNonce(addr common.Address) uint64 {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getNonce(addr)
}
func (sd *StateDB) getNonce(addr common.Address) uint64 {
	state := sd.getStateObject(addr)
	if state == nil {
		return 0
	}
	return state.GetNonce()
}

func (sd *StateDB) Exist(addr common.Address) bool {
	if stobj := sd.getStateObject(addr); stobj != nil {
		return true
	}
	return false
}

func (sd *StateDB) Empty(addr common.Address) bool {
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
	if stobj.data.Balances.IsEmpty() {
		return false
	}
	return true
}

// GetStateObject get a state from StateDB. If state not exist,
// load it from db.
func (sd *StateDB) GetStateObject(addr common.Address) *StateObject {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	//defer sd.refreshbeat(addr)
	return sd.getStateObject(addr)
}
func (sd *StateDB) getStateObject(addr common.Address) *StateObject {
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
func (sd *StateDB) GetOrCreateStateObject(addr common.Address) *StateObject {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	//defer sd.refreshbeat(addr)
	return sd.getOrCreateStateObject(addr)
}
func (sd *StateDB) getOrCreateStateObject(addr common.Address) *StateObject {
	state := sd.getStateObject(addr)
	if state == nil {
		sd.CreateAccount(addr)
		state = sd.getStateObject(addr)
	}
	return state
}

// DeleteStateObject remove a state from StateDB. Return error
// if it fails.
func (sd *StateDB) DeleteStateObject(addr common.Address) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	//defer sd.refreshbeat(addr)
	return sd.deleteStateObject(addr)
}
func (sd *StateDB) deleteStateObject(addr common.Address) error {
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
func (sd *StateDB) AddBalance(addr common.Address, increment *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.addBalance(addr, token.OGTokenID, increment)
}
func (sd *StateDB) AddTokenBalance(addr common.Address, tokenID int32, increment *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.addBalance(addr, tokenID, increment)
}
func (sd *StateDB) addBalance(addr common.Address, tokenID int32, increment *math.BigInt) {
	// check if increment is zero
	if increment.Sign() == 0 {
		return
	}
	state := sd.getOrCreateStateObject(addr)
	sd.setBalance(addr, tokenID, state.data.Balances.PreAdd(tokenID, increment))
}

// SubBalance
func (sd *StateDB) SubBalance(addr common.Address, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.subBalance(addr, token.OGTokenID, decrement)
}
func (sd *StateDB) SubTokenBalance(addr common.Address, tokenID int32, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.subBalance(addr, tokenID, decrement)
}
func (sd *StateDB) subBalance(addr common.Address, tokenID int32, decrement *math.BigInt) {
	// check if increment is zero
	if decrement.Sign() == 0 {
		return
	}
	state := sd.getOrCreateStateObject(addr)
	sd.setBalance(addr, tokenID, state.data.Balances.PreSub(tokenID, decrement))
}

func (sd *StateDB) GetRefund() uint64 {
	return sd.refund
}

func (sd *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return common.Hash{}
	}
	return stobj.GetCodeHash()
}

func (sd *StateDB) GetCode(addr common.Address) []byte {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return nil
	}
	return stobj.GetCode(sd.db)
}

func (sd *StateDB) GetCodeSize(addr common.Address) int {
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

func (sd *StateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return emptyStateHash
	}
	return stobj.GetState(sd.db, key)
}

func (sd *StateDB) GetCommittedState(addr common.Address, key common.Hash) common.Hash {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return emptyStateHash
	}
	return stobj.GetCommittedState(sd.db, key)
}

// SetBalance set origin OG token balance.
// TODO should be modified to satisfy all tokens.
func (sd *StateDB) SetBalance(addr common.Address, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setBalance(addr, token.OGTokenID, balance)
}
func (sd *StateDB) SetTokenBalance(addr common.Address, tokenID int32, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setBalance(addr, tokenID, balance)
}
func (sd *StateDB) setBalance(addr common.Address, tokenID int32, balance *math.BigInt) {
	state := sd.getOrCreateStateObject(addr)
	state.SetBalance(tokenID, balance)
}

func (sd *StateDB) SetNonce(addr common.Address, nonce uint64) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

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

func (sd *StateDB) SetStateObject(addr common.Address, stobj *StateObject) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setStateObject(addr, stobj)
}
func (sd *StateDB) setStateObject(addr common.Address, stobj *StateObject) {
	//defer sd.refreshbeat(addr)

	oldobj := sd.getStateObject(addr)
	if oldobj == nil {
		return
	}
	sd.journal.append(&resetObjectChange{
		prev: oldobj,
	})
	sd.states[addr] = stobj
}

func (sd *StateDB) SetCode(addr common.Address, code []byte) {
	stobj := sd.getOrCreateStateObject(addr)
	stobj.SetCode(crypto.Keccak256Hash(code), code)
}

func (sd *StateDB) SetState(addr common.Address, key, value common.Hash) {
	stobj := sd.getOrCreateStateObject(addr)
	if stobj == nil {
		return
	}
	stobj.SetState(sd.db, key, value)
}

func (sd *StateDB) Suicide(addr common.Address) bool {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return false
	}
	sd.journal.append(&suicideChange{
		account:     &addr,
		prev:        stobj.suicided,
		prevbalance: stobj.data.Balances.Copy(),
	})
	stobj.suicided = true
	stobj.data.Balances = NewBalanceSet()
	return true
}

func (sd *StateDB) HasSuicided(addr common.Address) bool {
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

func (sd *StateDB) AddPreimage(h common.Hash, b []byte) {
	// TODO
	// Not implemented yet
}

func (sd *StateDB) ForEachStorage(addr common.Address, f func(key, value common.Hash) bool) {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return
	}
	it := trie.NewIterator(stobj.openTrie(sd.db).NodeIterator(nil))
	for it.Next() {
		key := common.BytesToHash(sd.trie.GetKey(it.Key))
		if value, dirty := stobj.dirtyStorage[key]; dirty {
			f(key, value)
			continue
		}
		f(key, common.BytesToHash(it.Value))
	}
}

// laodState loads a state from current trie.
func (sd *StateDB) loadStateObject(addr common.Address) (*StateObject, error) {
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

//// purge tries to remove all the state that haven't sent any beat
//// for a long time.
////
//// Note that dirty states will not be removed.
//func (sd *StateDB) purge() {
//	// TODO
//	// purge will cause a panic problem, so temporaly comments the code.
//	//
//	// panic: [fatal error: concurrent map writes]
//	// reason: the reason is that [sd.beats] is been concurrently called and
//	// there is no lock handling this parameter.
//
//	// for addr, lastbeat := range sd.beats {
//	// 	// skip dirty states
//	// 	if _, in := sd.journal.dirties[addr]; in {
//	// 		continue
//	// 	}
//	// 	if _, in := sd.dirtyset[addr]; in {
//	// 		continue
//	// 	}
//	// 	if time.Since(lastbeat) > sd.conf.BeatExpireTime {
//	// 		sd.deleteStateObject(addr)
//	// 	}
//	// }
//}
//
//// refreshbeat update the beat time of an address.
//func (sd *StateDB) refreshbeat(addr common.Address) {
//	// sd.beats[addr] = time.Now()
//}

// Commit tries to save dirty data to memory trie db.
func (sd *StateDB) Commit() (common.Hash, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.commit()
}

// commit tries to save dirty data to memory trie db.
//
// Note that commit doesn't hold any StateDB locks.
func (sd *StateDB) commit() (common.Hash, error) {
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
		log.Tracef("Panic debug, statdb commit, addr: %x, data: %x", addr.ToBytes(), data)
		if err := sd.trie.TryUpdate(addr.ToBytes(), data); err != nil {
			log.Errorf("commit statedb error: %v", err)
		}
		delete(sd.dirtyset, addr)
	}
	// commit current trie into triedb.
	rootHash, err := sd.trie.Commit(func(leaf []byte, parent common.Hash) error {
		account := NewAccountData()
		if _, err := account.UnmarshalMsg(leaf); err != nil {
			return nil
		}
		// log.Tracef("onleaf called with address: %s, root: %v, codehash: %v", account.Address.Hex(), account.Root.ToBytes(), account.CodeHash)
		if account.Root != emptyStateRoot {
			sd.db.TrieDB().Reference(account.Root, parent)
		}
		codehash := common.BytesToHash(account.CodeHash)
		if codehash != emptyCodeHash {
			sd.db.TrieDB().Reference(codehash, parent)
		}
		return nil
	})

	//if trie commit fail ,nil root will write to db
	if err != nil {
		log.WithError(err).Warning("commit trie error")
	}
	log.WithField("rootHash", rootHash).Trace("state root set to")
	sd.root = rootHash

	sd.clearJournalAndRefund()
	return rootHash, err
}

func (sd *StateDB) loop() {
	if sd.conf.PurgeTimer < time.Millisecond {
		sd.conf.PurgeTimer = time.Second
	}
	purgeTimer := time.NewTicker(sd.conf.PurgeTimer)

	for {
		select {
		case <-sd.close:
			purgeTimer.Stop()
			return

		case <-purgeTimer.C:
			sd.mu.Lock()
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
