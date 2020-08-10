package state

import (
	"github.com/annchain/OG/arefactor/common/math"
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
)

// StateDB is an OVM database for full state querying.
type StateDBInterface interface {
	CreateAccount(ogTypes.Address)

	SubBalance(ogTypes.Address, *math.BigInt)
	SubTokenBalance(ogTypes.Address, int32, *math.BigInt)
	AddBalance(ogTypes.Address, *math.BigInt)
	AddTokenBalance(ogTypes.Address, int32, *math.BigInt)
	SetTokenBalance(ogTypes.Address, int32, *math.BigInt)
	// Retrieve the balance from the given address or 0 if object not found
	GetBalance(ogTypes.Address) *math.BigInt
	GetTokenBalance(ogTypes.Address, int32) *math.BigInt

	GetNonce(ogTypes.Address) uint64
	SetNonce(ogTypes.Address, uint64)

	GetCodeHash(ogTypes.Address) ogTypes.Hash
	GetCode(ogTypes.Address) []byte
	SetCode(ogTypes.Address, []byte)
	GetCodeSize(ogTypes.Address) int

	// AddRefund adds gas to the refund counter
	AddRefund(uint64)
	// SubRefund removes gas from the refund counter.
	// This method will panic if the refund counter goes below zero
	SubRefund(uint64)
	// GetRefund returns the current value of the refund counter.
	GetRefund() uint64

	GetCommittedState(ogTypes.Address, ogTypes.Hash) ogTypes.Hash
	// GetState retrieves a value from the given account's storage trie.
	GetState(ogTypes.Address, ogTypes.Hash) ogTypes.Hash
	SetState(ogTypes.Address, ogTypes.Hash, ogTypes.Hash)

	AppendJournal(JournalEntry)

	// Suicide marks the given account as suicided.
	// This clears the account balance.
	//
	// The account's state object is still available until the state is committed,
	// getStateObject will return a non-nil account after Suicide.
	Suicide(ogTypes.Address) bool
	HasSuicided(ogTypes.Address) bool

	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(ogTypes.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(ogTypes.Address) bool

	// RevertToSnapshot reverts all state changes made since the given revision.
	RevertToSnapshot(int)
	// Snapshot creates a new revision
	Snapshot() int

	//AddLog(*Log)
	AddPreimage(ogTypes.Hash, []byte)

	ForEachStorage(ogTypes.Address, func(ogTypes.Hash, ogTypes.Hash) bool)
	// for debug.
	String() string
}
