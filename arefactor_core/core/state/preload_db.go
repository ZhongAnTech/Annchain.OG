package state

import (
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	crypto "github.com/annchain/OG/arefactor/ogcrypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types/token"
	vmtypes "github.com/annchain/OG/vm/types"
	log "github.com/sirupsen/logrus"
)

type PreloadDB struct {
	root ogTypes.Hash

	db Database
	sd *StateDB

	trie      Trie
	worldTrie Trie

	// journal records every action which will change statedb's data
	// and it's for VM term revert only.
	journal *journal

	// states stores all the active state object, any changes on stateobject
	// will also update states.
	states   map[ogTypes.AddressKey]*StateObject
	dirtyset map[ogTypes.AddressKey]struct{}
}

func NewPreloadDB(db Database, statedb *StateDB) *PreloadDB {
	return &PreloadDB{
		root:     statedb.Root(),
		db:       db,
		sd:       statedb,
		journal:  newJournal(),
		states:   make(map[ogTypes.AddressKey]*StateObject),
		dirtyset: make(map[ogTypes.AddressKey]struct{}),
	}
}

func (pd *PreloadDB) Reset() {
	pd.trie = nil
	pd.journal = newJournal()
	pd.states = make(map[ogTypes.AddressKey]*StateObject)
	pd.dirtyset = make(map[ogTypes.AddressKey]struct{})
}

func (pd *PreloadDB) getOrCreateStateObject(addr ogTypes.Address) *StateObject {
	state := pd.getStateObject(addr)
	if state == nil {
		pd.CreateAccount(addr)
		state = pd.getStateObject(addr)
	}
	return state
}

func (pd *PreloadDB) getStateObject(addr ogTypes.Address) *StateObject {
	state, exist := pd.states[addr.AddressKey()]
	if !exist {
		sdState := pd.sd.GetStateObject(addr)
		if sdState == nil {
			return nil
		}
		state = NewStateObject(addr, pd)
		state.Copy(sdState)
		pd.states[addr.AddressKey()] = state
	}
	return state
}

func (pd *PreloadDB) CreateAccount(addr ogTypes.Address) {
	newstate := NewStateObject(addr, pd)
	oldstate := pd.sd.GetStateObject(addr)
	if oldstate != nil {
		pd.AppendJournal(&resetObjectChange{
			account: oldstate.address,
			prev:    oldstate,
		})
	} else {
		pd.AppendJournal(&createObjectChange{
			account: addr,
		})
	}
	pd.states[addr.AddressKey()] = newstate

}

func (pd *PreloadDB) SubBalance(addr ogTypes.Address, decrement *math.BigInt) {
	pd.subBalance(addr, token.OGTokenID, decrement)
}
func (pd *PreloadDB) SubTokenBalance(addr ogTypes.Address, tokenID int32, decrement *math.BigInt) {
	pd.subBalance(addr, tokenID, decrement)
}
func (pd *PreloadDB) subBalance(addr ogTypes.Address, tokenID int32, decrement *math.BigInt) {
	// check if increment is zero
	if decrement.Sign() == 0 {
		return
	}
	state := pd.getOrCreateStateObject(addr)
	pd.setBalance(addr, tokenID, state.data.Balances.PreSub(tokenID, decrement))
}

func (pd *PreloadDB) AddBalance(addr ogTypes.Address, increment *math.BigInt) {
	pd.addBalance(addr, token.OGTokenID, increment)
}
func (pd *PreloadDB) AddTokenBalance(addr ogTypes.Address, tokenID int32, increment *math.BigInt) {
	pd.addBalance(addr, tokenID, increment)
}
func (pd *PreloadDB) addBalance(addr ogTypes.Address, tokenID int32, increment *math.BigInt) {
	// check if increment is zero
	if increment.Sign() == 0 {
		return
	}
	state := pd.getOrCreateStateObject(addr)
	pd.setBalance(addr, tokenID, state.data.Balances.PreAdd(tokenID, increment))
}

