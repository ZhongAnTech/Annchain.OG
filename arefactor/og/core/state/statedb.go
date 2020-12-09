package state

import (
	"fmt"
	"sync"
	"time"

	ogTypes "github.com/annchain/OG/og_interface"
	crypto "github.com/annchain/OG/ogcrypto"
	"github.com/annchain/commongo/marshaller"
	"github.com/annchain/commongo/math"
	"github.com/annchain/trie"
	log "github.com/sirupsen/logrus"
)

var (
	// emptyStateRoot is the known hash of an empty state trie entry.
	emptyStateRoot = ogTypes.BytesToHash32(crypto.Keccak256Hash(nil))

	// emptyStateHash is the known hash of an empty state trie.
	emptyStateHash = ogTypes.Hash32{}

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = ogTypes.BytesToHash32(crypto.Keccak256Hash(nil))

	// emptyCodeHash is the known hash of the empty EVM bytecode.
	emptyCodeHash = ogTypes.BytesToHash32(crypto.Keccak256Hash(nil))
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
	db   trie.Database
	trie trie.TrieI
	root ogTypes.Hash

	refund uint64
	// journal records every action which will change statedb's data
	// and it's for VM term revert only.
	journal     *journal
	snapshotSet []shot
	snapshotID  int

	// states stores all the active state object, any changes on stateobject
	// will also update states.
	states   map[ogTypes.AddressKey]*StateObject
	dirtyset map[ogTypes.AddressKey]struct{}

	// token information
	latestTokenID      int32
	dirtyLatestTokenID bool
	tokens             map[int32]*TokenObject
	dirtyTokens        map[int32]struct{}

	close chan struct{}

	mu sync.RWMutex
}

func NewStateDB(conf StateDBConfig, db Database, root ogTypes.Hash) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}

	latestTokenID := int32(0)
	data, err := tr.TryGet(LatestTokenIDTrieKey())
	if err == nil && len(data) >= 4 {
		latestTokenID = marshaller.GetInt32(data, 0)
	}

	sd := &StateDB{
		conf:          conf,
		db:            db,
		trie:          tr,
		states:        make(map[ogTypes.AddressKey]*StateObject),
		dirtyset:      make(map[ogTypes.AddressKey]struct{}),
		latestTokenID: latestTokenID,
		tokens:        make(map[int32]*TokenObject),
		dirtyTokens:   make(map[int32]struct{}),
		journal:       newJournal(),
		snapshotID:    0,
		snapshotSet:   make([]shot, 0),
		close:         make(chan struct{}),
		root:          root,
	}

	return sd, nil
}

func (sd *StateDB) Stop() {
	close(sd.close)
}

func (sd *StateDB) Database() Database {
	return sd.db
}

func (sd *StateDB) Root() ogTypes.Hash {
	return sd.root
}

// CreateAccount will create a new state for input address.
func (sd *StateDB) CreateAccount(addr ogTypes.Address) {
	newstate := NewStateObject(addr, sd)
	oldstate := sd.getStateObject(addr)
	if oldstate != nil {
		sd.AppendJournal(&resetObjectChange{
			account: oldstate.address,
			prev:    oldstate,
		})
	} else {
		sd.AppendJournal(&createObjectChange{
			account: addr,
		})
	}
	sd.states[addr.AddressKey()] = newstate
}

func (sd *StateDB) GetBalance(addr ogTypes.Address) *math.BigInt {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getBalance(addr, OGTokenID)
}

func (sd *StateDB) GetTokenBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getBalance(addr, tokenID)
}

func (sd *StateDB) GetAllTokenBalance(addr ogTypes.Address) BalanceSet {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getAllBalance(addr)
}

func (sd *StateDB) getBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
	state := sd.getStateObject(addr)
	if state == nil {
		return math.NewBigInt(0)
	}
	balance := state.GetBalance(tokenID)
	if balance == nil {
		return math.NewBigInt(0)
	}
	return balance
}

func (sd *StateDB) getAllBalance(addr ogTypes.Address) BalanceSet {
	state := sd.getStateObject(addr)
	if state == nil {
		return NewBalanceSet()
	}
	balance := state.GetAllBalance()
	if balance == nil {
		return NewBalanceSet()
	}
	return balance
}

func (sd *StateDB) GetNonce(addr ogTypes.Address) uint64 {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.getNonce(addr)
}
func (sd *StateDB) getNonce(addr ogTypes.Address) uint64 {
	state := sd.getStateObject(addr)
	if state == nil {
		return 0
	}
	return state.GetNonce()
}

func (sd *StateDB) Exist(addr ogTypes.Address) bool {
	if stobj := sd.getStateObject(addr); stobj != nil {
		return true
	}
	return false
}

