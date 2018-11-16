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

package vm

import (
	"math/big"
	"github.com/annchain/OG/vm/ovm"
	"github.com/annchain/OG/types"
)

// CallContext provides a basic interface for the OVM calling conventions. The OVM
// depends on this context being implemented for doing subcalls and initialising new OVM contracts.
type CallContext interface {
	// Call another contract
	Call(env *ovm.OVM, me ovm.ContractRef, addr types.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// Take another's contract code and execute within our own context
	CallCode(env *ovm.OVM, me ovm.ContractRef, addr types.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// Same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(env *ovm.OVM, me ovm.ContractRef, addr types.Address, data []byte, gas *big.Int) ([]byte, error)
	// Create a new contract
	Create(env *ovm.OVM, me ovm.ContractRef, data []byte, gas, value *big.Int) ([]byte, types.Address, error)
}
