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

package types

import (
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"math/big"
)

// Caller provides a basic interface for the OVM calling conventions. The OVM
// depends on this context being implemented for doing subcalls and initialising new OVM contracts.
type Caller interface {
	// Call another contract
	Call(me ContractRef, addr ogTypes.Address20, data []byte, gas uint64, value *big.Int, txCall bool) (resp []byte, leftOverGas uint64, err error)
	// Take another's contract code and execute within our own context
	CallCode(me ContractRef, addr ogTypes.Address20, data []byte, gas uint64, value *big.Int) (resp []byte, leftOverGas uint64, err error)
	// Same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(me ContractRef, addr ogTypes.Address20, data []byte, gas uint64) (resp []byte, leftOverGas uint64, err error)
	// Create a new contract
	Create(me ContractRef, data []byte, gas uint64, value *big.Int, txCall bool) (resp []byte, contractAddr ogTypes.Address20, leftOverGas uint64, err error)
	// Create a new contract use sha3
	Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int, txCall bool) (ret []byte, contractAddr ogTypes.Address20, leftOverGas uint64, err error)

	StaticCall(caller ContractRef, addr ogTypes.Address20, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error)
}