func (sd *StateDB) Empty(addr ogTypes.Address) bool {
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
func (sd *StateDB) GetStateObject(addr ogTypes.Address) *StateObject {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.getStateObject(addr)
}
func (sd *StateDB) getStateObject(addr ogTypes.Address) *StateObject {
	state, exist := sd.states[addr.AddressKey()]
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
		sd.states[addr.AddressKey()] = state
	}
	return state
}

// GetOrCreateStateObject will find a state from memory by account address.
// If state not exists, it will load a state from db.
func (sd *StateDB) GetOrCreateStateObject(addr ogTypes.Address) *StateObject {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.getOrCreateStateObject(addr)
}
func (sd *StateDB) getOrCreateStateObject(addr ogTypes.Address) *StateObject {
	state := sd.getStateObject(addr)
	if state == nil {
		sd.CreateAccount(addr)
		state = sd.getStateObject(addr)
	}
	return state
}

// DeleteStateObject remove a state from StateDB. Return error
// if it fails.
func (sd *StateDB) DeleteStateObject(addr ogTypes.Address) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.deleteStateObject(addr)
}
func (sd *StateDB) deleteStateObject(addr ogTypes.Address) error {
	_, exist := sd.states[addr.AddressKey()]
	if !exist {
		return nil
	}
	delete(sd.states, addr.AddressKey())
	delete(sd.dirtyset, addr.AddressKey())
	return nil
}

// AddBalance
func (sd *StateDB) AddBalance(addr ogTypes.Address, increment *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.addBalance(addr, OGTokenID, increment)
}
func (sd *StateDB) AddTokenBalance(addr ogTypes.Address, tokenID int32, increment *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.addBalance(addr, tokenID, increment)
}
func (sd *StateDB) addBalance(addr ogTypes.Address, tokenID int32, increment *math.BigInt) {
	// check if increment is zero
	if increment.Sign() == 0 {
		return
	}
	state := sd.getOrCreateStateObject(addr)
	sd.setBalance(addr, tokenID, state.data.Balances.PreAdd(tokenID, increment))
}

// SubBalance
func (sd *StateDB) SubBalance(addr ogTypes.Address, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.subBalance(addr, OGTokenID, decrement)
}
func (sd *StateDB) SubTokenBalance(addr ogTypes.Address, tokenID int32, decrement *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.subBalance(addr, tokenID, decrement)
}
func (sd *StateDB) subBalance(addr ogTypes.Address, tokenID int32, decrement *math.BigInt) {
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

func (sd *StateDB) GetCodeHash(addr ogTypes.Address) ogTypes.Hash {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return &ogTypes.Hash32{}
	}
	return stobj.GetCodeHash()
}

func (sd *StateDB) GetCode(addr ogTypes.Address) []byte {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return nil
	}
	return stobj.GetCode(sd.db)
}

func (sd *StateDB) GetCodeSize(addr ogTypes.Address) int {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return 0
	}
	l, dberr := stobj.GetCodeSize(sd.db)
	if dberr != nil {
		log.Errorf("get code size from obj error: %v, obj: %s", dberr, stobj.address.AddressShortString())
		return 0
	}
	return l
}

func (sd *StateDB) GetState(addr ogTypes.Address, key ogTypes.Hash) ogTypes.Hash {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return &emptyStateHash
	}
	return stobj.GetState(sd.db, key)
}

func (sd *StateDB) GetCommittedState(addr ogTypes.Address, key ogTypes.Hash) ogTypes.Hash {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return &emptyStateHash
	}
	return stobj.GetCommittedState(sd.db, key)
}

// SetBalance set origin OG token balance.
// TODO should be modified to satisfy all tokens.
func (sd *StateDB) SetBalance(addr ogTypes.Address, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setBalance(addr, OGTokenID, balance)
}
func (sd *StateDB) SetTokenBalance(addr ogTypes.Address, tokenID int32, balance *math.BigInt) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setBalance(addr, tokenID, balance)
}
func (sd *StateDB) setBalance(addr ogTypes.Address, tokenID int32, balance *math.BigInt) {
	state := sd.getOrCreateStateObject(addr)
	state.SetBalance(tokenID, balance)
}

func (sd *StateDB) SetNonce(addr ogTypes.Address, nonce uint64) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	state := sd.getOrCreateStateObject(addr)
	state.SetNonce(nonce)
}

func (sd *StateDB) AddRefund(increment uint64) {
	sd.AppendJournal(&refundChange{
		prev: sd.refund,
	})
	sd.refund += increment
}

func (sd *StateDB) SubRefund(decrement uint64) {
	sd.AppendJournal(&refundChange{
		prev: sd.refund,
	})
	if decrement > sd.refund {
		panic("Refund counter below zero")
	}
	sd.refund -= decrement
}

