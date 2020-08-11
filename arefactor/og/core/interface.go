package core

import (
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/types"
	"github.com/annchain/commongo/math"

	// TODO move out evm package
	vmtypes "github.com/annchain/OG/vm/types"
)

type LedgerEngine interface {
	GetNonce(ogTypes.Address) uint64
	SetNonce(ogTypes.Address, uint64)
	IssueToken(issuer ogTypes.Address, name, symbol string, reIssuable bool, fstIssue *math.BigInt) (int32, error)
	ReIssueToken(tokenID int32, amount *math.BigInt) error
	DestroyToken(tokenID int32) error

	GetTokenBalance(ogTypes.Address, int32) *math.BigInt
	SubTokenBalance(ogTypes.Address, int32, *math.BigInt)
	AddTokenBalance(ogTypes.Address, int32, *math.BigInt)
}

type TxProcessor interface {
	Process(engine LedgerEngine, tx types.Txi) (*Receipt, error)
}

type VmProcessor interface {
	CanProcess(txi types.Txi) bool
	Process(engine VmStateDB, tx types.Txi, height uint64) (*Receipt, error)
	Call(engine VmStateDB)
}

// TODO try to delete those useless evm functions in OG.
// like AddRefund refund functions, Suicide functions, Log and PreImage functions etc.
type VmStateDB interface {
	CreateAccount(ogTypes.Address)

	SubBalance(ogTypes.Address, *math.BigInt)
	AddBalance(ogTypes.Address, *math.BigInt)
	// Retrieve the balance from the given address or 0 if object not found
	GetBalance(ogTypes.Address) *math.BigInt

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

	// Suicide marks the given account as suicided.
	// This clears the account balance.
	//
	// The account's state object is still available until the state is committed,
	// getStateObject will return a non-nil account after Suicide.
	Suicide(ogTypes.Address) bool
	HasSuicided(ogTypes.Address) bool

	// IsAddressExists reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(ogTypes.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(ogTypes.Address) bool

	// RevertToSnapshot reverts all state changes made since the given revision.
	RevertToSnapshot(int)
	// Snapshot creates a new revision
	Snapshot() int

	AddLog(log *vmtypes.Log)
	AddPreimage(ogTypes.Hash, []byte)

	ForEachStorage(ogTypes.Address, func(ogTypes.Hash, ogTypes.Hash) bool)
	// for debug.
	String() string
}
