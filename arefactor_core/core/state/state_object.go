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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/arefactor/utils/marshaller"
	"math/big"

	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	//"github.com/annchain/OG/common/crypto"
	crypto "github.com/annchain/OG/arefactor/ogcrypto"
	"github.com/annchain/OG/common/math"
	log "github.com/sirupsen/logrus"
)

//go:generate msgp

//msgp:tuple AccountData
type AccountData struct {
	Address  ogTypes.Address
	Balances BalanceSet
	Nonce    uint64
	Root     ogTypes.Hash
	CodeHash []byte
}

func NewAccountData() AccountData {
	return AccountData{
		Address:  &ogTypes.Address20{},
		Balances: NewBalanceSet(),
		Nonce:    0,
		Root:     &ogTypes.Hash32{},
		CodeHash: []byte{},
	}
}

type StateObject struct {
	address     ogTypes.Address
	addressHash ogTypes.Hash
	data        AccountData

	dbErr error

	code      []byte
	dirtycode bool
	suicided  bool // TODO suicided is useless now.

	committedStorage map[ogTypes.HashKey]ogTypes.Hash
	dirtyStorage     map[ogTypes.HashKey]ogTypes.Hash

	trie Trie
	db   StateDBInterface
}

func NewStateObject(addr ogTypes.Address, db StateDBInterface) *StateObject {
	a := AccountData{}
	a.Address = addr
	a.Balances = NewBalanceSet()
	a.Nonce = 0
	a.CodeHash = emptyCodeHash.Bytes()
	a.Root = emptyStateRoot

	s := &StateObject{}
	s.address = addr
	s.addressHash = ogTypes.BytesToHash32(crypto.Keccak256Hash(addr.Bytes()))
	s.committedStorage = make(map[ogTypes.HashKey]ogTypes.Hash)
	s.dirtyStorage = make(map[ogTypes.HashKey]ogTypes.Hash)
	s.data = a
	s.db = db
	return s
}

func (s *StateObject) Copy(src *StateObject) {
	s.address = src.address
	s.addressHash = src.addressHash
	s.data = src.data
}

func (s *StateObject) GetBalance(tokenID int32) *math.BigInt {
	b := s.data.Balances[tokenID]
	if b == nil {
		return math.NewBigInt(0)
	}
	return b
}

func (s *StateObject) GetAllBalance() BalanceSet {
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
	s.db.AppendJournal(&balanceChange{
		account: s.address,
		tokenID: tokenID,
		prev:    s.data.Balances[tokenID],
	})
	s.setBalance(tokenID, balance)
}

func (s *StateObject) setBalance(tokenID int32, balance *math.BigInt) {
	s.data.Balances[tokenID] = balance
}

func (s *StateObject) GetNonce() uint64 {
	return s.data.Nonce
}

func (s *StateObject) SetNonce(nonce uint64) {
	s.db.AppendJournal(&nonceChange{
		account: s.address,
		prev:    s.data.Nonce,
	})
	s.setNonce(nonce)
}

func (s *StateObject) setNonce(nonce uint64) {
	s.data.Nonce = nonce
}

func (s *StateObject) GetState(db Database, key ogTypes.Hash) ogTypes.Hash {
	value, ok := s.dirtyStorage[key.HashKey()]
	if ok {
		return value
	}
	return s.GetCommittedState(db, key)
}

func (s *StateObject) GetCommittedState(db Database, key ogTypes.Hash) ogTypes.Hash {
	value, ok := s.committedStorage[key.HashKey()]
	if ok {
		return value
	}
	// load state from trie db.
	b, err := s.openTrie(db).TryGet(key.Bytes())
	if err != nil {
		log.Errorf("get from trie db error: %v, key: %x", err, key.Bytes())
		s.setError(err)
	}

	value = ogTypes.BytesToHash32(b)
	s.committedStorage[key.HashKey()] = value

	return value
}

func (s *StateObject) SetState(db Database, key, value ogTypes.Hash) {
	s.db.AppendJournal(&storageChange{
		account:  s.address,
		key:      key,
		prevalue: s.GetState(db, key),
	})
	s.setState(key, value)
}

func (s *StateObject) setState(key, value ogTypes.Hash) {
	s.dirtyStorage[key.HashKey()] = value
}