func (sd *StateDB) SetStateObject(addr ogTypes.Address, stobj *StateObject) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.setStateObject(addr, stobj)
}
func (sd *StateDB) setStateObject(addr ogTypes.Address, stobj *StateObject) {
	oldobj := sd.getStateObject(addr)
	sd.AppendJournal(&resetObjectChange{
		account: addr,
		prev:    oldobj,
	})
	sd.states[addr.AddressKey()] = stobj
}

func (sd *StateDB) SetCode(addr ogTypes.Address, code []byte) {
	stobj := sd.getOrCreateStateObject(addr)
	stobj.SetCode(ogTypes.BytesToHash32(crypto.Keccak256Hash(code)), code)
}

func (sd *StateDB) SetState(addr ogTypes.Address, key, value ogTypes.Hash) {
	stobj := sd.getOrCreateStateObject(addr)
	if stobj == nil {
		return
	}
	stobj.SetState(sd.db, key, value)
}

/**
Token part
*/

// IssueToken creates a new token according to offered token information.
func (sd *StateDB) IssueToken(issuer ogTypes.Address, name, symbol string, reIssuable bool, fstIssue *math.BigInt) (int32, error) {
	tokenID := sd.latestTokenID + 1

	oldToken := sd.getTokenObject(tokenID)
	if oldToken != nil {
		return 0, fmt.Errorf("already exist token with tokenID: %d", tokenID)
	}
	newToken := NewTokenObject(tokenID, issuer, name, symbol, reIssuable, fstIssue, sd)
	sd.AppendJournal(&createTokenChange{
		prevLatestTokenID: sd.latestTokenID,
		tokenID:           tokenID,
	})
	sd.tokens[tokenID] = newToken
	sd.latestTokenID = tokenID
	sd.dirtyLatestTokenID = true

	sd.setBalance(issuer, tokenID, fstIssue)
	return tokenID, nil
}

func (sd *StateDB) ReIssueToken(tokenID int32, amount *math.BigInt) error {
	tkObj := sd.getTokenObject(tokenID)
	if tkObj == nil {
		return fmt.Errorf("token not exists")
	}
	err := tkObj.ReIssue(amount)
	if err != nil {
		return err
	}
	sd.addBalance(tkObj.Issuer, tokenID, amount)
	return nil
}

func (sd *StateDB) DestroyToken(tokenID int32) error {
	tkObj := sd.getTokenObject(tokenID)
	if tkObj == nil {
		return fmt.Errorf("token not exists")
	}
	tkObj.Destroy()
	return nil
}

// GetTokenObject get token object from StateDB.tokens . If not exists then
// try to load from trie db.
func (sd *StateDB) GetTokenObject(tokenID int32) *TokenObject {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.getTokenObject(tokenID)
}
func (sd *StateDB) getTokenObject(tokenID int32) *TokenObject {
	token, exist := sd.tokens[tokenID]
	if exist {
		return token
	}
	token, err := sd.loadTokenObject(tokenID)
	if err != nil {
		log.Errorf("load token from trie err: %v", err)
		return nil
	}
	if token == nil {
		return nil
	}
	sd.tokens[tokenID] = token
	return token
}

func (sd *StateDB) setTokenObject(tokenID int32, tkObj *TokenObject) {
	oldTkObj := sd.getTokenObject(tokenID)
	sd.AppendJournal(&resetTokenChange{
		tokenID: tokenID,
		prev:    oldTkObj,
	})
	sd.tokens[tokenID] = tkObj
}

func (sd *StateDB) LatestTokenID() int32 {
	return sd.latestTokenID
}

func (sd *StateDB) AppendJournal(entry JournalEntry) {
	sd.journal.append(entry)
}

func (sd *StateDB) Suicide(addr ogTypes.Address) bool {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return false
	}
	sd.AppendJournal(&suicideChange{
		account:     addr,
		prev:        stobj.suicided,
		prevbalance: stobj.data.Balances.Copy(),
	})
	stobj.suicided = true
	stobj.data.Balances = NewBalanceSet()
	return true
}

func (sd *StateDB) HasSuicided(addr ogTypes.Address) bool {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return false
	}
	return stobj.suicided
}

//func (sd *StateDB) AddLog(l *vmtypes.Log) {
//	// TODO
//	// Not implemented yet
//}

func (sd *StateDB) AddPreimage(h ogTypes.Hash, b []byte) {
	// TODO
	// Not implemented yet
}

