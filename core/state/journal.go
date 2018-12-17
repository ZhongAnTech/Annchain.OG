// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
)

// journalEntry is a modification entry in the state change journal that can be
// reverted on demand.
type journalEntry interface {
	// revert undoes the changes introduced by this journal entry.
	revert(*StateDB)

	// dirtied returns the Ethereum address modified by this journal entry.
	dirtied() *types.Address
}

// journal contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in case of an execution
// exception or revertal request.
type journal struct {
	entries []journalEntry        // Current changes tracked by the journal
	dirties map[types.Address]int // Dirty accounts and the number of changes
}

// newJournal create a new initialized journal.
func newJournal() *journal {
	return &journal{
		dirties: make(map[types.Address]int),
	}
}

// append inserts a new modification entry to the end of the change journal.
func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
	if addr := entry.dirtied(); addr != nil {
		j.dirties[*addr]++
	}
}

// revert undoes a batch of journalled modifications along with any reverted
// dirty handling too.
func (j *journal) revert(statedb *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// Undo the changes made by the operation
		j.entries[i].revert(statedb)

		// Drop any dirty tracking induced by the change
		if addr := j.entries[i].dirtied(); addr != nil {
			if j.dirties[*addr]--; j.dirties[*addr] == 0 {
				delete(j.dirties, *addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

// dirty explicitly sets an address to dirty, even if the change entries would
// otherwise suggest it as clean. This method is an ugly hack to handle the RIPEMD
// precompile consensus exception.
func (j *journal) dirty(addr types.Address) {
	j.dirties[addr]++
}

// length returns the current number of entries in the journal.
func (j *journal) length() int {
	return len(j.entries)
}

type (
	// Changes to the account trie.
	createObjectChange struct {
		account *types.Address
	}
	resetObjectChange struct {
		prev *StateObject
	}
	suicideChange struct {
		account     *types.Address
		prev        bool // whether account had already suicided
		prevbalance *math.BigInt
	}

	// Changes to individual accounts.
	balanceChange struct {
		account *types.Address
		prev    *math.BigInt
	}
	nonceChange struct {
		account *types.Address
		prev    uint64
	}
	storageChange struct {
		account       *types.Address
		key, prevalue types.Hash
	}
	codeChange struct {
		account            *types.Address
		prevcode, prevhash []byte
	}

	// Changes to other state values.
	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash types.Hash
	}
	addPreimageChange struct {
		hash types.Hash
	}
	touchChange struct {
		account   *types.Address
		prev      bool
		prevDirty bool
	}
)

func (ch createObjectChange) revert(s *StateDB) {
	delete(s.states, *ch.account)
	delete(s.dirtyset, *ch.account)
}

func (ch createObjectChange) dirtied() *types.Address {
	return ch.account
}

func (ch resetObjectChange) revert(s *StateDB) {
	s.setStateObject(ch.prev.address, ch.prev)
}

func (ch resetObjectChange) dirtied() *types.Address {
	return nil
}

func (ch suicideChange) revert(s *StateDB) {
	obj, _ := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prev
		obj.SetBalance(ch.prevbalance)
	}
}

func (ch suicideChange) dirtied() *types.Address {
	return ch.account
}

var ripemd = types.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) revert(s *StateDB) {
}

func (ch touchChange) dirtied() *types.Address {
	return ch.account
}

func (ch balanceChange) revert(s *StateDB) {
	stobj, _ := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.SetBalance(ch.prev)
	}
}

func (ch balanceChange) dirtied() *types.Address {
	return ch.account
}

func (ch nonceChange) revert(s *StateDB) {
	stobj, _ := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.SetNonce(ch.prev)
	}
}

func (ch nonceChange) dirtied() *types.Address {
	return ch.account
}

func (ch codeChange) revert(s *StateDB) {
	stobj, _ := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.SetCode(types.BytesToHash(ch.prevhash), ch.prevcode)
	}
}

func (ch codeChange) dirtied() *types.Address {
	return ch.account
}

func (ch storageChange) revert(s *StateDB) {
	stobj, _ := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.SetState(s.db, ch.key, ch.prevalue)
	}
}

func (ch storageChange) dirtied() *types.Address {
	return ch.account
}

func (ch refundChange) revert(s *StateDB) {
	s.refund = ch.prev
}

func (ch refundChange) dirtied() *types.Address {
	return nil
}

func (ch addLogChange) revert(s *StateDB) {
	// TODO
	// Log not implemented yet. To avoid the error, commment
	// this function.

	// logs := s.logs[ch.txhash]
	// if len(logs) == 1 {
	// 	delete(s.logs, ch.txhash)
	// } else {
	// 	s.logs[ch.txhash] = logs[:len(logs)-1]
	// }
	// s.logSize--
}

func (ch addLogChange) dirtied() *types.Address {
	return nil
}

func (ch addPreimageChange) revert(s *StateDB) {
	// TODO
	// preimage not implemented yet, comment temporarily this function
	// to avoid compile error.

	// delete(s.preimages, ch.hash)
}

func (ch addPreimageChange) dirtied() *types.Address {
	return nil
}
