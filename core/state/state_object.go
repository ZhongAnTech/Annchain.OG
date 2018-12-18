package state

import (
	"bytes"
	"fmt"

	"github.com/annchain/OG/common/math"
	// "github.com/annchain/OG/trie"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
)

//go:generate msgp

//msgp:tuple Account
type Account struct {
	Address  types.Address
	Balance  *math.BigInt
	Nonce    uint64
	Root     types.Hash
	CodeHash []byte
}

type StateObject struct {
	address     types.Address
	addressHash types.Hash
	data        Account

	dbErr error

	code      []byte
	dirtycode bool
	suicided  bool // TODO suicided is useless now.

	cacheStorage map[types.Hash]types.Hash
	dirtyStorage map[types.Hash]types.Hash

	trie Trie
	db   *StateDB
}

func NewStateObject(addr types.Address) *StateObject {
	s := &StateObject{}
	s.address = addr
	s.addressHash = crypto.Keccak256Hash(addr.ToBytes())
	s.cacheStorage = make(map[types.Hash]types.Hash)
	s.dirtyStorage = make(map[types.Hash]types.Hash)

	a := Account{}
	a.Address = addr
	a.Balance = math.NewBigInt(0)
	a.Nonce = 0
	a.CodeHash = emptyCodeHash.ToBytes()

	s.data = a
	return s
}

func (s *StateObject) GetBalance() *math.BigInt {
	return s.data.Balance
}

func (s *StateObject) AddBalance(increment *math.BigInt) {
	// check if increment is zero
	if increment.Sign() == 0 {
		return
	}
	s.SetBalance(s.data.Balance.Add(increment))
}

func (s *StateObject) SubBalance(decrement *math.BigInt) {
	// check if decrement is zero
	if decrement.Sign() == 0 {
		return
	}
	s.SetBalance(s.data.Balance.Sub(decrement))
}

func (s *StateObject) SetBalance(balance *math.BigInt) {
	s.db.journal.append(&balanceChange{
		account: &s.address,
		prev:    s.data.Balance,
	})
	s.data.Balance = balance
}

func (s *StateObject) GetNonce() uint64 {
	return s.data.Nonce
}

func (s *StateObject) SetNonce(nonce uint64) {
	s.db.journal.append(&nonceChange{
		account: &s.address,
		prev:    s.data.Nonce,
	})
	s.data.Nonce = nonce
}

func (s *StateObject) GetState(db Database, key types.Hash) types.Hash {
	value, ok := s.cacheStorage[key]
	if ok {
		return value
	}
	// load state from trie db.
	b, err := s.openTrie(db).TryGet(key.ToBytes())
	if err != nil {
		s.dbErr = err
	}
	// TODO:
	// rlp.Split  ?
	value = types.BytesToHash(b)
	s.cacheStorage[key] = value
	return value
}

func (s *StateObject) SetState(db Database, key, value types.Hash) {
	s.db.journal.append(&storageChange{
		account:  &s.address,
		key:      key,
		prevalue: s.GetState(db, key),
	})
	s.cacheStorage[key] = value
	s.dirtyStorage[key] = value
}

func (s *StateObject) GetCode(db Database) []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.GetCodeHash().ToBytes(), emptyCodeHash.ToBytes()) {
		return nil
	}
	code, err := db.ContractCode(s.addressHash, s.GetCodeHash())
	if err != nil {
		s.setError(fmt.Errorf("load code from db error: %v", err))
	}
	s.code = code
	return s.code
}

func (s *StateObject) SetCode(codehash types.Hash, code []byte) {
	s.db.journal.append(&codeChange{
		account:  &s.address,
		prevcode: s.code,
		prevhash: s.data.CodeHash,
	})
	s.code = code
	s.data.CodeHash = codehash.ToBytes()
	s.dirtycode = true
}

func (s *StateObject) GetCodeHash() types.Hash {
	return types.BytesToHash(s.data.CodeHash)
}

func (s *StateObject) GetCodeSize(db Database) (int, error) {
	if s.code != nil {
		return len(s.code), nil
	}
	return db.ContractCodeSize(s.addressHash, types.BytesToHash(s.data.CodeHash))
}

func (s *StateObject) openTrie(db Database) Trie {
	if s.trie != nil {
		return s.trie
	}
	t, err := db.OpenStorageTrie(s.addressHash, s.data.Root)
	if err != nil {
		t, _ = db.OpenStorageTrie(s.addressHash, types.BytesToHash([]byte{}))
	}
	s.trie = t
	return s.trie
}

func (s *StateObject) updateTrie(db Database) {
	t := s.openTrie(db)
	for key, value := range s.dirtyStorage {
		if len(value.ToBytes()) == 0 {
			s.setError(t.TryDelete(key.ToBytes()))
		}
		s.setError(t.TryUpdate(key.ToBytes(), value.ToBytes()))
		delete(s.dirtyStorage, key)
	}
}

func (s *StateObject) CommitStorage(db Database) error {
	s.updateTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.trie.Commit(nil)
	if err != nil {
		return err
	}
	s.data.Root = root
	return nil
}

/*
	Encode part
*/

func (s *StateObject) Encode() ([]byte, error) {
	return s.data.MarshalMsg(nil)
}

func (s *StateObject) Decode(b []byte) error {
	var a Account
	_, err := a.UnmarshalMsg(b)

	s.data = a
	s.address = a.Address
	s.addressHash = crypto.Keccak256Hash(a.Address.ToBytes())
	s.cacheStorage = make(map[types.Hash]types.Hash)
	s.dirtyStorage = make(map[types.Hash]types.Hash)
	return err
}

/*
	components
*/

func (s *StateObject) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}
