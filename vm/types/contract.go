// Copyright 2015 The go-ethereum Authors
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
	"github.com/annchain/OG/common/crypto"

	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"

	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/code"
	"github.com/annchain/OG/vm/common"
	"github.com/annchain/OG/vm/instruction"

	"github.com/sirupsen/logrus"
	"math/big"
)

type CodeAndHash struct {
	Code []byte
	hash types.Hash
}

func (c *CodeAndHash) Hash() types.Hash {
	if c.hash == (types.Hash{}) {
		c.hash = crypto.Keccak256Hash(c.Code)
	}
	return c.hash
}

// ContractRef is a reference to the contract's backing object
type ContractRef interface {
	Address() types.Address
}

// AccountRef implements ContractRef.
//
// Account references are used during OVM initialisation and
// it's primary use is to fetch addresses. Removing this object
// proves difficult because of the cached jump destinations which
// are fetched from the parent contract (i.e. the caller), which
// is a ContractRef.
type AccountRef types.Address

// Address casts AccountRef to a Address
func (ar AccountRef) Address() types.Address { return (types.Address)(ar) }

// Contract represents an general contract in the state database. It contains
// the contract code, calling arguments. Contract implements ContractRef
type Contract struct {
	// CallerAddress is the result of the caller which initialised this
	// contract. However when the "call method" is delegated this Value
	// needs to be initialised to that of the caller's caller.
	CallerAddress types.Address
	caller        ContractRef
	self          ContractRef

	jumpdests map[types.Hash]common.Bitvec // Aggregated result of JUMPDEST analysis.
	analysis  common.Bitvec                // Locally cached result of JUMPDEST analysis

	Code     []byte
	CodeHash types.Hash
	CodeAddr *types.Address
	Input    []byte

	Gas   uint64
	value *big.Int
}

// NewContract returns a new contract environment for the execution of OVM.
func NewContract(caller ContractRef, object ContractRef, value *big.Int, gas uint64) *Contract {
	c := &Contract{CallerAddress: caller.Address(), caller: caller, self: object}

	if parent, ok := caller.(*Contract); ok {
		// Reuse JUMPDEST analysis from parent context if available.
		c.jumpdests = parent.jumpdests
	} else {
		c.jumpdests = make(map[types.Hash]common.Bitvec)
	}

	// Gas should be a pointer so it can safely be reduced through the run
	// This pointer will be off the state transition
	c.Gas = gas
	// ensures a Value is set
	c.value = value

	return c
}

func (c *Contract) ValidJumpdest(dest *big.Int) bool {
	udest := dest.Uint64()
	// PC cannot go beyond len(code) and certainly can't be bigger than 63bits.
	// Don't bother checking for JUMPDEST in that case.
	if dest.BitLen() >= 63 || udest >= uint64(len(c.Code)) {
		return false
	}
	// Only JUMPDESTs allowed for destinations
	if instruction.OpCode(c.Code[udest]) != instruction.JUMPDEST {
		return false
	}
	// Do we have a contract hash already?
	if c.CodeHash != (types.Hash{}) {
		// Does parent context have the analysis?
		analysis, exist := c.jumpdests[c.CodeHash]
		if !exist {
			// Do the analysis and save in parent context
			// We do not need to store it in c.analysis
			analysis = code.CodeBitmap(c.Code)
			c.jumpdests[c.CodeHash] = analysis
		}
		return analysis.CodeSegment(udest)
	}
	// We don't have the code hash, most likely a piece of initcode not already
	// in state trie. In that case, we do an analysis, and save it locally, so
	// we don't have to recalculate it for every JUMP instruction in the execution
	// However, we don't save it within the parent context
	if c.analysis == nil {
		c.analysis = code.CodeBitmap(c.Code)
	}
	return c.analysis.CodeSegment(udest)
}

// AsDelegate sets the contract to be a delegate call and returns the current
// contract (for chaining calls)
func (c *Contract) AsDelegate() *Contract {
	// NOTE: caller must, at all times be a contract. It should never happen
	// that caller is something other than a Contract.
	parent := c.caller.(*Contract)
	c.CallerAddress = parent.CallerAddress
	c.value = parent.value

	return c
}

// GetOp returns the n'th element in the contract's byte array
func (c *Contract) GetOp(n uint64) instruction.OpCode {
	return instruction.OpCode(c.GetByte(n))
}

// GetByte returns the n'th byte in the contract's byte array
func (c *Contract) GetByte(n uint64) byte {
	if n < uint64(len(c.Code)) {
		return c.Code[n]
	}

	return 0
}

// Caller returns the caller of the contract.
//
// Caller will recursively call caller when the contract is a delegate
// call, including that of caller's caller.
func (c *Contract) Caller() types.Address {
	return c.CallerAddress
}

// UseGas attempts the use gas and subtracts it and returns true on success
func (c *Contract) UseGas(gas uint64) (ok bool) {
	if c.Gas < gas {
		return false
	}
	c.Gas -= gas
	return true
}

// Address returns the contracts address
func (c *Contract) Address() types.Address {
	return c.self.Address()
}

// Value returns the contracts Value (sent to it from it's caller)
func (c *Contract) Value() *big.Int {
	return c.value
}

// SetCallCode sets the code of the contract and address of the backing data
// object
func (c *Contract) SetCallCode(addr *types.Address, hash types.Hash, code []byte) {
	logrus.WithFields(logrus.Fields{
		"addr":  addr.Hex(),
		"hash":  hash.Hex(),
		"bytes": hexutil.Encode(code[0:math.MinInt(20, len(code))]) + "...",
	}).Info("SetCallCode")
	c.Code = code
	c.CodeHash = hash
	c.CodeAddr = addr
}

// SetCodeOptionalHash can be used to provide code, but it's optional to provide hash.
// In case hash is not provided, the jumpdest analysis will not be saved to the parent context
func (c *Contract) SetCodeOptionalHash(addr *types.Address, codeAndHash *CodeAndHash) {
	c.Code = codeAndHash.Code
	c.CodeHash = codeAndHash.hash
	c.CodeAddr = addr
}
