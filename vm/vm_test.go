package vm_test

import (
	"testing"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/vm/ovm"
	"github.com/annchain/OG/types"
	vmtypes "github.com/annchain/OG/vm/types"
	"github.com/annchain/OG/common/math"
	"math/big"
	"github.com/annchain/OG/vm/eth/core/vm"
	"fmt"
	"io/ioutil"
	"encoding/hex"
	"github.com/sirupsen/logrus"
)

func readFile(filename string) []byte{
	bytes, err := ioutil.ReadFile(filename)
	if err != nil{
		panic(err)
	}
	bytes, err = hex.DecodeString(string(bytes))
	return bytes
}

func TestSmallContract(t *testing.T) {
	txContext := &ovm.TxContext{
		From:     types.HexToAddress("0x01"),
		//To:       types.HexToAddress("0x02"),
		Value:    math.NewBigInt(10),
		Data:     readFile("./compiler/d/hello.bin"),
		GasPrice: math.NewBigInt(10000),
		GasLimit: 600000,
	}
	coinBase := types.HexToAddress("0x03")

	context := ovm.NewEVMContext(txContext, &ovm.DefaultChainContext{}, &coinBase)
	mmdb := ovm.NewMemoryStateDB()
	ldb := ovm.NewLayerDB(mmdb)
	ldb.NewLayer()
	ldb.CreateAccount(txContext.From)
	ldb.AddBalance(txContext.From, big.NewInt(100000))
	ldb.CreateAccount(coinBase)
	ldb.AddBalance(coinBase, big.NewInt(100000))
	logrus.Info("Init accounts done")

	evmInterpreter := vm.NewEVMInterpreter(&context, &vm.InterpreterConfig{})

	ovm := ovm.NewOVM(context, ldb, []ovm.Interpreter{evmInterpreter}, &ovm.OVMConfig{NoRecursion: false})

	logrus.Info("Deploying contract")
	ret, contractAddr, leftOverGas, err := ovm.Create(&context, vmtypes.AccountRef(coinBase), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	logrus.Info("Deployed contract")
	fmt.Println("CP1", common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)
	fmt.Println(ldb.String())

	logrus.Info("Calling contract")
	ret, leftOverGas, err = ovm.Call(&context, vmtypes.AccountRef(coinBase), contractAddr, nil, txContext.GasLimit, txContext.Value.Value)
	logrus.Info("Called contract")
	fmt.Println("CP2", common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)
	fmt.Println(ldb.String())
}