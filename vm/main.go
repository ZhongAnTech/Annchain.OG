package main

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/vmcommon"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/vm/eth/params"
)

func ExampleExecute() {

	txContext := &vmcommon.TxContext{
		From:     types.HexToAddress("0x01"),
		To:       types.HexToAddress("0x02"),
		Value:    math.NewBigInt(10),
		Data:     common.Hex2Bytes("6060604052600a8060106000396000f360606040526008565b00"),
		GasPrice: math.NewBigInt(10000),
		GasLimit: 600000,
	}
	coinBase := types.HexToAddress("0x03")

	context := vmcommon.NewEVMContext(txContext, &vmcommon.DefaultChainContext{}, &coinBase)
	db := &vmcommon.MemoryStateDB{}

	evm := vm.NewEVM(context, db, &params.ChainConfig{ChainID: 0}, vm.Config{})

	ret, contractAddr, leftOverGas, err := evm.Create(vmcommon.AccountRef(coinBase), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(ret, contractAddr, leftOverGas, err)

	ret, leftOverGas, err = evm.Call(vmcommon.AccountRef(coinBase), contractAddr, txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(ret, contractAddr, leftOverGas, err)
}

// loads a solidity file and run it
func main() {
	ExampleExecute()
}
