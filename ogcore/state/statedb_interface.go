package state

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
)

// StateDB is an OVM database for full state querying.
type StateDBInterface interface {
	CreateAccount(common.Address)

	SubBalance(common.Address, *math.BigInt)
	SubTokenBalance(common.Address, int32, *math.BigInt)
	AddBalance(common.Address, *math.BigInt)
	AddTokenBalance(common.Address, int32, *math.BigInt)
	SetTokenBalance(common.Address, int32, *math.BigInt)
	// Retrieve the balance from the given address or 0 if object not found
	GetBalance(common.Address) *math.BigInt
	GetTokenBalance(common.Address, int32) *math.BigInt

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	// AddRefund adds gas to the refund counter
	AddRefund(uint64)
	// SubRefund removes gas from the refund counter.
	// This method will panic if the refund counter goes below zero
	SubRefund(uint64)
	// GetRefund returns the current value of the refund counter.
	GetRefund() uint64

	GetCommittedState(common.Address, common.Hash) common.Hash
	// GetState retrieves a value from the given account's storage trie.
	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	AppendJournal(JournalEntry)

	// Suicide marks the given account as suicided.
	// This clears the account balance.
	//
	// The account's state object is still available until the state is committed,
	// getStateObject will return a non-nil account after Suicide.
	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// IsAddressExists reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(common.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(common.Address) bool

	// RevertToSnapshot reverts all state changes made since the given revision.
	RevertToSnapshot(int)
	// Snapshot creates a new revision
	Snapshot() int

	//AddLog(*Log)
	AddPreimage(common.Hash, []byte)

	ForEachStorage(common.Address, func(common.Hash, common.Hash) bool)
	// for debug.
	String() string
}
