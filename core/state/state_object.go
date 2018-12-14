package state

import (
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

	code []byte
	trie Trie
	db   *StateDB

	cacheStorage map[types.Hash]types.Hash
	dirtyStorage map[types.Hash]types.Hash
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
	s.data.Balance = balance
}

func (s *StateObject) GetNonce() uint64 {
	return s.data.Nonce
}

func (s *StateObject) SetNonce(nonce uint64) {
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
