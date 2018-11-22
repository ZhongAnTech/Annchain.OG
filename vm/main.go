package main

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/vm/ovm"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/vm/eth/params"
)

func ExampleExecute() {

	txContext := &ovm.TxContext{
		From:     types.HexToAddress("0x01"),
		To:       types.HexToAddress("0x02"),
		Value:    math.NewBigInt(10),
		Data:     common.Hex2Bytes("6060604052600a8060106000396000f360606040526008565b00"),
		GasPrice: math.NewBigInt(10000),
		GasLimit: 600000,
	}
	coinBase := types.HexToAddress("0x03")

	context := ovm.NewEVMContext(txContext, &ovm.DefaultChainContext{}, &coinBase)
	db := &ovm.MemoryStateDB{}

	ovm := ovm.NewOVM(context, db, &params.ChainConfig{ChainID: 0}, ovm.Config{})

	ret, contractAddr, leftOverGas, err := ovm.Create(vmtypes.AccountRef(coinBase), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(ret, contractAddr, leftOverGas, err)

	ret, leftOverGas, err = ovm.Call(vmtypes.AccountRef(coinBase), contractAddr, txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(ret, contractAddr, leftOverGas, err)
}

// loads a solidity file and run it
func main() {
	ExampleExecute()
}
