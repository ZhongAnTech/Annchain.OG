package vm_test

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/eth/common/hexutil"
	"github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/ovm"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestContractSmallStorage(t *testing.T) {
	from := types.HexToAddress("0x01")

	txContext := &ovm.TxContext{
		From: types.HexToAddress("0x01"),
		//To:       types.HexToAddress("0x02"),
		Value:      math.NewBigInt(0),
		Data:       readFile("OwnedToken.bin"),
		GasPrice:   math.NewBigInt(1),
		GasLimit:   DefaultGasLimit,
		Coinbase:   types.HexToAddress("0x01"),
		SequenceID: 0,
	}
	coinBase := types.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	ldb := DefaultLDB(from, coinBase)

	logrus.Info("Init accounts done")

	context := ovm.NewEVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb)

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})

	evmInterpreter := vm.NewEVMInterpreter(context, txContext, &vm.InterpreterConfig{
		Debug:  true,
		Tracer: tracer,
	})

	ovm := ovm.NewOVM(context, []ovm.Interpreter{evmInterpreter}, &ovm.OVMConfig{NoRecursion: false})

	logrus.Info("Deploying contract")
	ret, contractAddr, leftOverGas, err := ovm.Create(vmtypes.AccountRef(txContext.From), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	// make duplicate
	//ovm.StateDB.SetNonce(coinBase, 0)
	//ret, contractAddr, leftOverGas, err = ovm.Create(&context, vmtypes.AccountRef(coinBase), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	logrus.Info("Deployed contract")
	fmt.Println("CP1", common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)
	fmt.Println(ldb.String())
	vm.WriteTrace(os.Stdout, tracer.Logs)
	assert.NoError(t, err)

	txContext.Value = math.NewBigInt(0)

	logrus.Info("Calling contract")

	var name [32]byte
	copy(name[:], "abcdefghijklmnopqrstuvwxyz")

	var input []byte
	contractAddress, err := hexutil.Decode("0x898855ed")
	assert.NoError(t, err)
	input = append(input, contractAddress...)
	input = append(input, name[:]...)

	ret, leftOverGas, err = ovm.Call(vmtypes.AccountRef(txContext.From), contractAddr, input, txContext.GasLimit, txContext.Value.Value)
	logrus.Info("Called contract")
	fmt.Println("CP2", common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)
	fmt.Println(ldb.String())
	vm.WriteTrace(os.Stdout, tracer.Logs)
	assert.NoError(t, err)
}

func TestContractHelloWorld(t *testing.T) {
	from := types.HexToAddress("0x01")
	coinBase := types.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})

	ldb := DefaultLDB(from, coinBase)

	rt := &Runtime{
		Tracer:    tracer,
		VmContext: ovm.NewEVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb),
		TxContext: &ovm.TxContext{
			From: types.HexToAddress("0x01"),
			//To:       types.HexToAddress("0x02"),
			Value:      math.NewBigInt(0),
			Data:       readFile("hello.bin"),
			GasPrice:   math.NewBigInt(1),
			GasLimit:   DefaultGasLimit,
			Coinbase:   coinBase,
			SequenceID: 0,
		},
	}

	_, contractAddr, _, err := DeployContract("hello.bin", from, coinBase, rt, nil)
	assert.NoError(t, err)

	value := math.NewBigInt(0)

	_, _, err = CallContract(contractAddr, from, coinBase, rt, value, "898855ed", []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	assert.NoError(t, err)

}