func (sd *StateDB) ForEachStorage(addr ogTypes.Address, f func(key, value ogTypes.Hash) bool) {
	stobj := sd.getStateObject(addr)
	if stobj == nil {
		return
	}
	it := trie.NewIterator(stobj.openTrie(sd.db).NodeIterator(nil))
	for it.Next() {
		key := ogTypes.BytesToHash32(sd.trie.GetKey(it.Key))
		if value, dirty := stobj.dirtyStorage[key.HashKey()]; dirty {
			f(key, value)
			continue
		}
		f(key, ogTypes.BytesToHash32(it.Value))
	}
}

// loadState loads a state from current trie.
func (sd *StateDB) loadStateObject(addr ogTypes.Address) (*StateObject, error) {
	data, err := sd.trie.TryGet(addr.Bytes())
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

// loadTokenObject loads token object from trie.
func (sd *StateDB) loadTokenObject(tokenID int32) (*TokenObject, error) {
	data, err := sd.trie.TryGet(TokenTrieKey(tokenID))
	if err != nil {
		return nil, fmt.Errorf("get token from trie err: %v", err)
	}
	if data == nil {
		return nil, nil
	}
	var token TokenObject
	err = token.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode token err: %v", err)
	}
	return &token, nil
}

// Commit tries to save dirty data to memory trie db.
func (sd *StateDB) Commit() (ogTypes.Hash, error) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return sd.commit()
}

// commit tries to save dirty data to memory trie db.
//
// Note that commit doesn't hold any StateDB locks.
func (sd *StateDB) commit() (ogTypes.Hash, error) {
	// update dirtyset according to journal
	for addr := range sd.journal.dirties {
		sd.dirtyset[addr.AddressKey()] = struct{}{}
	}
	for addr, state := range sd.states {
		if _, isdirty := sd.dirtyset[addr]; !isdirty {
			continue
		}
		log.Tracef("commit state, addr: %s, state: %s", addr, state.String())
		// commit state's code
		if state.code != nil && state.dirtycode {
			sd.db.TrieDB().Insert(state.GetCodeHash(), state.code)
			state.dirtycode = false
		}
		// commit state's storage
		if err := state.CommitStorage(sd.db, false); err != nil {
			log.Errorf("commit state's storage error: %v", err)
		}
		// update state data in current trie.
		data, _ := state.Encode()
		if err := sd.trie.TryUpdate(addr.Bytes(), data); err != nil {
			log.Errorf("commit statedb error: %v", err)
		}
		delete(sd.dirtyset, addr)
	}

	// commit dirty token information
	for tokenID := range sd.journal.tokenDirties {
		sd.dirtyTokens[tokenID] = struct{}{}
	}
	for tokenID, token := range sd.tokens {
		if _, isDirty := sd.dirtyTokens[tokenID]; !isDirty {
			continue
		}
		data, _ := token.Encode()
		if err := sd.trie.TryUpdate(TokenTrieKey(tokenID), data); err != nil {
			log.Errorf("commit token %d to trie error: %d", tokenID, err)
		}
		delete(sd.dirtyTokens, tokenID)
	}
	if sd.dirtyLatestTokenID {
		data := make([]byte, 4)
		marshaller.SetInt32(data, 0, sd.latestTokenID)
		if err := sd.trie.TryUpdate(LatestTokenIDTrieKey(), data); err != nil {
			log.Errorf("commit latest token id %d to trie error: %d", sd.latestTokenID, err)
		}
		sd.dirtyLatestTokenID = false
	}

	// commit current trie into triedb.
	rootHash, err := sd.trie.Commit(func(leaf []byte, parent ogTypes.Hash) error {
		account := NewAccountData()
		if _, err := account.UnmarshalMsg(leaf); err != nil {
			return nil
		}
		// log.Tracef("onleaf called with address: %s, root: %v, codehash: %v", account.Address.Hex(), account.Root.ToBytes(), account.CodeHash)
		if account.Root != emptyStateRoot {
			sd.db.TrieDB().Reference(account.Root, parent)
		}
		codehash := ogTypes.BytesToHash32(account.CodeHash)
		if codehash.HashKey() != emptyCodeHash.HashKey() {
			sd.db.TrieDB().Reference(codehash, parent)
		}
		return nil
	}, false)

	//if trie commit fail ,nil root will write to db
	if err != nil {
		log.WithError(err).Warning("commit trie error")
	}
	log.WithField("rootHash", rootHash).Trace("state root set to")
	sd.root = rootHash

	return rootHash, err
}

func (sd *StateDB) Snapshot() int {
	id := sd.snapshotID
	sd.snapshotID++
	sd.snapshotSet = append(sd.snapshotSet, shot{shotid: id, journalIndex: sd.journal.length()})
	log.Tracef("snapshot id: %d", id)
	return id
}

func (sd *StateDB) RevertToSnapshot(snapshotid int) {
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

func (sd *StateDB) ClearJournalAndRefund() {
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
