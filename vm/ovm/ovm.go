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
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	ogcrypto2 "github.com/annchain/OG/deprecated/ogcrypto"
	"math/big"
	"sync/atomic"

	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/vm/eth/params"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/sirupsen/logrus"
)

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var emptyCodeHash = ogcrypto2.Keccak256Hash(nil)

// run runs the given contract and takes care of running precompiles with a fallback to the byte Code interpreter.
func run(ovm *OVM, contract *vmtypes.Contract, input []byte, readOnly bool) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContractsByzantium
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	for _, interpreter := range ovm.Interpreters {
		if interpreter.CanRun(contract.Code) {
			if ovm.Interpreter != interpreter {
				// Ensure that the interpreter pointer is set back
				// to its current value upon return.
				defer func(i Interpreter) {
					ovm.Interpreter = i
				}(ovm.Interpreter)
				ovm.Interpreter = interpreter
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
	VMContext *vmtypes.Context
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
func NewOVM(ctx *vmtypes.Context, supportInterpreters []Interpreter, ovmConfig *OVMConfig) *OVM {
	ovm := &OVM{
		VMContext:  ctx,
		OVMConfigs: ovmConfig,
		//chainRules:   chainConfig.Rules(ctx.SequenceID),
		Interpreters: supportInterpreters,
	}
	if ovm.Interpreters != nil && len(ovm.Interpreters) > 0 {
		// set callers
		for _, interpreter := range ovm.Interpreters {
			interpreter.SetCaller(ovm)
		}

		ovm.Interpreter = ovm.Interpreters[0]
	}
	return ovm
}

// Cancel cancels any running OVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (ovm *OVM) Cancel() {
	atomic.StoreInt32(&ovm.VMContext.Abort, 1)
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (ovm *OVM) Call(caller vmtypes.ContractRef, addr ogTypes.Address20, input []byte, gas uint64, value *big.Int, txCall bool) (ret []byte, leftOverGas uint64, err error) {
	logrus.WithFields(logrus.Fields{
		"caller": caller.Address().Hex(),
		"addr":   addr.Hex(),
		"input":  hexutil.Encode(input),
		"gas":    gas,
		"value":  value,
	}).Info("It is calling")
	ctx := ovm.VMContext
	if ovm.OVMConfigs.NoRecursion && ctx.Depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if ctx.Depth > int(params.CallCreateDepth) {
		return nil, gas, vmtypes.ErrDepth
	}
	// Fail if we're trying to transfer more than the available Balance
	if !ctx.CanTransfer(ctx.StateDB, caller.Address(), value) {
		return nil, gas, vmtypes.ErrInsufficientBalance
	}

	var (
		to       = vmtypes.AccountRef(addr)
		snapshot = ctx.StateDB.Snapshot()
	)
	if !ctx.StateDB.Exist(&addr) {
		precompiles := PrecompiledContractsByzantium
		if precompiles[addr] == nil && value.Sign() == 0 {
			// Calling a non existing account, don't do anything, but ping the tracer
			//if ovm.InterpreterConfig.Debug && ovm.Depth == 0 {
			//	ovm.InterpreterConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
			//	ovm.InterpreterConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
			//}
			return nil, gas, nil
		}
		ctx.StateDB.CreateAccount(&addr)
	}
	if value.Sign() != 0 && !txCall {

		ctx.Transfer(ctx.StateDB, caller.Address(), to.Address(), value)
	}

	// Initialise a new contract and set the Code that is to be used by the OVM.
	// The contract is a scoped environment for this execution context only.
	contract := vmtypes.NewContract(caller, to, value, gas)
	codeHash := ctx.StateDB.GetCodeHash(&addr).(*ogTypes.Hash32)
	contract.SetCallCode(addr, *codeHash, ctx.StateDB.GetCode(&addr))

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
		ctx.StateDB.RevertToSnapshot(snapshot)
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
func (ovm *OVM) CallCode(caller vmtypes.ContractRef, addr ogTypes.Address20, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	logrus.WithFields(logrus.Fields{
		"caller": caller.Address().Hex(),
		"addr":   addr.Hex(),
		"input":  hexutil.Encode(input),
		"gas":    gas,
		"value":  value,
	}).Info("It is code calling")

	ctx := ovm.VMContext
	if ovm.OVMConfigs.NoRecursion && ctx.Depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if ctx.Depth > int(params.CallCreateDepth) {
		return nil, gas, vmtypes.ErrDepth
	}
	// Fail if we're trying to transfer more than the available Balance
	if !ctx.CanTransfer(ctx.StateDB, caller.Address(), value) {
		return nil, gas, vmtypes.ErrInsufficientBalance
	}

	var (
		snapshot = ctx.StateDB.Snapshot()
		to       = vmtypes.AccountRef(caller.Address())
	)
	// initialise a new contract and set the Code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := vmtypes.NewContract(caller, to, value, gas)
	callCodeHash := ctx.StateDB.GetCodeHash(&addr).(*ogTypes.Hash32)
	contract.SetCallCode(addr, *callCodeHash, ctx.StateDB.GetCode(&addr))

	ret, err = run(ovm, contract, input, false)
	if err != nil {
		ctx.StateDB.RevertToSnapshot(snapshot)
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
func (ovm *OVM) DelegateCall(caller vmtypes.ContractRef, addr ogTypes.Address20, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	logrus.WithFields(logrus.Fields{
		"caller": caller.Address().Hex(),
		"addr":   addr.Hex(),
		"input":  hexutil.Encode(input),
		"gas":    gas,
		//"value": value,
	}).Info("It is delegate calling")
	ctx := ovm.VMContext
	if ovm.OVMConfigs.NoRecursion && ctx.Depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if ctx.Depth > int(params.CallCreateDepth) {
		return nil, gas, vmtypes.ErrDepth
	}

	var (
		snapshot = ctx.StateDB.Snapshot()
		to       = vmtypes.AccountRef(caller.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := vmtypes.NewContract(caller, to, nil, gas).AsDelegate()
	callCodeHash := ctx.StateDB.GetCodeHash(&addr).(*ogTypes.Hash32)
	contract.SetCallCode(addr, *callCodeHash, ctx.StateDB.GetCode(&addr))

	ret, err = run(ovm, contract, input, false)
	if err != nil {
		ctx.StateDB.RevertToSnapshot(snapshot)
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
func (ovm *OVM) StaticCall(caller vmtypes.ContractRef, addr ogTypes.Address20, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	logrus.WithFields(logrus.Fields{
		"caller": caller.Address().Hex(),
		"addr":   addr.Hex(),
		"input":  hexutil.Encode(input),
		"gas":    gas,
		//"value": value,
	}).Info("It is static calling")
	ctx := ovm.VMContext
	if ovm.OVMConfigs.NoRecursion && ctx.Depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if ctx.Depth > int(params.CallCreateDepth) {
		return nil, gas, vmtypes.ErrDepth
	}

	var (
		to       = vmtypes.AccountRef(addr)
		snapshot = ctx.StateDB.Snapshot()
	)
	// Initialise a new contract and set the Code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := vmtypes.NewContract(caller, to, new(big.Int), gas)
	callCodeHash := ctx.StateDB.GetCodeHash(&addr).(*ogTypes.Hash32)
	contract.SetCallCode(addr, *callCodeHash, ctx.StateDB.GetCode(&addr))

	// When an error was returned by the OVM or when setting the creation Code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for Code storage gas errors.
	ret, err = run(ovm, contract, input, true)
	if err != nil {
		ctx.StateDB.RevertToSnapshot(snapshot)
		if err != vmtypes.ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// create creates a new contract using Code as deployment Code.
func (ovm *OVM) create(caller vmtypes.ContractRef, codeAndHash *vmtypes.CodeAndHash, gas uint64, value *big.Int, address ogTypes.Address20, txCall bool) ([]byte, ogTypes.Address20, uint64, error) {
	ctx := ovm.VMContext
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if ctx.Depth > int(params.CallCreateDepth) {
		return nil, ogTypes.Address20{}, gas, vmtypes.ErrDepth
	}
	if !ctx.CanTransfer(ctx.StateDB, caller.Address(), value) {
		return nil, ogTypes.Address20{}, gas, vmtypes.ErrInsufficientBalance
	}
	callerAddr := caller.Address()
	nonce := ctx.StateDB.GetNonce(&callerAddr)
	if !txCall {
		ctx.StateDB.SetNonce(&callerAddr, nonce+1)
	}

	// Ensure there's no existing contract already at the designated address
	contractHash := ctx.StateDB.GetCodeHash(&address).(*ogTypes.Hash32)
	if ctx.StateDB.GetNonce(&address) != 0 || (*contractHash != (ogTypes.Hash32{}) && contractHash != emptyCodeHash) {
		return nil, ogTypes.Address20{}, 0, vmtypes.ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := ctx.StateDB.Snapshot()
	ctx.StateDB.CreateAccount(&address)
	ctx.StateDB.SetNonce(&address, 1)

	if value.Sign() != 0 && !txCall {
		ctx.Transfer(ctx.StateDB, caller.Address(), address, value)
	}

	// initialise a new contract and set the Code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := vmtypes.NewContract(caller, vmtypes.AccountRef(address), value, gas)
	contract.SetCodeOptionalHash(address, codeAndHash)

	if ovm.OVMConfigs.NoRecursion && ctx.Depth > 0 {
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
			ctx.StateDB.SetCode(&address, ret)
		} else {
			err = vmtypes.ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the OVM or when setting the creation Code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for Code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && err != vmtypes.ErrCodeStoreOutOfGas) {
		ctx.StateDB.RevertToSnapshot(snapshot)
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
func (ovm *OVM) Create(caller vmtypes.ContractRef, code []byte, gas uint64, value *big.Int, txCall bool) (ret []byte, contractAddr ogTypes.Address20, leftOverGas uint64, err error) {
	callerAddr := caller.Address()
	contractAddr = ogTypes.CreateAddress20(caller.Address(), ovm.VMContext.StateDB.GetNonce(&callerAddr))
	return ovm.create(caller, &vmtypes.CodeAndHash{Code: code}, gas, value, contractAddr, txCall)
}

// Create2 creates a new contract using Code as deployment Code.
//
// The different between Create2 with Create is Create2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// instead of the usual sender-and-Nonce-hash as the address where the contract is initialized at.
func (ovm *OVM) Create2(caller vmtypes.ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int, txCall bool) (ret []byte, contractAddr ogTypes.Address20, leftOverGas uint64, err error) {
	codeAndHash := &vmtypes.CodeAndHash{Code: code}

	saltHash32 := ogTypes.BigToHash32(salt)
	contractAddr = ogTypes.CreateAddress20_2(caller.Address(), *saltHash32, codeAndHash.Hash().Bytes())
	return ovm.create(caller, codeAndHash, gas, endowment, contractAddr, txCall)
}
