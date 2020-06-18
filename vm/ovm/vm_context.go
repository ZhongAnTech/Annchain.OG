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
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/common/math"
	vmtypes "github.com/annchain/OG/vm/types"
	"math/big"
)

// TxContext represents all information that evm needs to know about the tx current processing.
type TxContext struct {
	From  ogTypes.Address
	To    ogTypes.Address
	Value *math.BigInt
	Data  []byte

	// Temporarily keep using gas as resource billing
	GasLimit   uint64
	GasPrice   *math.BigInt
	Coinbase   ogTypes.Address // Provides information for COINBASE
	SequenceID uint64          // Provides information for SequenceID
	//Time        *math.BigInt      // Provides information for TIME
	//Difficulty  *math.BigInt      // Provides information for DIFFICULTY
}

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
}

type DefaultChainContext struct {
}

// NewOVMContext creates a new context for use in the OVM.
func NewOVMContext(chainContext ChainContext, coinBase ogTypes.Address, stateDB vmtypes.StateDB) *vmtypes.Context {
	return &vmtypes.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		StateDB:     stateDB,
	}
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vmtypes.StateDB, addr ogTypes.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Value.Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vmtypes.StateDB, sender, recipient ogTypes.Address, amount *big.Int) {
	a := math.NewBigIntFromBigInt(amount)
	db.SubBalance(sender, a)
	db.AddBalance(recipient, a)
}
