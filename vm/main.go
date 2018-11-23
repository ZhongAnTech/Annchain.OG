package main

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/vm/ovm"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/annchain/OG/common/math"
	"math/big"
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
	db := ovm.NewMemoryStateDB()
	db.CreateAccount(txContext.From)
	db.AddBalance(txContext.From, big.NewInt(100000))
	db.CreateAccount(coinBase)
	db.AddBalance(coinBase, big.NewInt(100000))


	ovm := ovm.NewOVM(context, db, nil, &ovm.OVMConfig{NoRecursion: false})

	ret, contractAddr, leftOverGas, err := ovm.Create(&context, vmtypes.AccountRef(coinBase), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)

	ret, leftOverGas, err = ovm.Call(&context, vmtypes.AccountRef(coinBase), contractAddr, txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)

	fmt.Println(db.Dump())
}

// loads a solidity file and run it
func main() {
	ExampleExecute()
}
