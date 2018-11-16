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
	"time"

	"github.com/annchain/OG/vm/eth/crypto"
	"github.com/annchain/OG/vm/eth/params"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/eth/core/vm"
)

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var emptyCodeHash = crypto.Keccak256Hash(nil)

type (
	// CanTransferFunc is the signature of a transfer guard function
	CanTransferFunc func(StateDB, types.Address, *big.Int) bool
	// TransferFunc is the signature of a transfer function
	TransferFunc func(StateDB, types.Address, types.Address, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH OVM op code.
	GetHashFunc func(uint64) types.Hash
)

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evm *OVM, contract *Contract, input []byte, readOnly bool) ([]byte, error) {
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
	return nil, ErrNoCompatibleInterpreter
}

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

	// Message information
	Origin   types.Address // Provides information for ORIGIN
	GasPrice *big.Int      // Provides information for GASPRICE

	// Block information
	Coinbase   types.Address // Provides information for COINBASE
	GasLimit   uint64        // Provides information for GASLIMIT
	SequenceID uint64        // Provides information for SequenceID
	//Time        *math.BigInt      // Provides information for TIME
	//Difficulty  *math.BigInt      // Provides information for DIFFICULTY
}

// OVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The OVM should never be reused and is not thread safe.
type OVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	Depth int

	// chainConfig contains information about the current chain
	ChainConfig *params.ChainConfig
	// chain rules contains the chain rules for the current epoch
	//chainRules params.Rules
	// virtual machine configuration options used to initialise the
	// evm.
	VmConfig Config
	// global (to this context) ethereum virtual machine
	// used throughout the execution of the tx.
	Interpreters []Interpreter
	Interpreter  Interpreter
	// abort is used to abort the OVM calling operations
	// NOTE: must be set atomically
	Abort int32
	// callGasTemp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	CallGasTemp uint64
}

