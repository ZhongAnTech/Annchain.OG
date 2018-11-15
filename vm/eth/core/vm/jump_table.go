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

package vm

import (
	"errors"
	"math/big"

	"github.com/annchain/OG/vm/eth/params"
	"github.com/annchain/OG/vm/instruction"
	"github.com/annchain/OG/vm/vmcommon"
)

type (
	executionFunc       func(pc *uint64, interpreter *EVMInterpreter, contract *vmcommon.Contract, memory *Memory, stack *Stack) ([]byte, error)
	gasFunc             func(params.GasTable, *EVM, *vmcommon.Contract, *Stack, *Memory, uint64) (uint64, error) // last parameter is the requested memory size as a uint64
	stackValidationFunc func(*Stack) error
	memorySizeFunc      func(*Stack) *big.Int
)

var errGasUintOverflow = errors.New("gas uint64 overflow")

type operation struct {
	// execute is the operation function
	execute executionFunc
	// gasCost is the gas function and returns the gas required for execution
	gasCost gasFunc
	// validateStack validates the stack (size) for the operation
	validateStack stackValidationFunc
	// memorySize returns the memory size required for the operation
	memorySize memorySizeFunc

	halts   bool // indicates whether the operation should halt further execution
	jumps   bool // indicates whether the program counter should not increment
	writes  bool // determines whether this a state modifying operation
	valid   bool // indication whether the retrieved operation is valid and known
	reverts bool // determines whether the operation reverts state (implicitly halts)
	returns bool // determines whether the operations sets the return data content
}

var (
	byzantiumInstructionSet      = newByzantiumInstructionSet()
)


