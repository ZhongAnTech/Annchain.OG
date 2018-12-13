package vm_test

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/vm/ovm"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/annchain/OG/common/math"
	"math/big"
	"github.com/annchain/OG/vm/eth/core/vm"
)

func ExampleExecute() {

	txContext := &ovm.TxContext{
		From:     types.HexToAddress("0x01"),
		To:       types.HexToAddress("0x02"),
		Value:    math.NewBigInt(10),
		Data:     common.Hex2Bytes("6060604052600a8060106000396000f360606040526008565b00"),
		GasPrice: math.NewBigInt(10000),
		GasLimit: DefaultGasLimit,
	}
	coinBase := types.HexToAddress("0x03")


	db := ovm.NewMemoryStateDB()
	db.CreateAccount(txContext.From)
	db.AddBalance(txContext.From, big.NewInt(100000))
	db.CreateAccount(coinBase)
	db.AddBalance(coinBase, big.NewInt(100000))

	context := ovm.NewEVMContext(txContext, &ovm.DefaultChainContext{}, &coinBase, db, nil)

	evmInterpreter := vm.NewEVMInterpreter(&context, &vm.InterpreterConfig{})

	ovm := ovm.NewOVM(context, []ovm.Interpreter{evmInterpreter}, &ovm.OVMConfig{NoRecursion: false})

	ret, contractAddr, leftOverGas, err := ovm.Create(&context, vmtypes.AccountRef(txContext.From), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)

	ret, leftOverGas, err = ovm.Call(&context, vmtypes.AccountRef(txContext.From), contractAddr, txContext.Data, txContext.GasLimit, txContext.Value.Value)
	fmt.Println(common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)

	fmt.Println(db.String())
}
