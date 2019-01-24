package vm_test

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/ovm"
	vmtypes "github.com/annchain/OG/vm/types"
)

func ExampleExecute() {

	txContext := &ovm.TxContext{
		From:       types.HexToAddress("0x01"),
		To:         types.HexToAddress("0x02"),
		Value:      math.NewBigInt(10),
		Data:       common.Hex2Bytes("6060604052600a8060106000396000f360606040526008565b00"),
		GasPrice:   math.NewBigInt(10000),
		GasLimit:   DefaultGasLimit,
		Coinbase:   types.HexToAddress("0x01"),
		SequenceID: 0,
	}
	coinBase := types.HexToAddress("0x03")

	db := ovm.NewMemoryStateDB()
	db.CreateAccount(txContext.From)
	db.AddBalance(txContext.From, math.NewBigInt(10000000))
	db.CreateAccount(coinBase)
	db.AddBalance(coinBase, math.NewBigInt(10000000))

	context := ovm.NewOVMContext(&ovm.DefaultChainContext{}, &coinBase, db)

	evmInterpreter := vm.NewEVMInterpreter(context, txContext, &vm.InterpreterConfig{})

	ovm := ovm.NewOVM(context, []ovm.Interpreter{evmInterpreter}, &ovm.OVMConfig{NoRecursion: false})

	ret, contractAddr, leftOverGas, err := ovm.Create(vmtypes.AccountRef(txContext.From), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)

	ret, leftOverGas, err = ovm.Call(vmtypes.AccountRef(txContext.From), contractAddr, txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)

	fmt.Println(db.String())
}
