package types

import (
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"math/big"
)

type (
	// CanTransferFunc is the signature of a transfer guard function
	CanTransferFunc func(StateDB, ogTypes.Address20, *big.Int) bool
	// TransferFunc is the signature of a transfer function
	TransferFunc func(StateDB, ogTypes.Address20, ogTypes.Address20, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH OVM op code.
	GetHashFunc func(uint64) ogTypes.Hash
)

// Context provides the OVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient ether to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers ether from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	//GetHash GetHashFunc
	StateDB     StateDB
	CallGasTemp uint64
	// Depth is the current call stack
	Depth int

	// abort is used to abort the OVM calling operations
	// NOTE: must be set atomically
	Abort int32
}