func (ps *PreloadDB) SetTokenBalance(addr ogTypes.Address, tokenID int32, balance *math.BigInt) {
	ps.setBalance(addr, tokenID, balance)
}
func (pd *PreloadDB) setBalance(addr ogTypes.Address, tokenID int32, balance *math.BigInt) {
	state := pd.getOrCreateStateObject(addr)
	state.SetBalance(tokenID, balance)
}

// Retrieve the balance from the given address or 0 if object not found
func (pd *PreloadDB) GetBalance(addr ogTypes.Address) *math.BigInt {
	return pd.getBalance(addr, token.OGTokenID)
}
func (pd *PreloadDB) GetTokenBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
	return pd.getBalance(addr, tokenID)
}
func (pd *PreloadDB) getBalance(addr ogTypes.Address, tokenID int32) *math.BigInt {
	state := pd.getStateObject(addr)
	if state == nil {
		return math.NewBigInt(0)
	}
	return state.GetBalance(tokenID)
}

func (pd *PreloadDB) GetNonce(addr ogTypes.Address) uint64 {
	state := pd.getStateObject(addr)
	if state == nil {
		return 0
	}
	return state.GetNonce()
}
func (pd *PreloadDB) SetNonce(addr ogTypes.Address, nonce uint64) {
	state := pd.getOrCreateStateObject(addr)
	state.SetNonce(nonce)
}

func (pd *PreloadDB) GetCodeHash(addr ogTypes.Address) ogTypes.Hash {
	state := pd.getStateObject(addr)
	if state == nil {
		return nil
	}
	return state.GetCodeHash()
}

func (pd *PreloadDB) GetCode(addr ogTypes.Address) []byte {
	state := pd.getStateObject(addr)
	if state == nil {
		return nil
	}
	return state.GetCode(pd.db)
}

func (pd *PreloadDB) SetCode(addr ogTypes.Address, code []byte) {
	state := pd.getOrCreateStateObject(addr)
	state.SetCode(ogTypes.BytesToHash32(crypto.Keccak256Hash(code)), code)
}

func (pd *PreloadDB) GetCodeSize(addr ogTypes.Address) int {
	state := pd.getStateObject(addr)
	if state == nil {
		return 0
	}
	l, dberr := state.GetCodeSize(pd.db)
	if dberr != nil {
		log.Errorf("get code size from obj error: %v, obj: %s", dberr, state.address.AddressShortString())
		return 0
	}
	return l
}

func (pd *PreloadDB) AddRefund(uint64)  {}
func (pd *PreloadDB) SubRefund(uint64)  {}
func (pd *PreloadDB) GetRefund() uint64 { return 0 }

func (pd *PreloadDB) GetCommittedState(addr ogTypes.Address, key ogTypes.Hash) ogTypes.Hash {
	state := pd.getStateObject(addr)
	if state == nil {
		return &emptyStateHash
	}
	return state.GetCommittedState(pd.db, key)
}

// GetState retrieves a value from the given account's storage trie.
func (pd *PreloadDB) GetState(addr ogTypes.Address, key ogTypes.Hash) ogTypes.Hash {
	state := pd.getStateObject(addr)
	if state == nil {
		return &emptyStateHash
	}
	return state.GetState(pd.db, key)
}

func (pd *PreloadDB) SetState(addr ogTypes.Address, key, value ogTypes.Hash) {
	state := pd.getOrCreateStateObject(addr)
	if state == nil {
		return
	}
	state.SetState(pd.db, key, value)
}

