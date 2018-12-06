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

package ovm

import (
	"math/big"
	"sync/atomic"
	"github.com/annchain/OG/vm/eth/crypto"
	"github.com/annchain/OG/vm/eth/params"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
)

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var emptyCodeHash = crypto.Keccak256Hash(nil)

// run runs the given contract and takes care of running precompiles with a fallback to the byte Code interpreter.
func run(evm *OVM, contract *vmtypes.Contract, input []byte, readOnly bool) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContractsByzantium
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	for _, interpreter := range evm.Interpreters {
		if interpreter.CanRun(contract.Code) {
			if evm.Interpreter != interpreter {
				// Ensure that the interpreter pointer is set back
				// to its current value upon return.
				defer func(i Interpreter) {
					evm.Interpreter = i
				}(evm.Interpreter)
				evm.Interpreter = interpreter
			}
			return interpreter.Run(contract, input, readOnly)
		}
	}
	return nil, vmtypes.ErrNoCompatibleInterpreter
}

// OVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty Code.
//
// The OVM should never be reused and is not thread safe.
type OVM struct {
	// Context provides auxiliary blockchain related information
	vmtypes.Context
	// StateDB gives access to the underlying state
	// StateDB ovm.StateDB moved to Context

	// chainConfig contains information about the current chain
	// ChainConfig *params.ChainConfig
	// chain rules contains the chain rules for the current epoch
	// chainRules params.Rules
	// virtual machine configuration options used to initialise the
	// evm.
	OVMConfigs *OVMConfig
	// global (to this context) ethereum virtual machine
	// used throughout the execution of the tx.
	Interpreters []Interpreter
	Interpreter  Interpreter
	// abort is used to abort the OVM calling operations
	// NOTE: must be set atomically
	// Abort int32
	// callGasTemp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	// CallGasTemp uint64 moved to Context
}

// NewOVM returns a new OVM. The returned OVM is not thread safe and should
// only ever be used *once*.
func NewOVM(ctx vmtypes.Context, supportInterpreters []Interpreter, ovmConfig *OVMConfig) *OVM {
	evm := &OVM{
		Context:    ctx,
		OVMConfigs: ovmConfig,
		//chainRules:   chainConfig.Rules(ctx.SequenceID),
		Interpreters: supportInterpreters,
	}
	if evm.Interpreter != nil {

		evm.Interpreter = evm.Interpreters[0]
	}
	evm.Interpreter = evm.Interpreters[0]

	evm.StateDB = ctx.StateDB

	return evm
}

