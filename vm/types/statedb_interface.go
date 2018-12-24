package types

import (
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
)

// StateDB is an OVM database for full state querying.
type StateDB interface {
	CreateAccount(types.Address)

	SubBalance(types.Address, *math.BigInt)
	AddBalance(types.Address, *math.BigInt)
	// Retrieve the balance from the given address or 0 if object not found
	GetBalance(types.Address) *math.BigInt

	GetNonce(types.Address) uint64
	SetNonce(types.Address, uint64)

	GetCodeHash(types.Address) types.Hash
	GetCode(types.Address) []byte
	SetCode(types.Address, []byte)
	GetCodeSize(types.Address) int

	// AddRefund adds gas to the refund counter
	AddRefund(uint64)
	// SubRefund removes gas from the refund counter.
	// This method will panic if the refund counter goes below zero
	SubRefund(uint64)
	// GetRefund returns the current value of the refund counter.
	GetRefund() uint64

	GetCommittedState(types.Address, types.Hash) types.Hash
	// GetState retrieves a value from the given account's storage trie.
	GetState(types.Address, types.Hash) types.Hash
	SetState(types.Address, types.Hash, types.Hash)

	// Suicide marks the given account as suicided.
	// This clears the account balance.
	//
	// The account's state object is still available until the state is committed,
	// getStateObject will return a non-nil account after Suicide.
	Suicide(types.Address) bool
	HasSuicided(types.Address) bool

	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(types.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(types.Address) bool

	// RevertToSnapshot reverts all state changes made since the given revision.
	RevertToSnapshot(int)
	// Snapshot creates a new revision
	Snapshot() int

	AddLog(*Log)
	AddPreimage(types.Hash, []byte)

	ForEachStorage(types.Address, func(types.Hash, types.Hash) bool)
	// for debug.
	String() string

	// GetStateObject(addr types.Address) *state.StateObject
	// SetStateObject(addr types.Address, stateObject *state.StateObject)
}
