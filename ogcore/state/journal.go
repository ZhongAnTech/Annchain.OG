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
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
)

const TokenNotDirtied int32 = -1

// journalEntry is a modification entry in the state change journal that can be
// reverted on demand.
type JournalEntry interface {
	// revert undoes the changes introduced by this journal entry.
	Revert(db *StateDB)

	// dirtied returns the address modified by this journal entry.
	Dirtied() *common.Address

	// TokenDirtied returns the Token ID modified by this journal entry.
	TokenDirtied() int32
}

// journal contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in case of an execution
// exception or revertal request.
type journal struct {
	entries      []JournalEntry         // Current changes tracked by the journal
	dirties      map[common.Address]int // Dirty accounts and the number of changes
	tokenDirties map[int32]int          // Dirty tokens and the number of changes.
}

// newJournal create a new initialized journal.
func newJournal() *journal {
	return &journal{
		dirties:      make(map[common.Address]int),
		tokenDirties: make(map[int32]int),
	}
}

// append inserts a new modification entry to the end of the change journal.
func (j *journal) append(entry JournalEntry) {
	j.entries = append(j.entries, entry)
	if addr := entry.Dirtied(); addr != nil {
		j.dirties[*addr]++
	}
	if tokenID := entry.TokenDirtied(); tokenID > TokenNotDirtied {
		j.tokenDirties[tokenID]++
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
		if tokenID := j.entries[i].TokenDirtied(); tokenID > TokenNotDirtied {
			if j.tokenDirties[tokenID]--; j.tokenDirties[tokenID] == 0 {
				delete(j.tokenDirties, tokenID)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

// dirty explicitly sets an address to dirty, even if the change entries would
// otherwise suggest it as clean. This method is an ugly hack to handle the RIPEMD
// precompile consensus exception.
// func (j *journal) dirty(addr common.Address) {
//	j.dirties[addr]++
// }

// func (j *journal) tokenDirty(tokenID int32) {
// 	j.tokenDirties[tokenID]++
// }

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
		account *common.Address
		prev    *StateObject
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
		key, prevalue types.Hash
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
		txhash types.Hash
	}
	addPreimageChange struct {
		hash types.Hash
	}
	touchChange struct {
		account   *common.Address
		prev      bool
		prevDirty bool
	}

	// Changes to token
	createTokenChange struct {
		prevLatestTokenID int32
		tokenID           int32
	}
	resetTokenChange struct {
		tokenID int32
		prev    *TokenObject
	}
	reIssueChange struct {
		tokenID int32
	}
	destroyChange struct {
		tokenID       int32
		prevDestroyed bool
	}
)

func (ch createObjectChange) Revert(s *StateDB) {
	delete(s.states, *ch.account)
	delete(s.dirtyset, *ch.account)
}

func (ch createObjectChange) Dirtied() *common.Address {
	return ch.account
}

func (ch createObjectChange) TokenDirtied() int32 {
	return TokenNotDirtied
}

func (ch resetObjectChange) Revert(s *StateDB) {
	if ch.prev == nil {
		delete(s.states, *ch.account)
	} else {
		s.states[ch.prev.address] = ch.prev
	}
}

func (ch resetObjectChange) Dirtied() *common.Address {
	return &ch.prev.address
}

func (ch resetObjectChange) TokenDirtied() int32 {
	return TokenNotDirtied
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

func (ch suicideChange) TokenDirtied() int32 {
	return TokenNotDirtied
}

func (ch touchChange) Revert(s *StateDB) {
}

func (ch touchChange) Dirtied() *common.Address {
	return ch.account
}

func (ch touchChange) TokenDirtied() int32 {
	return TokenNotDirtied
}

func (ch balanceChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.setBalance(ch.tokenID, ch.prev)
	}
}

func (ch balanceChange) Dirtied() *common.Address {
	return ch.account
}

func (ch balanceChange) TokenDirtied() int32 {
	return TokenNotDirtied
}

func (ch nonceChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.setNonce(ch.prev)
	}
}

func (ch nonceChange) Dirtied() *common.Address {
	return ch.account
}

func (ch nonceChange) TokenDirtied() int32 {
	return TokenNotDirtied
}

func (ch codeChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.setCode(types.BytesToHash(ch.prevhash), ch.prevcode)
	}
}

func (ch codeChange) Dirtied() *common.Address {
	return ch.account
}

func (ch codeChange) TokenDirtied() int32 {
	return TokenNotDirtied
}

func (ch storageChange) Revert(s *StateDB) {
	stobj := s.getStateObject(*ch.account)
	if stobj != nil {
		stobj.setState(ch.key, ch.prevalue)
	}
}

func (ch storageChange) Dirtied() *common.Address {
	return ch.account
}

func (ch storageChange) TokenDirtied() int32 {
	return TokenNotDirtied
}

func (ch refundChange) Revert(s *StateDB) {
	s.refund = ch.prev
}

func (ch refundChange) Dirtied() *common.Address {
	return nil
}

func (ch refundChange) TokenDirtied() int32 {
	return TokenNotDirtied
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

func (ch addLogChange) TokenDirtied() int32 {
	return TokenNotDirtied
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

func (ch addPreimageChange) TokenDirtied() int32 {
	return TokenNotDirtied
}

func (ch createTokenChange) Revert(s *StateDB) {
	s.latestTokenID = ch.prevLatestTokenID
	delete(s.tokens, ch.tokenID)
	delete(s.dirtyTokens, ch.tokenID)
}

func (ch createTokenChange) Dirtied() *common.Address {
	return nil
}

func (ch createTokenChange) TokenDirtied() int32 {
	return ch.tokenID
}

func (ch resetTokenChange) Revert(s *StateDB) {
	if ch.prev == nil {
		delete(s.tokens, ch.tokenID)
	} else {
		s.tokens[ch.prev.TokenID] = ch.prev
	}
}

func (ch resetTokenChange) Dirtied() *common.Address {
	return nil
}

func (ch resetTokenChange) TokenDirtied() int32 {
	return ch.prev.TokenID
}

func (ch reIssueChange) Revert(s *StateDB) {
	tkObj := s.getTokenObject(ch.tokenID)
	if tkObj != nil {
		tkObj.Issues = tkObj.Issues[:len(tkObj.Issues)-1]
	}
}

func (ch reIssueChange) Dirtied() *common.Address {
	return nil
}

func (ch reIssueChange) TokenDirtied() int32 {
	return ch.tokenID
}

func (ch destroyChange) Revert(s *StateDB) {
	tkObj := s.getTokenObject(ch.tokenID)
	if tkObj != nil {
		tkObj.Destroyed = ch.prevDestroyed
	}
}

func (ch destroyChange) Dirtied() *common.Address {
	return nil
}

func (ch destroyChange) TokenDirtied() int32 {
	return ch.tokenID
}
