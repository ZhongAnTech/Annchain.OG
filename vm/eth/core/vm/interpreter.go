// Copyright 2014 The go-ethereum Authors
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
	"fmt"
	"hash"
	"sync/atomic"

	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/eth/common"
	"github.com/annchain/OG/vm/eth/common/math"
	"github.com/annchain/OG/vm/eth/params"
	"github.com/annchain/OG/vm/instruction"
	vmtypes "github.com/annchain/OG/vm/types"
)

// keccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

// EVMInterpreter represents an OVM interpreter
//
type EVMInterpreter struct {
	//ovm      *ovm.OVM
	ctx        *vmtypes.Context
	Cfg        *InterpreterConfig
	gasTable   params.GasTable
	intPool    *intPool
	hasher     keccakState    // Keccak256 hasher instance shared across opcodes
	hasherBuf  types.Hash     // Keccak256 hasher result array shared aross opcodes
	jumpTable  [256]operation // JumpTable contains the OVM instruction table.
	readOnly   bool           // Whether to throw on stateful modifications
	returnData []byte         // Last CALL's return data for subsequent reuse
}

// NewEVMInterpreter returns a new instance of the Interpreter.
func NewEVMInterpreter(ctx *vmtypes.Context, cfg *InterpreterConfig) *EVMInterpreter {
	return &EVMInterpreter{
		ctx:       ctx,
		Cfg:       cfg,
		gasTable:  params.GetGasTable(ctx.SequenceID),
		jumpTable: byzantiumInstructionSet,
	}
}

func (in *EVMInterpreter) enforceRestrictions(op instruction.OpCode, operation operation, stack *Stack) error {
	if in.readOnly {
		// If the interpreter is operating in readonly mode, make sure no
		// state-modifying operation is performed. The 3rd stack item
		// for a call operation is the value. Transferring value from one
		// account to the others means the state is modified and should also
		// return with an error.
		if operation.writes || (op == instruction.CALL && stack.Back(2).BitLen() > 0) {
			return vmtypes.ErrWriteProtection
		}
	}
	return nil
}

// Run loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// It's important to note that any errors returned by the interpreter should be
// considered a revert-and-consume-all-gas operation except for
// errExecutionReverted which means revert-and-keep-gas-left.
func (in *EVMInterpreter) Run(contract *vmtypes.Contract, input []byte, readOnly bool) (ret []byte, err error) {
	if in.intPool == nil {
		in.intPool = poolOfIntPools.get()
		defer func() {
			poolOfIntPools.put(in.intPool)
			in.intPool = nil
		}()
	}

	// Increment the call depth which is restricted to 1024
	in.ctx.Depth++
	defer func() { in.ctx.Depth-- }()

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This makes also sure that the readOnly flag isn't removed for child calls.
	if readOnly && !in.readOnly {
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	in.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	var (
		op    instruction.OpCode // current opcode
		mem   = NewMemory()      // bound memory
		stack = newstack()       // local stack
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc   = uint64(0) // program counter
		cost uint64
		// copies used by tracer
		pcCopy  uint64 // needed for the deferred Tracer
		gasCopy uint64 // for Tracer to log gas remaining before execution
		logged  bool   // deferred Tracer should ignore already logged steps
	)
	contract.Input = input

	// Reclaim the stack as an int pool when the execution stops
	defer func() { in.intPool.put(stack.data...) }()

	if in.Cfg.Debug {
		defer func() {
			if err != nil {
				if !logged {
					in.Cfg.Tracer.CaptureState(in.ctx, pcCopy, op, gasCopy, cost, mem, stack, contract, in.ctx.Depth, err)
				} else {
					in.Cfg.Tracer.CaptureFault(in.ctx, pcCopy, op, gasCopy, cost, mem, stack, contract, in.ctx.Depth, err)
				}
			}
		}()
	}
	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	for atomic.LoadInt32(&in.ctx.Abort) == 0 {
		if in.Cfg.Debug {
			// Capture pre-execution values for tracing.
			logged, pcCopy, gasCopy = false, pc, contract.Gas
		}

		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.GetOp(pc)
		operation := in.jumpTable[op]
		if !operation.valid {
			return nil, fmt.Errorf("invalid opcode 0x%x", int(op))
		}
		if err := operation.validateStack(stack); err != nil {
			return nil, err
		}
		// If the operation is valid, enforce and write restrictions
		if err := in.enforceRestrictions(op, operation, stack); err != nil {
			return nil, err
		}
		fmt.Printf("%d OP: %s\n", pc, op.String())
		//if op.String() == "CALLER" {
		//	fmt.Println(pc, "CALLER")
		//}
		//if pc == 392{
		//	fmt.Println(pc, "PC")
		//}

		//stack.Print()
		//mem.Print()
		//fmt.Println(in.ctx.StateDB.String())

		var memorySize uint64
		// calculate the new memory size and expand the memory to fit
		// the operation
		if operation.memorySize != nil {
			memSize, overflow := common.BigUint64(operation.memorySize(stack))
			if overflow {
				return nil, errGasUintOverflow
			}
			// memory is expanded in words of 32 bytes. Gas
			// is also calculated in words.
			if memorySize, overflow = math.SafeMul(common.ToWordSize(memSize), 32); overflow {
				return nil, errGasUintOverflow
			}
		}
		// consume the gas and return an error if not enough gas is available.
		// cost is explicitly set so that the capture state defer method can get the proper cost
		cost, err = operation.gasCost(in.gasTable, in.ctx, contract, stack, mem, memorySize)
		if err != nil || !contract.UseGas(cost) {
			return nil, vmtypes.ErrOutOfGas
		}
		if memorySize > 0 {
			mem.Resize(memorySize)
		}

		if in.Cfg.Debug {
			in.Cfg.Tracer.CaptureState(in.ctx, pc, op, gasCopy, cost, mem, stack, contract, in.ctx.Depth, err)
			logged = true
		}

		// execute the operation
		res, err := operation.execute(&pc, in, contract, mem, stack)
		// verifyPool is a build flag. Pool verification makes sure the integrity
		// of the integer pool by comparing values to a default value.
		if verifyPool {
			verifyIntegerPool(in.intPool)
		}
		// if the operation clears the return data (e.g. it has returning data)
		// set the last return to the result of the operation.
		if operation.returns {
			in.returnData = res
		}

		switch {
		case err != nil:
			return nil, err
		case operation.reverts:
			return res, vmtypes.ErrExecutionReverted
		case operation.halts:
			return res, nil
		case !operation.jumps:
			pc++
		}

		if op.String() == "SSTORE" {
			fmt.Println(in.ctx.StateDB.String())
			stack.Print()
			mem.Print()
		}

		//stack.Print()
		//mem.Print()

	}
	return nil, nil
}

// CanRun tells if the contract, passed as an argument, can be
// run by the current interpreter.
func (in *EVMInterpreter) CanRun(code []byte) bool {
	return true
}