func (s *StateObject) GetCode(db Database) []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.GetCodeHash().Bytes(), emptyCodeHash.Bytes()) {
		return nil
	}
	code, err := db.ContractCode(s.addressHash, s.GetCodeHash())
	if err != nil {
		s.setError(fmt.Errorf("load code from db error: %v", err))
	}
	s.code = code
	return s.code
}

func (s *StateObject) SetCode(codehash ogTypes.Hash, code []byte) {
	s.db.AppendJournal(&codeChange{
		account:  s.address,
		prevcode: s.code,
		prevhash: s.data.CodeHash,
	})
	s.setCode(codehash, code)
}

func (s *StateObject) setCode(codehash ogTypes.Hash, code []byte) {
	s.code = code
	s.data.CodeHash = codehash.Bytes()
	s.dirtycode = true
}

func (s *StateObject) GetCodeHash() ogTypes.Hash {
	return ogTypes.BytesToHash32(s.data.CodeHash)
}

func (s *StateObject) GetCodeSize(db Database) (int, error) {
	if s.code != nil {
		return len(s.code), nil
	}
	return db.ContractCodeSize(s.addressHash, ogTypes.BytesToHash32(s.data.CodeHash))
}

func (s *StateObject) openTrie(db Database) Trie {
	if s.trie != nil {
		return s.trie
	}
	t, err := db.OpenStorageTrie(s.addressHash, s.data.Root)
	if err != nil {
		t, _ = db.OpenStorageTrie(s.addressHash, ogTypes.BytesToHash32([]byte{}))
	}
	s.trie = t
	return s.trie
}

func (s *StateObject) updateTrie(db Database) {
	var err error
	t := s.openTrie(db)
	for key, value := range s.dirtyStorage {
		if len(value.Bytes()) == 0 {
			err = t.TryDelete(key.Bytes())
			if err != nil {
				s.setError(err)
				continue
			}
			delete(s.committedStorage, key)
			continue
		}
		//log.Tracef("Panic debug, StateObject updateTrie, key: %x, value: %x", key.ToBytes(), value.ToBytes())
		err = t.TryUpdate(key.Bytes(), value.Bytes())
		if err != nil {
			s.setError(err)
		}
		s.committedStorage[key] = value
		delete(s.dirtyStorage, key)
	}
}

func (s *StateObject) CommitStorage(db Database, preCommit bool) error {
	s.updateTrie(db)
	if s.dbErr != nil {
		return s.dbErr
	}
	root, err := s.trie.Commit(nil, preCommit)
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
	s.committedStorage = make(map[ogTypes.HashKey]ogTypes.Hash)
	s.dirtyStorage = make(map[ogTypes.HashKey]ogTypes.Hash)
}

/*
	Encode part
*/

func (s *StateObject) Map() map[string]interface{} {
	stobjMap := map[string]interface{}{}

	stobjMap["code"] = hex.EncodeToString(s.code)
	stobjMap["committed"] = s.committedStorage
	stobjMap["dirty"] = s.dirtyStorage
	stobjMap["data"] = s.data

	return stobjMap
}

func (s *StateObject) String() string {
	stobjMap := s.Map()
	b, _ := json.Marshal(stobjMap)
	return string(b)
}

/**
marshalling part
 */

func (s *StateObject) Encode() ([]byte, error) {
	return s.data.MarshalMsg()
}

func (s *StateObject) Decode(b []byte, db *StateDB) error {
	a := NewAccountData()
	_, err := a.UnmarshalMsg(b)

	s.data = a
	s.address = a.Address
	s.addressHash = ogTypes.BytesToHash32(crypto.Keccak256Hash(a.Address.Bytes()))
	s.committedStorage = make(map[ogTypes.HashKey]ogTypes.Hash)
	s.dirtyStorage = make(map[ogTypes.HashKey]ogTypes.Hash)
	s.db = db
	return err
}

func (a *AccountData) MarshalMsg() (b []byte, err error) {
	b = make([]byte, marshaller.HeaderSize)

	// (Address) Address
	addrB, err := a.Address.MarshalMsg()
	if err != nil {
		return b, err
	}
	b = append(b, addrB...)

	// (BalanceSet) Balances
	blcB, err := a.Balances.MarshalMsg()
	if err != nil {
		return b, err
	}
	b = append(b, blcB...)

	// (uint64) Nonce
	b = marshaller.AppendUint64(b, a.Nonce)

	// (Hash) Root
	hashB, err := a.Root.MarshalMsg()
	if err != nil {
		return b, err
	}
	b = append(b, hashB...)

	// ([]byte) CodeHash
	b = marshaller.AppendBytes(b, a.CodeHash)

	// Fill header
	b = marshaller.FillHeaderData(b)

	return b, nil
}