// Cancel cancels any running OVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (ovm *OVM) Cancel() {
	atomic.StoreInt32(&ovm.Abort, 1)
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (ovm *OVM) Call(ctx *vmtypes.Context, caller vmtypes.ContractRef, addr types.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if ovm.OVMConfigs.NoRecursion && ovm.Depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, gas, vmtypes.ErrDepth
	}
	// Fail if we're trying to transfer more than the available Balance
	if !ovm.Context.CanTransfer(ovm.StateDB, caller.Address(), value) {
		return nil, gas, vmtypes.ErrInsufficientBalance
	}

	var (
		to       = vmtypes.AccountRef(addr)
		snapshot = ovm.StateDB.Snapshot()
	)
	if !ovm.StateDB.Exist(addr) {
		precompiles := PrecompiledContractsByzantium
		if precompiles[addr] == nil && value.Sign() == 0 {
			// Calling a non existing account, don't do anything, but ping the tracer
			//if ovm.InterpreterConfig.Debug && ovm.Depth == 0 {
			//	ovm.InterpreterConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
			//	ovm.InterpreterConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
			//}
			return nil, gas, nil
		}
		ovm.StateDB.CreateAccount(addr)
	}
	ovm.Transfer(ovm.StateDB, caller.Address(), to.Address(), value)
	// Initialise a new contract and set the Code that is to be used by the OVM.
	// The contract is a scoped environment for this execution context only.
	contract := vmtypes.NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, ovm.StateDB.GetCodeHash(addr), ovm.StateDB.GetCode(addr))

	// Even if the account has no Code, we need to continue because it might be a precompile
	//start := time.Now()

	// Capture the tracer start/end events in debug mode
	//if ovm.InterpreterConfig.Debug && ovm.Depth == 0 {
	//	ovm.InterpreterConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
	//
	//	defer func() { // Lazy evaluation of the parameters
	//		ovm.InterpreterConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	//	}()
	//}
	ret, err = run(ovm, contract, input, false)

	// When an error was returned by the OVM or when setting the creation Code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for Code storage gas errors.
	if err != nil {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != vmtypes.ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// CallCode executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// Code with the caller as context.
func (ovm *OVM) CallCode(ctx *vmtypes.Context, caller vmtypes.ContractRef, addr types.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if ovm.OVMConfigs.NoRecursion && ovm.Depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, gas, vmtypes.ErrDepth
	}
	// Fail if we're trying to transfer more than the available Balance
	if !ovm.CanTransfer(ovm.StateDB, caller.Address(), value) {
		return nil, gas, vmtypes.ErrInsufficientBalance
	}

	var (
		snapshot = ovm.StateDB.Snapshot()
		to       = vmtypes.AccountRef(caller.Address())
	)
	// initialise a new contract and set the Code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := vmtypes.NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, ovm.StateDB.GetCodeHash(addr), ovm.StateDB.GetCode(addr))

	ret, err = run(ovm, contract, input, false)
	if err != nil {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != vmtypes.ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// Code with the caller as context and the caller is set to the caller of the caller.
func (ovm *OVM) DelegateCall(ctx *vmtypes.Context, caller vmtypes.ContractRef, addr types.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if ovm.OVMConfigs.NoRecursion && ovm.Depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, gas, vmtypes.ErrDepth
	}

	var (
		snapshot = ovm.StateDB.Snapshot()
		to       = vmtypes.AccountRef(caller.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := vmtypes.NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, ovm.StateDB.GetCodeHash(addr), ovm.StateDB.GetCode(addr))

	ret, err = run(ovm, contract, input, false)
	if err != nil {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != vmtypes.ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (ovm *OVM) StaticCall(ctx *vmtypes.Context, caller vmtypes.ContractRef, addr types.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if ovm.OVMConfigs.NoRecursion && ovm.Depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, gas, vmtypes.ErrDepth
	}

	var (
		to       = vmtypes.AccountRef(addr)
		snapshot = ovm.StateDB.Snapshot()
	)
	// Initialise a new contract and set the Code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := vmtypes.NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, ovm.StateDB.GetCodeHash(addr), ovm.StateDB.GetCode(addr))

	// When an error was returned by the OVM or when setting the creation Code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for Code storage gas errors.
	ret, err = run(ovm, contract, input, true)
	if err != nil {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != vmtypes.ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// create creates a new contract using Code as deployment Code.
func (ovm *OVM) create(ctx *vmtypes.Context, caller vmtypes.ContractRef, codeAndHash *vmtypes.CodeAndHash, gas uint64, value *big.Int, address types.Address) ([]byte, types.Address, uint64, error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, types.Address{}, gas, vmtypes.ErrDepth
	}
	if !ovm.CanTransfer(ovm.StateDB, caller.Address(), value) {
		return nil, types.Address{}, gas, vmtypes.ErrInsufficientBalance
	}
	nonce := ovm.StateDB.GetNonce(caller.Address())
	ovm.StateDB.SetNonce(caller.Address(), nonce+1)

	// Ensure there's no existing contract already at the designated address
	contractHash := ovm.StateDB.GetCodeHash(address)
	if ovm.StateDB.GetNonce(address) != 0 || (contractHash != (types.Hash{}) && contractHash != emptyCodeHash) {
		return nil, types.Address{}, 0, vmtypes.ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := ovm.StateDB.Snapshot()
	ovm.StateDB.CreateAccount(address)
	ovm.StateDB.SetNonce(address, 1)

	ovm.Transfer(ovm.StateDB, caller.Address(), address, value)

	// initialise a new contract and set the Code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := vmtypes.NewContract(caller, vmtypes.AccountRef(address), value, gas)
	contract.SetCodeOptionalHash(&address, codeAndHash)

	if ovm.OVMConfigs.NoRecursion && ovm.Depth > 0 {
		return nil, address, gas, nil
	}

	//if ovm.InterpreterConfig.Debug && ovm.Depth == 0 {
	//	ovm.InterpreterConfig.Tracer.CaptureStart(caller.Address(), address, true, codeAndHash.Code, gas, value)
	//}
	//start := time.Now()

	ret, err := run(ovm, contract, nil, false)

	// check whether the max Code size has been exceeded
	maxCodeSizeExceeded := len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the Code. If the Code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			ovm.StateDB.SetCode(address, ret)
		} else {
			err = vmtypes.ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the OVM or when setting the creation Code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for Code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && err != vmtypes.ErrCodeStoreOutOfGas) {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != vmtypes.ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract Code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = vmtypes.ErrMaxCodeSizeExceeded
	}
	//if ovm.InterpreterConfig.Debug && ovm.Depth == 0 {
	//	ovm.InterpreterConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	//}
	return ret, address, contract.Gas, err

}

// Create creates a new contract using Code as deployment Code.
func (ovm *OVM) Create(ctx *vmtypes.Context, caller vmtypes.ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr types.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), ovm.StateDB.GetNonce(caller.Address()))
	return ovm.create(ctx, caller, &vmtypes.CodeAndHash{Code: code}, gas, value, contractAddr)
}

// Create2 creates a new contract using Code as deployment Code.
//
// The different between Create2 with Create is Create2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// instead of the usual sender-and-Nonce-hash as the address where the contract is initialized at.
func (ovm *OVM) Create2(ctx *vmtypes.Context, caller vmtypes.ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, contractAddr types.Address, leftOverGas uint64, err error) {
	codeAndHash := &vmtypes.CodeAndHash{Code: code}
	contractAddr = crypto.CreateAddress2(caller.Address(), types.BigToHash(salt).Bytes, codeAndHash.Hash().ToBytes())
	return ovm.create(ctx, caller, codeAndHash, gas, endowment, contractAddr)
}
