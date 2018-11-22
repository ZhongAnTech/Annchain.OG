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

package ovm

import (
	"math/big"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/annchain/OG/common/math"
)

// TxContext represents all information that evm needs to know about the tx current processing.
type TxContext struct {
	From  types.Address
	To    types.Address
	Value *math.BigInt
	Data  []byte

	// Temporarily keep using gas as resource billing
	GasLimit uint64
	GasPrice *math.BigInt
}

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
}

type DefaultChainContext struct {
}

// NewEVMContext creates a new context for use in the OVM.
func NewEVMContext(txContext *TxContext, chainContext ChainContext, coinBase *types.Address) vmtypes.Context {
	return vmtypes.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		Origin:      txContext.From,
		//Coinbase:    beneficiary,
		GasLimit: txContext.GasLimit,
		GasPrice: txContext.GasPrice.Value,
	}
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vmtypes.StateDB, addr types.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vmtypes.StateDB, sender, recipient types.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// Config are the configuration options for the Interpreter
type Config struct {
	// Debug enabled debugging Interpreter options
	Debug bool
	// Tracer is the op code logger
	Tracer Tracer
	// NoRecursion disabled Interpreter call, callcode,
	// delegate call and create.
	NoRecursion bool
	// Enable recording of SHA3/keccak preimages
	EnablePreimageRecording bool
}