// NewOVM returns a new OVM. The returned OVM is not thread safe and should
// only ever be used *once*.
func NewOVM(ctx Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config) *OVM {
	evm := &OVM{
		Context:     ctx,
		StateDB:     statedb,
		VmConfig:    vmConfig,
		ChainConfig: chainConfig,
		//chainRules:   chainConfig.Rules(ctx.SequenceID),
		Interpreters: make([]Interpreter, 0, 1),
	}

	// vmConfig.EVMInterpreter will be used by OVM-C, it won't be checked here
	// as we always want to have the built-in OVM as the failover option.
	evm.Interpreters = append(evm.Interpreters, vm.NewEVMInterpreter(evm, vmConfig))
	evm.Interpreter = evm.Interpreters[0]

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
func (ovm *OVM) Call(caller ContractRef, addr types.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if ovm.VmConfig.NoRecursion && ovm.Depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !ovm.Context.CanTransfer(ovm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		to       = AccountRef(addr)
		snapshot = ovm.StateDB.Snapshot()
	)
	if !ovm.StateDB.Exist(addr) {
		precompiles := PrecompiledContractsByzantium
		if precompiles[addr] == nil && value.Sign() == 0 {
			// Calling a non existing account, don't do anything, but ping the tracer
			if ovm.VmConfig.Debug && ovm.Depth == 0 {
				ovm.VmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
				ovm.VmConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
			}
			return nil, gas, nil
		}
		ovm.StateDB.CreateAccount(addr)
	}
	ovm.Transfer(ovm.StateDB, caller.Address(), to.Address(), value)
	// Initialise a new contract and set the code that is to be used by the OVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, ovm.StateDB.GetCodeHash(addr), ovm.StateDB.GetCode(addr))

	// Even if the account has no code, we need to continue because it might be a precompile
	start := time.Now()

	// Capture the tracer start/end events in debug mode
	if ovm.VmConfig.Debug && ovm.Depth == 0 {
		ovm.VmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)

		defer func() { // Lazy evaluation of the parameters
			ovm.VmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		}()
	}
	ret, err = run(ovm, contract, input, false)

	// When an error was returned by the OVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
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
// code with the caller as context.
func (ovm *OVM) CallCode(caller ContractRef, addr types.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if ovm.VmConfig.NoRecursion && ovm.Depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !ovm.CanTransfer(ovm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		snapshot = ovm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, ovm.StateDB.GetCodeHash(addr), ovm.StateDB.GetCode(addr))

	ret, err = run(ovm, contract, input, false)
	if err != nil {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (ovm *OVM) DelegateCall(caller ContractRef, addr types.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if ovm.VmConfig.NoRecursion && ovm.Depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		snapshot = ovm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, ovm.StateDB.GetCodeHash(addr), ovm.StateDB.GetCode(addr))

	ret, err = run(ovm, contract, input, false)
	if err != nil {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (ovm *OVM) StaticCall(caller ContractRef, addr types.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if ovm.VmConfig.NoRecursion && ovm.Depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		to       = AccountRef(addr)
		snapshot = ovm.StateDB.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, ovm.StateDB.GetCodeHash(addr), ovm.StateDB.GetCode(addr))

	// When an error was returned by the OVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(ovm, contract, input, true)
	if err != nil {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// create creates a new contract using code as deployment code.
func (ovm *OVM) create(caller ContractRef, codeAndHash *CodeAndHash, gas uint64, value *big.Int, address types.Address) ([]byte, types.Address, uint64, error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if ovm.Depth > int(params.CallCreateDepth) {
		return nil, types.Address{}, gas, ErrDepth
	}
	if !ovm.CanTransfer(ovm.StateDB, caller.Address(), value) {
		return nil, types.Address{}, gas, ErrInsufficientBalance
	}
	nonce := ovm.StateDB.GetNonce(caller.Address())
	ovm.StateDB.SetNonce(caller.Address(), nonce+1)

	// Ensure there's no existing contract already at the designated address
	contractHash := ovm.StateDB.GetCodeHash(address)
	if ovm.StateDB.GetNonce(address) != 0 || (contractHash != (types.Hash{}) && contractHash != emptyCodeHash) {
		return nil, types.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := ovm.StateDB.Snapshot()
	ovm.StateDB.CreateAccount(address)
	ovm.StateDB.SetNonce(address, 1)

	ovm.Transfer(ovm.StateDB, caller.Address(), address, value)

	// initialise a new contract and set the code that is to be used by the
	// OVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(address), value, gas)
	contract.SetCodeOptionalHash(&address, codeAndHash)

	if ovm.VmConfig.NoRecursion && ovm.Depth > 0 {
		return nil, address, gas, nil
	}

	if ovm.VmConfig.Debug && ovm.Depth == 0 {
		ovm.VmConfig.Tracer.CaptureStart(caller.Address(), address, true, codeAndHash.Code, gas, value)
	}
	start := time.Now()

	ret, err := run(ovm, contract, nil, false)

	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			ovm.StateDB.SetCode(address, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the OVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && err != ErrCodeStoreOutOfGas) {
		ovm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = ErrMaxCodeSizeExceeded
	}
	if ovm.VmConfig.Debug && ovm.Depth == 0 {
		ovm.VmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	}
	return ret, address, contract.Gas, err

}

// Create creates a new contract using code as deployment code.
func (ovm *OVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr types.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), ovm.StateDB.GetNonce(caller.Address()))
	return ovm.create(caller, &CodeAndHash{Code: code}, gas, value, contractAddr)
}

// Create2 creates a new contract using code as deployment code.
//
// The different between Create2 with Create is Create2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// instead of the usual sender-and-nonce-hash as the address where the contract is initialized at.
func (ovm *OVM) Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, contractAddr types.Address, leftOverGas uint64, err error) {
	codeAndHash := &CodeAndHash{Code: code}
	contractAddr = crypto.CreateAddress2(caller.Address(), types.BigToHash(salt).Bytes, codeAndHash.Hash().Bytes[:])
	return ovm.create(caller, codeAndHash, gas, endowment, contractAddr)
}
