package vmcommon

import (
	"math/big"
	"github.com/annchain/OG/types"
)

// StateDB is an EVM database for full state querying.
type StateDB interface {
	CreateAccount(types.Address)

	SubBalance(types.Address, *big.Int)
	AddBalance(types.Address, *big.Int)
	GetBalance(types.Address) *big.Int

	GetNonce(types.Address) uint64
	SetNonce(types.Address, uint64)

	GetCodeHash(types.Address) types.Hash
	GetCode(types.Address) []byte
	SetCode(types.Address, []byte)
	GetCodeSize(types.Address) int

	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64

	GetCommittedState(types.Address, types.Hash) types.Hash
	GetState(types.Address, types.Hash) types.Hash
	SetState(types.Address, types.Hash, types.Hash)

	Suicide(types.Address) bool
	HasSuicided(types.Address) bool

	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(types.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(types.Address) bool

	RevertToSnapshot(int)
	Snapshot() int

	AddLog(*types.Log)
	AddPreimage(types.Hash, []byte)

	ForEachStorage(types.Address, func(types.Hash, types.Hash) bool)
}
