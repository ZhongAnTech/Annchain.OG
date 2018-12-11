package vm_test

import (
	"github.com/sirupsen/logrus"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/annchain/OG/common/math"
	"testing"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/ovm"
	"os"
	"github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/common"
	"encoding/hex"
)

func TestMultiContract(t *testing.T) {
	from := types.HexToAddress("0x01")
	coinBase := types.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})
	ldb := DefaultLDB(from, coinBase)

	contracts := make(map[string]types.Address)

	for _, file := range ([]string{"ABBToken", "owned", "SafeMath", "TokenCreator", "TokenERC20"}) {
		txContext := &ovm.TxContext{
			From: types.HexToAddress("0x01"),
			//To:       types.HexToAddress("0x02"),
			Value:    math.NewBigInt(0),
			Data:     readFile(file + ".bin"),
			GasPrice: math.NewBigInt(1),
			GasLimit: DefaultGasLimit,
		}

		var params []byte

		switch file{
		case "ABBToken":
			p, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000493e000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000a7465737206d6520202000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a5050505050505050505000000000000000000000000000000000000000000000")
			assert.NoError(t, err)
			params = p
		}

		rt := &Runtime{
			Tracer:  tracer,
			Context: ovm.NewEVMContext(txContext, &ovm.DefaultChainContext{}, &coinBase, ldb),
		}
		_, contractAddr, _, err := DeployContract(file+".bin", from, coinBase, rt, params)
		assert.NoError(t, err)
		contracts[file] = contractAddr

		switch file {
		case "ABBToken":
			fmt.Println(ldb.String())
			vm.WriteTrace(os.Stdout, tracer.Logs)
		}

	}
	return

	txContext := &ovm.TxContext{
		From:     types.HexToAddress("0x01"),
		To:       contracts["TokenERC20"],
		Value:    math.NewBigInt(0),
		GasPrice: math.NewBigInt(1),
		GasLimit: DefaultGasLimit,
	}

	rt := &Runtime{
		Tracer:  tracer,
		Context: ovm.NewEVMContext(txContext, &ovm.DefaultChainContext{}, &coinBase, ldb),
	}

	// query symbol name
	ret, leftOverGas, err := CallContract(contracts["TokenERC20"], from, coinBase, rt, math.NewBigInt(0), "95d89b41", nil)
	// get balance
	// ret, leftOverGas, err := CallContract(contracts["TokenERC20"], from, coinBase, rt, math.NewBigInt(0), "70a08231", nil)

	logrus.Info("Called contract")
	fmt.Println(ldb.String())
	vm.WriteTrace(os.Stdout, tracer.Logs)
	fmt.Println("CP2", common.Bytes2Hex(ret), leftOverGas, err)

	assert.NoError(t, err)
}