func (pd *PreloadDB) Commit() (ogTypes.Hash, error) {
	trie, err := pd.db.OpenTrie(pd.sd.Root())
	if err != nil {
		return nil, err
	}
	pd.trie = trie

	// update dirtyset according to journal
	for addr := range pd.journal.dirties {
		pd.dirtyset[addr.AddressKey()] = struct{}{}
	}
	for addr, state := range pd.states {
		if _, isdirty := pd.dirtyset[addr]; !isdirty {
			continue
		}
		log.Tracef("commit preload state, addr: %s, state: %s", addr, state.String())
		// commit state's code
		if state.code != nil && state.dirtycode {
			pd.db.TrieDB().Insert(state.GetCodeHash(), state.code)
			state.dirtycode = false
		}
		// commit state's storage
		if err := state.CommitStorage(pd.db, true); err != nil {
			log.Errorf("commit state's storage error: %v", err)
		}
		// update state data in current trie.
		data, _ := state.Encode()
		if err := pd.trie.TryUpdate(addr.Bytes(), data); err != nil {
			log.Errorf("commit statedb error: %v", err)
		}
		delete(pd.dirtyset, addr)
	}
	// commit current trie into triedb.
	rootHash, err := pd.trie.Commit(func(leaf []byte, parent ogTypes.Hash) error {
		account := NewAccountData()
		if _, err := account.UnmarshalMsg(leaf); err != nil {
			return nil
		}
		// log.Tracef("onleaf called with address: %s, root: %v, codehash: %v", account.Address.Hex(), account.Root.ToBytes(), account.CodeHash)
		if account.Root != emptyStateRoot {
			//
			//pd.db.TrieDB().Reference(account.Root, parent)
		}
		codehash := ogTypes.BytesToHash32(account.CodeHash)
		if codehash.Cmp(emptyCodeHash) != 0 {
			//pd.db.TrieDB().Reference(codehash, parent)
		}
		return nil
	}, true)

	//if trie commit fail ,nil root will write to db
	if err != nil {
		log.WithError(err).Warning("commit trie error")
	}
	log.WithField("rootHash", rootHash).Trace("state root set to")
	pd.root = rootHash

	pd.Reset()
	return rootHash, err
}

func (pd *PreloadDB) AppendJournal(entry JournalEntry) {
	pd.journal.append(entry)
}

func (pd *PreloadDB) Suicide(addr ogTypes.Address) bool {
	state := pd.getStateObject(addr)
	if state == nil {
		return false
	}
	pd.AppendJournal(&suicideChange{
		account:     addr,
		prev:        state.suicided,
		prevbalance: state.data.Balances.Copy(),
	})
	state.suicided = true
	state.data.Balances = NewBalanceSet()
	return true
}

func (pd *PreloadDB) HasSuicided(addr ogTypes.Address) bool {
	state := pd.getStateObject(addr)
	if state == nil {
		return false
	}
	return state.suicided
}

// Exist reports whether the given account exists in state.
// Notably this should also return true for suicided accounts.
func (pd *PreloadDB) Exist(addr ogTypes.Address) bool {
	if state := pd.getStateObject(addr); state != nil {
		return true
	}
	return false
}

// Empty returns whether the given account is empty. Empty
// is defined according to EIP161 (balance = nonce = code = 0).
func (pd *PreloadDB) Empty(addr ogTypes.Address) bool {
	state := pd.getStateObject(addr)
	if state == nil {
		return true
	}
	if len(state.code) != 0 {
		return false
	}
	if state.data.Nonce != uint64(0) {
		return false
	}
	if state.data.Balances.IsEmpty() {
		return false
	}
	return true
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (pd *PreloadDB) RevertToSnapshot(int) {}

// Snapshot creates a new revision
func (pd *PreloadDB) Snapshot() int {
	return 0
}

func (pd *PreloadDB) AddLog(l *vmtypes.Log)           {}
func (pd *PreloadDB) AddPreimage(ogTypes.Hash, []byte) {}

func (pd *PreloadDB) ForEachStorage(ogTypes.Address, func(ogTypes.Hash, ogTypes.Hash) bool) {}

// for debug.
func (pd *PreloadDB) String() string {
	return ""
}
