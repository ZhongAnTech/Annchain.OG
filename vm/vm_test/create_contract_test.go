package vm_test

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/ovm"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContractCreation(t *testing.T) {
	from := common.HexToAddress("0xABCDEF88")
	coinBase := common.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})
	ldb := DefaultLDB(from, coinBase)

	rt := &Runtime{
		Tracer:    tracer,
		VmContext: ovm.NewOVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb),
		TxContext: &ovm.TxContext{
			From:       common.HexToAddress("0xABCDEF88"),
			Value:      math.NewBigInt(0),
			GasPrice:   math.NewBigInt(1),
			GasLimit:   DefaultGasLimit,
			Coinbase:   coinBase,
			SequenceID: 0,
		},
	}

	_, contractAddr, _, err := DeployContract("C.bin", from, coinBase, rt, nil)
	assert.NoError(t, err)

	// Use C to create D
	{
		params := EncodeParams([]interface{}{55, 66})
		ret, leftGas, err := CallContract(contractAddr, from, coinBase, rt, math.NewBigInt(66), "8dcd64cc", params)
		dump(t, ldb, ret, leftGas, err)
	}

	//vm.WriteTrace(os.Stdout, tracer.Logs)
}
