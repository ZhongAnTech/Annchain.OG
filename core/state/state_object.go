// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package state

import (
	"bytes"
	"fmt"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	log "github.com/sirupsen/logrus"
	"github.com/tinylib/msgp/msgp"
)

//go:generate msgp

//msgp:tuple AccountData
type AccountData struct {
	Address  common.Address
	Balances BalanceSet
	Nonce    uint64
	Root     common.Hash
	CodeHash []byte
}

func NewAccountData() AccountData {
	return AccountData{
		Address:  common.Address{},
		Balances: NewBalanceSet(),
		Nonce:    0,
		Root:     common.Hash{},
		CodeHash: []byte{},
	}
}

type StateObject struct {
	address     common.Address
	addressHash common.Hash
	data        AccountData

	dbErr error

	code      []byte
	dirtycode bool
	suicided  bool // TODO suicided is useless now.

	committedStorage map[common.Hash]common.Hash
	dirtyStorage     map[common.Hash]common.Hash

	trie Trie
	db   *StateDB
}

func NewStateObject(addr common.Address, db *StateDB) *StateObject {
	a := AccountData{}
	a.Address = addr
	a.Balances = NewBalanceSet()
	a.Nonce = 0
	a.CodeHash = emptyCodeHash.ToBytes()
	a.Root = emptyStateRoot

	s := &StateObject{}
	s.address = addr
	s.addressHash = crypto.Keccak256Hash(addr.ToBytes())
	s.committedStorage = make(map[common.Hash]common.Hash)
	s.dirtyStorage = make(map[common.Hash]common.Hash)
	s.data = a
	s.db = db
	return s
}

func (s *StateObject) GetBalance(tokenID int32) *math.BigInt {
	return s.data.Balances[tokenID]
}

func (s*StateObject)GetAllBalance() BalanceSet{
	return s.data.Balances
}

func (s *StateObject) AddBalance(tokenID int32, increment *math.BigInt) {
	// check if increment is zero
	if increment.Sign() == 0 {
		return
	}
	if s.data.Balances[tokenID] != nil {
		s.SetBalance(tokenID, s.data.Balances[tokenID].Add(increment))
	}
}

func (s *StateObject) SubBalance(tokenID int32, decrement *math.BigInt) {
	// check if decrement is zero
	if decrement.Sign() == 0 {
		return
	}
	if s.data.Balances[tokenID] != nil {
		s.SetBalance(tokenID, s.data.Balances[tokenID].Sub(decrement))
	}
}