// NewByzantiumInstructionSet returns the frontier, homestead and
// byzantium instructions.
func newByzantiumInstructionSet() [256]operation {
	// instructions that can be executed during the homestead phase.
	instructionSet := newHomesteadInstructionSet()
	instructionSet[instruction.STATICCALL] = operation{
		execute:       opStaticCall,
		gasCost:       gasStaticCall,
		validateStack: makeStackFunc(6, 1),
		memorySize:    memoryStaticCall,
		valid:         true,
		returns:       true,
	}
	instructionSet[instruction.RETURNDATASIZE] = operation{
		execute:       opReturnDataSize,
		gasCost:       constGasFunc(GasQuickStep),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	instructionSet[instruction.RETURNDATACOPY] = operation{
		execute:       opReturnDataCopy,
		gasCost:       gasReturnDataCopy,
		validateStack: makeStackFunc(3, 0),
		memorySize:    memoryReturnDataCopy,
		valid:         true,
	}
	instructionSet[instruction.REVERT] = operation{
		execute:       opRevert,
		gasCost:       gasRevert,
		validateStack: makeStackFunc(2, 0),
		memorySize:    memoryRevert,
		valid:         true,
		reverts:       true,
		returns:       true,
	}
	return instructionSet
}

// NewHomesteadInstructionSet returns the frontier and homestead
// instructions that can be executed during the homestead phase.
func newHomesteadInstructionSet() [256]operation {
	instructionSet := newFrontierInstructionSet()
	instructionSet[instruction.DELEGATECALL] = operation{
		execute:       opDelegateCall,
		gasCost:       gasDelegateCall,
		validateStack: makeStackFunc(6, 1),
		memorySize:    memoryDelegateCall,
		valid:         true,
		returns:       true,
	}
	return instructionSet
}

// NewFrontierInstructionSet returns the frontier instructions
// that can be executed during the frontier phase.
func newFrontierInstructionSet() [256]operation {
	return [256]operation{
		instruction.STOP: {
			execute:       opStop,
			gasCost:       constGasFunc(0),
			validateStack: makeStackFunc(0, 0),
			halts:         true,
			valid:         true,
		},
		instruction.ADD: {
			execute:       opAdd,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.MUL: {
			execute:       opMul,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.SUB: {
			execute:       opSub,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.DIV: {
			execute:       opDiv,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.SDIV: {
			execute:       opSdiv,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.MOD: {
			execute:       opMod,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.SMOD: {
			execute:       opSmod,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.ADDMOD: {
			execute:       opAddmod,
			gasCost:       constGasFunc(GasMidStep),
			validateStack: makeStackFunc(3, 1),
			valid:         true,
		},
		instruction.MULMOD: {
			execute:       opMulmod,
			gasCost:       constGasFunc(GasMidStep),
			validateStack: makeStackFunc(3, 1),
			valid:         true,
		},
		instruction.EXP: {
			execute:       opExp,
			gasCost:       gasExp,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.SIGNEXTEND: {
			execute:       opSignExtend,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.LT: {
			execute:       opLt,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.GT: {
			execute:       opGt,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.SLT: {
			execute:       opSlt,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.SGT: {
			execute:       opSgt,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.EQ: {
			execute:       opEq,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.ISZERO: {
			execute:       opIszero,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		instruction.AND: {
			execute:       opAnd,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.XOR: {
			execute:       opXor,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.OR: {
			execute:       opOr,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.NOT: {
			execute:       opNot,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		instruction.BYTE: {
			execute:       opByte,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		instruction.SHA3: {
			execute:       opSha3,
			gasCost:       gasSha3,
			validateStack: makeStackFunc(2, 1),
			memorySize:    memorySha3,
			valid:         true,
		},
		instruction.ADDRESS: {
			execute:       opAddress,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.BALANCE: {
			execute:       opBalance,
			gasCost:       gasBalance,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		instruction.ORIGIN: {
			execute:       opOrigin,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.CALLER: {
			execute:       opCaller,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.CALLVALUE: {
			execute:       opCallValue,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.CALLDATALOAD: {
			execute:       opCallDataLoad,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		instruction.CALLDATASIZE: {
			execute:       opCallDataSize,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.CALLDATACOPY: {
			execute:       opCallDataCopy,
			gasCost:       gasCallDataCopy,
			validateStack: makeStackFunc(3, 0),
			memorySize:    memoryCallDataCopy,
			valid:         true,
		},
		instruction.CODESIZE: {
			execute:       opCodeSize,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.CODECOPY: {
			execute:       opCodeCopy,
			gasCost:       gasCodeCopy,
			validateStack: makeStackFunc(3, 0),
			memorySize:    memoryCodeCopy,
			valid:         true,
		},
		instruction.GASPRICE: {
			execute:       opGasprice,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.EXTCODESIZE: {
			execute:       opExtCodeSize,
			gasCost:       gasExtCodeSize,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		instruction.EXTCODECOPY: {
			execute:       opExtCodeCopy,
			gasCost:       gasExtCodeCopy,
			validateStack: makeStackFunc(4, 0),
			memorySize:    memoryExtCodeCopy,
			valid:         true,
		},
		instruction.BLOCKHASH: {
			execute:       opBlockhash,
			gasCost:       constGasFunc(GasExtStep),
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		instruction.COINBASE: {
			execute:       opCoinbase,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.TIMESTAMP: {
			execute:       opTimestamp,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.NUMBER: {
			execute:       opNumber,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.DIFFICULTY: {
			execute:       opDifficulty,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.GASLIMIT: {
			execute:       opGasLimit,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.POP: {
			execute:       opPop,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(1, 0),
			valid:         true,
		},
		instruction.MLOAD: {
			execute:       opMload,
			gasCost:       gasMLoad,
			validateStack: makeStackFunc(1, 1),
			memorySize:    memoryMLoad,
			valid:         true,
		},
		instruction.MSTORE: {
			execute:       opMstore,
			gasCost:       gasMStore,
			validateStack: makeStackFunc(2, 0),
			memorySize:    memoryMStore,
			valid:         true,
		},
		instruction.MSTORE8: {
			execute:       opMstore8,
			gasCost:       gasMStore8,
			memorySize:    memoryMStore8,
			validateStack: makeStackFunc(2, 0),

			valid: true,
		},
		instruction.SLOAD: {
			execute:       opSload,
			gasCost:       gasSLoad,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		instruction.SSTORE: {
			execute:       opSstore,
			gasCost:       gasSStore,
			validateStack: makeStackFunc(2, 0),
			valid:         true,
			writes:        true,
		},
		instruction.JUMP: {
			execute:       opJump,
			gasCost:       constGasFunc(GasMidStep),
			validateStack: makeStackFunc(1, 0),
			jumps:         true,
			valid:         true,
		},
		instruction.JUMPI: {
			execute:       opJumpi,
			gasCost:       constGasFunc(GasSlowStep),
			validateStack: makeStackFunc(2, 0),
			jumps:         true,
			valid:         true,
		},
		instruction.PC: {
			execute:       opPc,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.MSIZE: {
			execute:       opMsize,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.GAS: {
			execute:       opGas,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.JUMPDEST: {
			execute:       opJumpdest,
			gasCost:       constGasFunc(params.JumpdestGas),
			validateStack: makeStackFunc(0, 0),
			valid:         true,
		},
		instruction.PUSH1: {
			execute:       makePush(1, 1),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH2: {
			execute:       makePush(2, 2),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH3: {
			execute:       makePush(3, 3),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH4: {
			execute:       makePush(4, 4),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH5: {
			execute:       makePush(5, 5),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH6: {
			execute:       makePush(6, 6),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH7: {
			execute:       makePush(7, 7),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH8: {
			execute:       makePush(8, 8),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH9: {
			execute:       makePush(9, 9),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH10: {
			execute:       makePush(10, 10),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH11: {
			execute:       makePush(11, 11),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH12: {
			execute:       makePush(12, 12),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH13: {
			execute:       makePush(13, 13),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH14: {
			execute:       makePush(14, 14),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH15: {
			execute:       makePush(15, 15),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH16: {
			execute:       makePush(16, 16),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH17: {
			execute:       makePush(17, 17),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH18: {
			execute:       makePush(18, 18),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH19: {
			execute:       makePush(19, 19),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH20: {
			execute:       makePush(20, 20),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH21: {
			execute:       makePush(21, 21),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH22: {
			execute:       makePush(22, 22),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH23: {
			execute:       makePush(23, 23),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH24: {
			execute:       makePush(24, 24),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH25: {
			execute:       makePush(25, 25),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH26: {
			execute:       makePush(26, 26),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH27: {
			execute:       makePush(27, 27),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH28: {
			execute:       makePush(28, 28),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH29: {
			execute:       makePush(29, 29),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH30: {
			execute:       makePush(30, 30),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH31: {
			execute:       makePush(31, 31),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.PUSH32: {
			execute:       makePush(32, 32),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		instruction.DUP1: {
			execute:       makeDup(1),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(1),
			valid:         true,
		},
		instruction.DUP2: {
			execute:       makeDup(2),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(2),
			valid:         true,
		},
		instruction.DUP3: {
			execute:       makeDup(3),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(3),
			valid:         true,
		},
		instruction.DUP4: {
			execute:       makeDup(4),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(4),
			valid:         true,
		},
		instruction.DUP5: {
			execute:       makeDup(5),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(5),
			valid:         true,
		},
		instruction.DUP6: {
			execute:       makeDup(6),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(6),
			valid:         true,
		},
		instruction.DUP7: {
			execute:       makeDup(7),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(7),
			valid:         true,
		},
		instruction.DUP8: {
			execute:       makeDup(8),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(8),
			valid:         true,
		},
		instruction.DUP9: {
			execute:       makeDup(9),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(9),
			valid:         true,
		},
		instruction.DUP10: {
			execute:       makeDup(10),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(10),
			valid:         true,
		},
		instruction.DUP11: {
			execute:       makeDup(11),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(11),
			valid:         true,
		},
		instruction.DUP12: {
			execute:       makeDup(12),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(12),
			valid:         true,
		},
		instruction.DUP13: {
			execute:       makeDup(13),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(13),
			valid:         true,
		},
		instruction.DUP14: {
			execute:       makeDup(14),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(14),
			valid:         true,
		},
		instruction.DUP15: {
			execute:       makeDup(15),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(15),
			valid:         true,
		},
		instruction.DUP16: {
			execute:       makeDup(16),
			gasCost:       gasDup,
			validateStack: makeDupStackFunc(16),
			valid:         true,
		},
		instruction.SWAP1: {
			execute:       makeSwap(1),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(2),
			valid:         true,
		},
		instruction.SWAP2: {
			execute:       makeSwap(2),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(3),
			valid:         true,
		},
		instruction.SWAP3: {
			execute:       makeSwap(3),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(4),
			valid:         true,
		},
		instruction.SWAP4: {
			execute:       makeSwap(4),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(5),
			valid:         true,
		},
		instruction.SWAP5: {
			execute:       makeSwap(5),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(6),
			valid:         true,
		},
		instruction.SWAP6: {
			execute:       makeSwap(6),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(7),
			valid:         true,
		},
		instruction.SWAP7: {
			execute:       makeSwap(7),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(8),
			valid:         true,
		},
		instruction.SWAP8: {
			execute:       makeSwap(8),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(9),
			valid:         true,
		},
		instruction.SWAP9: {
			execute:       makeSwap(9),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(10),
			valid:         true,
		},
		instruction.SWAP10: {
			execute:       makeSwap(10),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(11),
			valid:         true,
		},
		instruction.SWAP11: {
			execute:       makeSwap(11),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(12),
			valid:         true,
		},
		instruction.SWAP12: {
			execute:       makeSwap(12),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(13),
			valid:         true,
		},
		instruction.SWAP13: {
			execute:       makeSwap(13),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(14),
			valid:         true,
		},
		instruction.SWAP14: {
			execute:       makeSwap(14),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(15),
			valid:         true,
		},
		instruction.SWAP15: {
			execute:       makeSwap(15),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(16),
			valid:         true,
		},
		instruction.SWAP16: {
			execute:       makeSwap(16),
			gasCost:       gasSwap,
			validateStack: makeSwapStackFunc(17),
			valid:         true,
		},
		instruction.LOG0: {
			execute:       makeLog(0),
			gasCost:       makeGasLog(0),
			validateStack: makeStackFunc(2, 0),
			memorySize:    memoryLog,
			valid:         true,
			writes:        true,
		},
		instruction.LOG1: {
			execute:       makeLog(1),
			gasCost:       makeGasLog(1),
			validateStack: makeStackFunc(3, 0),
			memorySize:    memoryLog,
			valid:         true,
			writes:        true,
		},
		instruction.LOG2: {
			execute:       makeLog(2),
			gasCost:       makeGasLog(2),
			validateStack: makeStackFunc(4, 0),
			memorySize:    memoryLog,
			valid:         true,
			writes:        true,
		},
		instruction.LOG3: {
			execute:       makeLog(3),
			gasCost:       makeGasLog(3),
			validateStack: makeStackFunc(5, 0),
			memorySize:    memoryLog,
			valid:         true,
			writes:        true,
		},
		instruction.LOG4: {
			execute:       makeLog(4),
			gasCost:       makeGasLog(4),
			validateStack: makeStackFunc(6, 0),
			memorySize:    memoryLog,
			valid:         true,
			writes:        true,
		},
		instruction.CREATE: {
			execute:       opCreate,
			gasCost:       gasCreate,
			validateStack: makeStackFunc(3, 1),
			memorySize:    memoryCreate,
			valid:         true,
			writes:        true,
			returns:       true,
		},
		instruction.CALL: {
			execute:       opCall,
			gasCost:       gasCall,
			validateStack: makeStackFunc(7, 1),
			memorySize:    memoryCall,
			valid:         true,
			returns:       true,
		},
		instruction.CALLCODE: {
			execute:       opCallCode,
			gasCost:       gasCallCode,
			validateStack: makeStackFunc(7, 1),
			memorySize:    memoryCall,
			valid:         true,
			returns:       true,
		},
		instruction.RETURN: {
			execute:       opReturn,
			gasCost:       gasReturn,
			validateStack: makeStackFunc(2, 0),
			memorySize:    memoryReturn,
			halts:         true,
			valid:         true,
		},
		instruction.SELFDESTRUCT: {
			execute:       opSuicide,
			gasCost:       gasSuicide,
			validateStack: makeStackFunc(1, 0),
			halts:         true,
			valid:         true,
			writes:        true,
		},
	}
}
