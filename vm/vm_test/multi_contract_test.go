package vm_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/annchain/OG/common/math"
	"testing"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/ovm"
	"github.com/annchain/OG/vm/eth/core/vm"
	"encoding/hex"
	"strconv"
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
	for _, file := range ([]string{"ABBToken"}) {
		txContext := &ovm.TxContext{
			From: types.HexToAddress("0xABCDEF88"),
			//To:       types.HexToAddress("0x02"),
			Value:    math.NewBigInt(0),
			Data:     readFile(file + ".bin"),
			GasPrice: math.NewBigInt(1),
			GasLimit: DefaultGasLimit,
		}

		var params []byte

		switch file {
		case "ABBToken":
			params = EncodeParams([]interface{}{1, "AAAAAA", "ZZZZZZ"})
			fmt.Println(hex.Dump(params))
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
			//vm.WriteTrace(os.Stdout, tracer.Logs)
		}

	}

	txContext := &ovm.TxContext{
		From:     types.HexToAddress("0xABCDEF88"),
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
	//ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "95d89b41", nil)
	//ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "18160ddd", nil)
	//ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "8da5cb5b", nil)
	// get balance
	//params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
	//ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "70a08231", params)


	{
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "95d89b41", nil)
		dump(t, ldb, ret, err)
	}
	{
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "18160ddd", nil)
		dump(t, ldb, ret, err)
	}
	{
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "8da5cb5b", nil)
		dump(t, ldb, ret, err)
	}
	{
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "b414d4b6", params)
		dump(t, ldb, ret, err)
	}
	{
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88"), true})
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "e724529c", params)
		dump(t, ldb, ret, err)
	}

	{
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "b414d4b6", params)
		dump(t, ldb, ret, err)
	}

	{
		params := EncodeParams([]interface{}{100000000})
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "42966c68", params)
		dump(t, ldb, ret, err)
	}
	//{
	//	params := EncodeParams([]interface{}{100000000})
	//	ret, _, err := CallContract(contracts["ABBToken"], types.HexToAddress("0xDEADBEEF"), coinBase, rt, math.NewBigInt(0), "42966c68", params)
	//	dump(t, ldb, ret, err)
	//}
	{
		params := EncodeParams([]interface{}{100000000})
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "42966c68", params)
		dump(t, ldb, ret, err)
	}
	// query balance
	{
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "70a08231", params)
		dump(t, ldb, ret, err)
	}

	// transfer
	{
		u64, err := strconv.ParseUint("0de0b6b39b783e00", 16, 64)
		assert.NoError(t, err)
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF87"), u64})
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "a9059cbb", params)
		dump(t, ldb, ret, err)
	}
	// query again
	{
		params := EncodeParams([]interface{}{types.HexToAddress("0xABCDEF88")})
		ret, _, err := CallContract(contracts["ABBToken"], from, coinBase, rt, math.NewBigInt(0), "70a08231", params)
		dump(t, ldb, ret, err)
	}
}

func dump(t *testing.T, ldb *ovm.LayerStateDB, ret []byte, err error) {
	fmt.Println(ldb.String())
	//vm.WriteTrace(os.Stdout, tracer.Logs)
	fmt.Printf("Return value: [%s]\n", DecodeParamToString(ret))
	fmt.Printf("Return value: [%s]\n", DecodeParamToBigInt(ret))
	fmt.Printf("Return value: [%s]\n", DecodeParamToByteString(ret))
	assert.NoError(t, err)
}