func (s *StateObject) SetBalance(tokenID int32, balance *math.BigInt) {
	s.db.journal.append(&balanceChange{
		account: &s.address,
		tokenID: tokenID,
		prev:    s.data.Balances[tokenID],
	})
	s.data.Balances[tokenID] = balance
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

func (s *StateObject) GetState(db Database, key common.Hash) common.Hash {
	value, ok := s.dirtyStorage[key]
	if ok {
		return value
	}
	return s.GetCommittedState(db, key)
}

func (s *StateObject) GetCommittedState(db Database, key common.Hash) common.Hash {
	value, ok := s.committedStorage[key]
	if ok {
		return value
	}
	// load state from trie db.
	b, err := s.openTrie(db).TryGet(key.ToBytes())
	if err != nil {
		log.Errorf("get from trie db error: %v, key: %x", err, key.ToBytes())
		s.setError(err)
	} else {
		s.committedStorage[key] = value
	}
	value = common.BytesToHash(b)
	return value
}

func (s *StateObject) SetState(db Database, key, value common.Hash) {
	s.db.journal.append(&storageChange{
		account:  &s.address,
		key:      key,
		prevalue: s.GetState(db, key),
	})
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

func (s *StateObject) SetCode(codehash common.Hash, code []byte) {
	s.db.journal.append(&codeChange{
		account:  &s.address,
		prevcode: s.code,
		prevhash: s.data.CodeHash,
	})
	s.code = code
	s.data.CodeHash = codehash.ToBytes()
	s.dirtycode = true
}

func (s *StateObject) GetCodeHash() common.Hash {
	return common.BytesToHash(s.data.CodeHash)
}

func (s *StateObject) GetCodeSize(db Database) (int, error) {
	if s.code != nil {
		return len(s.code), nil
	}
	return db.ContractCodeSize(s.addressHash, common.BytesToHash(s.data.CodeHash))
}

func (s *StateObject) openTrie(db Database) Trie {
	if s.trie != nil {
		return s.trie
	}
	t, err := db.OpenStorageTrie(s.addressHash, s.data.Root)
	if err != nil {
		t, _ = db.OpenStorageTrie(s.addressHash, common.BytesToHash([]byte{}))
	}
	s.trie = t
	return s.trie
}

func (s *StateObject) updateTrie(db Database) {
	var err error
	t := s.openTrie(db)
	for key, value := range s.dirtyStorage {
		if len(value.ToBytes()) == 0 {
			err = t.TryDelete(key.ToBytes())
			if err != nil {
				s.setError(err)
				continue
			}
			delete(s.committedStorage, key)
			continue
		}
		log.Tracef("Panic debug, StateObject updateTrie, key: %x, value: %x", key.ToBytes(), value.ToBytes())
		err = t.TryUpdate(key.ToBytes(), value.ToBytes())
		if err != nil {
			s.setError(err)
		}
		s.committedStorage[key] = value
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

// Uncache clears dirtyStorage and committedStorage. This is aimed
// to check if state is committed into db.
//
// Note that this function is for test debug only, should not
// be called by other functions.
func (s *StateObject) Uncache() {
	s.committedStorage = make(map[common.Hash]common.Hash)
	s.dirtyStorage = make(map[common.Hash]common.Hash)
}

/*
	Encode part
*/

func (s *StateObject) Encode() ([]byte, error) {
	return s.data.MarshalMsg(nil)
}

func (s *StateObject) Decode(b []byte, db *StateDB) error {
	var a AccountData
	_, err := a.UnmarshalMsg(b)

	s.data = a
	s.address = a.Address
	s.addressHash = crypto.Keccak256Hash(a.Address.ToBytes())
	s.committedStorage = make(map[common.Hash]common.Hash)
	s.dirtyStorage = make(map[common.Hash]common.Hash)
	s.db = db
	return err
}

// BalanceSet
type BalanceSet map[int32]*math.BigInt

func NewBalanceSet() BalanceSet {
	return BalanceSet(make(map[int32]*math.BigInt))
}

func (b *BalanceSet) PreAdd(tokenID int32, increment *math.BigInt) *math.BigInt {
	bi := (*b)[tokenID]
	if bi == nil {
		return math.NewBigInt(0)
	}
	return bi.Add(increment)
}

func (b *BalanceSet) PreSub(tokenID int32, decrement *math.BigInt) *math.BigInt {
	bi := (*b)[tokenID]
	if bi == nil {
		return math.NewBigInt(0)
	}
	return bi.Sub(decrement)
}

func (b *BalanceSet) Copy() BalanceSet {
	bs := NewBalanceSet()
	for k, v := range *b {
		bs[k] = v
	}
	return bs
}

func (b *BalanceSet) IsEmpty() bool {
	for _, v := range *b {
		if v.GetInt64() != int64(0) {
			return false
		}
	}
	return true
}

// MarshalMsg - For every [key, value] pair, marshal it in [size (int32) + key (int32) + bigint.bytes]
func (b *BalanceSet) MarshalMsg(bts []byte) (o []byte, err error) {

	msgpSize := b.Msgsize()
	o = msgp.Require(bts, msgpSize)

	// add total size
	o = append(o, common.ByteInt32(int32(0))...)

	for k, v := range *b {
		o = append(o, common.ByteInt32(k)...)

		o, err = v.MarshalMsg(o)
		//fmt.Println(fmt.Sprintf("cur o: %x", o))
		if err != nil {
			return
		}
	}
	size := len(o) - len(bts)
	common.SetInt32(o, len(bts), int32(size))

	return o, nil
}

func (b *BalanceSet) UnmarshalMsg(bts []byte) (o []byte, err error) {
	size := common.GetInt32(bts, 0)
	bsBytes := bts[4:size]

	for len(bsBytes) > 0 {
		key := common.GetInt32(bsBytes, 0)
		bsBytes = bsBytes[4:]

		value := math.BigInt{}
		bsBytes, err = value.UnmarshalMsg(bsBytes)
		if err != nil {
			return bsBytes, err
		}
		(*b)[key] = &value
	}

	return bts[size:], nil
}

// Msgsize - BalanceSet size = size (4 bytes for int32) + every key pair size
func (b *BalanceSet) Msgsize() int {
	l := 4
	for _, v := range *b {
		l += 4 + v.Msgsize()
	}
	return l
}

/*
	components
*/

func (s *StateObject) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}