func (a *AccountData) UnmarshalMsg(b []byte) ([]byte, error) {
	b, _, err := marshaller.DecodeHeader(b)
	if err != nil {
		return b, err
	}

	// Address
	a.Address, b, err = ogTypes.UnmarshalAddress(b)
	if err != nil {
		return b, err
	}

	// BalanceSet
	bs := &BalanceSet{}
	b, err = bs.UnmarshalMsg(b)
	if err != nil {
		return b, err
	}
	a.Balances = *bs

	// Nonce
	a.Nonce, b, err = marshaller.ReadUint64(b)
	if err != nil {
		return b, err
	}

	// Hash Root
	a.Root, b, err = ogTypes.UnmarshalHash(b)
	if err != nil {
		return b, err
	}

	// ([]byte) CodeHash
	a.CodeHash, b, err = marshaller.ReadBytes(b)
	if err != nil {
		return b, err
	}

	return b, nil
}

func (a *AccountData) MsgSize() int {
	// Address + Balances + Root + CodeHash
	size := marshaller.CalIMarshallerSize(a.Address.MsgSize()) +
		marshaller.CalIMarshallerSize(a.Balances.MsgSize()) +
		marshaller.CalIMarshallerSize(a.Root.MsgSize()) +
		marshaller.CalIMarshallerSize(len(a.CodeHash))
	// Nonce
	size += marshaller.Uint64Size

	return size
}

/**
BalanceSet
 */

type BalanceSet map[int32]*math.BigInt

func NewBalanceSet() BalanceSet {
	return make(map[int32]*math.BigInt)
}

func (bs *BalanceSet) PreAdd(tokenID int32, increment *math.BigInt) *math.BigInt {
	bi := (*bs)[tokenID]
	if bi == nil {
		bi = math.NewBigInt(0)
	}
	return bi.Add(increment)
}

func (bs *BalanceSet) PreSub(tokenID int32, decrement *math.BigInt) *math.BigInt {
	bi := (*bs)[tokenID]
	if bi == nil {
		return math.NewBigInt(0)
	}
	return bi.Sub(decrement)
}

func (bs *BalanceSet) Copy() BalanceSet {
	b := NewBalanceSet()
	for k, v := range *bs {
		b[k] = v
	}
	return b
}

func (bs *BalanceSet) IsEmpty() bool {
	for _, v := range *bs {
		if v.GetInt64() != int64(0) {
			return false
		}
	}
	return true
}

/**
marshaller part
 */

func (bs *BalanceSet) MarshalMsg() (b []byte, err error) {
	b = make([]byte, marshaller.HeaderSize)

	for k, v := range *bs {
		// marshal key
		b = marshaller.AppendInt32(b, k)
		// marshal value
		b = marshaller.AppendBigInt(b, v.Value)
	}

	b = marshaller.FillHeaderDataNum(b, len(*bs))

	return b, nil
}

func (bs *BalanceSet) UnmarshalMsg(bts []byte) (b []byte, err error) {

	b, mapSize, err := marshaller.DecodeHeader(bts)
	if err != nil {
		return nil, err
	}

	bsRaw := BalanceSet(make(map[int32]*math.BigInt))
	var k int32
	var v *big.Int
	for i := 0; i < mapSize; i++ {
		k, b, err = marshaller.ReadInt32(b)
		if err != nil {
			return nil, err
		}
		v, b, err = marshaller.ReadBigInt(b)
		if err != nil {
			return nil, err
		}

		bsRaw[k] = math.NewBigIntFromBigInt(v)
	}

	bs = &bsRaw

	return b, nil
}

func (bs *BalanceSet) MsgSize() int {
	sz := 0
	for _, v := range *bs {
		sz += marshaller.Int32Size + marshaller.CalBigIntSize(v.Value)
	}

	return sz
}

/*
	components
*/

func (s *StateObject) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}
