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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
)

// journalEntry is a modification entry in the state change journal that can be
// reverted on demand.
type JournalEntry interface {
	// revert undoes the changes introduced by this journal entry.
	Revert(db *StateDB)

	// dirtied returns the Ethereum address modified by this journal entry.
	Dirtied() *common.Address
}

// journal contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in case of an execution
// exception or revertal request.
type journal struct {
	entries []JournalEntry        // Current changes tracked by the journal
	dirties map[common.Address]int // Dirty accounts and the number of changes
}

// newJournal create a new initialized journal.
func newJournal() *journal {
	return &journal{
		dirties: make(map[common.Address]int),
	}
}

// append inserts a new modification entry to the end of the change journal.
func (j *journal) append(entry JournalEntry) {
	j.entries = append(j.entries, entry)
	if addr := entry.Dirtied(); addr != nil {
		j.dirties[*addr]++
	}
}

// revert undoes a batch of journalled modifications along with any reverted
// dirty handling too.
func (j *journal) revert(statedb *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// Undo the changes made by the operation
		j.entries[i].Revert(statedb)

		// Drop any dirty tracking induced by the change
		if addr := j.entries[i].Dirtied(); addr != nil {
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
func (j *journal) dirty(addr common.Address) {
	j.dirties[addr]++
}

// length returns the current number of entries in the journal.
func (j *journal) length() int {
	return len(j.entries)
}

type (
	// Changes to the account trie.
	createObjectChange struct {
		account *common.Address
	}
	resetObjectChange struct {
		prev *StateObject
	}
	suicideChange struct {
		account     *common.Address
		prev        bool // whether account had already suicided
		prevbalance BalanceSet
	}

	// Changes to individual accounts.
	balanceChange struct {
		account *common.Address
		tokenID int32
		prev    *math.BigInt
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	storageChange struct {
		account       *common.Address
		key, prevalue common.Hash
	}
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
	}

	// Changes to other state values.
	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash common.Hash
	}
	addPreimageChange struct {
		hash common.Hash
	}
	touchChange struct {
		account   *common.Address
		prev      bool
		prevDirty bool
	}
)

func (ch createObjectChange) Revert(s *StateDB) {
	delete(s.states, *ch.account)
	delete(s.dirtyset, *ch.account)
}

func (ch createObjectChange) Dirtied() *common.Address {
	return ch.account
}

func (ch resetObjectChange) Revert(s *StateDB) {
	s.setStateObject(ch.prev.address, ch.prev)
}

func (ch resetObjectChange) Dirtied() *common.Address {
	return nil
}

func (ch suicideChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.suicided = ch.prev
		for k, v := range ch.prevbalance {
			stobj.SetBalance(k, v)
		}
	}
}

func (ch suicideChange) Dirtied() *common.Address {
	return ch.account
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) Revert(s *StateDB) {
}

func (ch touchChange) Dirtied() *common.Address {
	return ch.account
}

func (ch balanceChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.SetBalance(ch.tokenID, ch.prev)
	}
}

func (ch balanceChange) Dirtied() *common.Address {
	return ch.account
}

func (ch nonceChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.SetNonce(ch.prev)
	}
}

func (ch nonceChange) Dirtied() *common.Address {
	return ch.account
}

func (ch codeChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.SetCode(common.BytesToHash(ch.prevhash), ch.prevcode)
	}
}

func (ch codeChange) Dirtied() *common.Address {
	return ch.account
}

func (ch storageChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.SetState(s.db, ch.key, ch.prevalue)
	}
}

func (ch storageChange) Dirtied() *common.Address {
	return ch.account
}

func (ch refundChange) Revert(s *StateDB) {
	s.refund = ch.prev
}

func (ch refundChange) Dirtied() *common.Address {
	return nil
}

func (ch addLogChange) Revert(s *StateDB) {
	// TODO
	// Log not implemented yet. To avoid the error, comment
	// this function.

	// logs := s.logs[ch.txhash]
	// if len(logs) == 1 {
	// 	delete(s.logs, ch.txhash)
	// } else {
	// 	s.logs[ch.txhash] = logs[:len(logs)-1]
	// }
	// s.logSize--
}

func (ch addLogChange) Dirtied() *common.Address {
	return nil
}

func (ch addPreimageChange) Revert(s *StateDB) {
	// TODO
	// preimage not implemented yet, comment temporarily this function
	// to avoid compile error.

	// delete(s.preimages, ch.hash)
}

func (ch addPreimageChange) Dirtied() *common.Address {
	return nil
}
