package ovm

import (
	"math/big"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/annchain/OG/vm/runtime"
)


type VM interface {
	// Cancel cancels any running VM operation. This may be called concurrently and
	// it's safe to be called multiple times.
	Cancel()

	// Interpreter returns the current interpreter
	Interpreter() runtime.Interpreter

	// Call executes the vmtypes.Contract associated with the addr with the given input as
	// parameters. It also handles any necessary Value transfer required and takes
	// the necessary steps to create accounts and reverses the state in case of an
	// execution error or failed Value transfer.
	Call(caller vmtypes.ContractRef, addr types.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error)

	// CallCode executes the vmtypes.Contract associated with the addr with the given input
	// as parameters. It also handles any necessary Value transfer required and takes
	// the necessary steps to create accounts and reverses the state in case of an
	// execution error or failed Value transfer.
	//
	// CallCode differs from Call in the sense that it executes the given address'
	// code with the caller as context.
	CallCode(caller vmtypes.ContractRef, addr types.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error)

	// DelegateCall executes the vmtypes.Contract associated with the addr with the given input
	// as parameters. It reverses the state in case of an execution error.
	//
	// DelegateCall differs from CallCode in the sense that it executes the given address'
	// code with the caller as context and the caller is set to the caller of the caller.
	DelegateCall(caller vmtypes.ContractRef, addr types.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error)

	// StaticCall executes the vmtypes.Contract associated with the addr with the given input
	// as parameters while disallowing any modifications to the state during the call.
	// Opcodes that attempt to perform such modifications will result in exceptions
	// instead of performing the modifications.
	StaticCall(caller vmtypes.ContractRef, addr types.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error)

	// Create creates a new vmtypes.Contract using code as deployment code.
	Create(caller vmtypes.ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, ContractAddr types.Address, leftOverGas uint64, err error)

	// Create2 creates a new vmtypes.Contract using code as deployment code.
	//
	// The different between Create2 with Create is Create2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
	// instead of the usual sender-and-nonce-hash as the address where the vmtypes.Contract is initialized at.
	Create2(caller vmtypes.ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, ContractAddr types.Address, leftOverGas uint64, err error)
}
