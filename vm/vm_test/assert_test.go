package vm_test

import (
	"testing"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/ovm"
	"github.com/annchain/OG/common/math"
	"github.com/stretchr/testify/assert"
	"github.com/annchain/OG/vm/eth/core/vm"
)

func TestAsserts(t *testing.T) {
	from := types.HexToAddress("0xABCDEF88")
	from2 := types.HexToAddress("0xABCDEF87")
	coinBase := types.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})
	ldb := DefaultLDB(from, coinBase)
	ldb.CreateAccount(from2)
	ldb.AddBalance(from2, math.NewBigInt(10000000))

	rt := &Runtime{
		Tracer:    tracer,
		VmContext: ovm.NewEVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb),
		TxContext: &ovm.TxContext{
			From:       types.HexToAddress("0xABCDEF88"),
			Value:      math.NewBigInt(0),
			GasPrice:   math.NewBigInt(1),
			GasLimit:   DefaultGasLimit,
			Coinbase:   coinBase,
			SequenceID: 0,
		},
	}

	_, contractAddr, _, err := DeployContract("asserts.bin", from, coinBase, rt, nil)
	assert.NoError(t, err)

	{
		// op jumps to 0xfe and then raise a non-existing op
		ret, leftGas, err := CallContract(contractAddr, from, coinBase, rt, math.NewBigInt(2), "2911e7b2", nil)
		dump(t, ldb, ret, leftGas, err)
	}
	//vm.WriteTrace(os.Stdout, tracer.Logs)
	{
		ret, leftGas, err := CallContract(contractAddr, from2, coinBase, rt, math.NewBigInt(2), "0d43aaf2", nil)
		dump(t, ldb, ret, leftGas, err)
	}
	//vm.WriteTrace(os.Stdout, tracer.Logs)
}