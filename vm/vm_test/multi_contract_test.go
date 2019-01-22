package vm_test

import (
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/ovm"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestMultiContract(t *testing.T) {
	from := types.HexToAddress("0xABCDEF88")
	coinBase := types.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})
	ldb := DefaultLDB(from, coinBase)

	contracts := make(map[string]types.Address)

	//for _, file := range ([]string{"ABBToken", "owned", "SafeMath", "TokenCreator", "TokenERC20"}) {
	for _, file := range []string{"ABBToken"} {
		var params []byte

		switch file {
		case "ABBToken":
			params = EncodeParams([]interface{}{1, "AAAAAA", "ZZZZZZ"})
			fmt.Println(hex.Dump(params))
		}

		rt := &Runtime{
			Tracer:    tracer,
			VmContext: ovm.NewOVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb),
			TxContext: &ovm.TxContext{
				From: types.HexToAddress("0xABCDEF88"),
				//To:       types.HexToAddress("0x02"),
				Value:      math.NewBigInt(0),
				Data:       readFile(file + ".bin"),
				GasPrice:   math.NewBigInt(1),
				GasLimit:   DefaultGasLimit,
				Coinbase:   coinBase,
				SequenceID: 0,
			},
		}
		_, contractAddr, _, err := DeployContract(file+".bin", from, coinBase, rt, params)
		assert.NoError(t, err)
		contracts[file] = contractAddr

		switch file {
		case "ABBToken":
			fmt.Println(ldb.String())
			//vm.WriteTrace(os.Stdout, tracer.Logs)
		}

	}
	rt := &Runtime{
		Tracer:    tracer,
		VmContext: ovm.NewOVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb),
		TxContext: &ovm.TxContext{
			From:       types.HexToAddress("0xABCDEF88"),
			To:         contracts["TokenERC20"],
			Value:      math.NewBigInt(0),
			GasPrice:   math.NewBigInt(1),
			GasLimit:   DefaultGasLimit,
			Coinbase:   coinBase,
			SequenceID: 0,
		},
	}

	// query symbol name
	//ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "95d89b41", nil)
	//ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "18160ddd", nil)
	//ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "8da5cb5b", nil)
	// get balance
	//params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
	//ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "70a08231", params)

	//symbol
	{
		ret, leftGas, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "95d89b41", nil)
		dump(t, ldb, ret, leftGas, err)
	}
	//totalsupply
	{
		ret, leftGas, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "18160ddd", nil)
		dump(t, ldb, ret, leftGas, err)
	}
	//owner
	{
		ret, leftGas, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "8da5cb5b", nil)
		dump(t, ldb, ret, leftGas, err)
	}
	//frozenAccount
	//{
	//	params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
	//	ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "b414d4b6", params)
	//	dump(t, ldb, ret, err)
	//}
	//freezeAccount
	//{
	//	params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88"), true})
	//	ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "e724529c", params)
	//	dump(t, ldb, ret, err)
	//}
	////frozenAccount
	//{
	//	params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
	//	ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "b414d4b6", params)
	//	dump(t, ldb, ret, err)
	//}
	//burn
	//{
	//	params := EncodeParams([]interface{}{100000000})
	//	ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "42966c68", params)
	//	dump(t, ldb, ret, err)
	//}
	//burn2
	//{
	//	params := EncodeParams([]interface{}{100000000})
	//	ret, _, err := CallContract(contracts["ABBToken"], types.HexToAddress("0xDEADBEEF"), coinBase, rt, math.NewBigInt(0), "42966c68", params)
	//	dump(t, ldb, ret, err)
	//}
	// burn
	//{
	//	params := EncodeParams([]interface{}{100000000})
	//	ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "42966c68", params)
	//	dump(t, ldb, ret, err)
	//}
	// query balance
	{
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
		ret, leftGas, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "70a08231", params)
		dump(t, ldb, ret, leftGas, err)
	}
	// transfer
	{
		u64, err := strconv.ParseUint("0101", 16, 64)
		assert.NoError(t, err)
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF87"), u64})
		ret, leftGas, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "a9059cbb", params)
		dump(t, ldb, ret, leftGas, err)
	}
	// query again
	{
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
		ret, leftGas, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "70a08231", params)
		dump(t, ldb, ret, leftGas, err)
	}
}

func TestInterCall(t *testing.T) {
	from := types.HexToAddress("0xABCDEF88")
	coinBase := types.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})
	ldb := DefaultLDB(from, coinBase)

	contracts := make(map[string]types.Address)

	for _, file := range []string{"C1", "C2"} {

		var params []byte

		rt := &Runtime{
			Tracer:    tracer,
			VmContext: ovm.NewOVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb),
			TxContext: &ovm.TxContext{
				From:       from,
				Value:      math.NewBigInt(0),
				Data:       readFile(file + ".bin"),
				GasPrice:   math.NewBigInt(1),
				GasLimit:   DefaultGasLimit,
				Coinbase:   coinBase,
				SequenceID: 0,
			},
		}
		_, contractAddr, _, err := DeployContract(file+".bin", from, coinBase, rt, params)
		assert.NoError(t, err)
		contracts[file] = contractAddr
	}

	rt := &Runtime{
		Tracer:    tracer,
		VmContext: ovm.NewOVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb),
		TxContext: &ovm.TxContext{
			From:       types.HexToAddress("0xABCDEF88"),
			To:         contracts["TokenERC20"],
			Value:      math.NewBigInt(0),
			GasPrice:   math.NewBigInt(1),
			GasLimit:   DefaultGasLimit,
			Coinbase:   coinBase,
			SequenceID: 0,
		},
	}

	{
		ret, leftGas, err := CallContract(contracts["C1"], from, coinBase, rt, math.NewBigInt(0), "c27fc305", nil)
		dump(t, ldb, ret, leftGas, err)
	}
	{
		params := EncodeParams([]interface{}{contracts["C1"]})
		ret, leftGas, err := CallContract(contracts["C2"], from, coinBase, rt, math.NewBigInt(0), "c3642756", params)
		dump(t, ldb, ret, leftGas, err)
	}

}
