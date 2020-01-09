package vm_test

import (
	"fmt"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/ovm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCall(t *testing.T) {
	from := common.HexToAddress("0xABCDEF88")
	from2 := common.HexToAddress("0xABCDEF87")
	coinBase := common.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})
	ldb := DefaultLDB(from, coinBase)
	ldb.CreateAccount(from2)
	ldb.AddBalance(from2, math.NewBigInt(10000000))

	rt := &Runtime{
		Tracer:    tracer,
		VmContext: ovm.NewOVMContext(&ovm.DefaultChainContext{}, &coinBase, ldb),
		TxContext: &ovm.TxContext{
			From:       common.HexToAddress("0xABCDEF88"),
			Value:      math.NewBigInt(0),
			GasPrice:   math.NewBigInt(1),
			GasLimit:   DefaultGasLimit,
			Coinbase:   coinBase,
			SequenceID: 0,
		},
	}

	addrs := make(map[string]common.Address)

	for _, filename := range []string{"Callee.bin", "Caller.bin"} {
		_, contractAddr, leftGas, err := DeployContract(filename, from, coinBase, rt, nil)
		assert.NoError(t, err)
		addrs[filename] = contractAddr
		dump(t, ldb, nil, leftGas, err)

	}

	logrus.Info("delegatecall")
	{
		params := EncodeParams([]interface{}{
			addrs["Callee.bin"],
			0xD0D0,
		})
		ret, leftGas, err := CallContract(addrs["Caller.bin"], from, coinBase, rt, math.NewBigInt(0), "9207ba9a", params)
		dump(t, ldb, ret, leftGas, err)
	}
	//vm.WriteTrace(os.Stdout, tracer.Logs)

	rt.Tracer = vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})

	logrus.Info("call")
	{
		params := EncodeParams([]interface{}{
			addrs["Callee.bin"],
			0xB0B0,
		})
		ret, leftGas, err := CallContract(addrs["Caller.bin"], from, coinBase, rt, math.NewBigInt(0), "fdfa868f", params)
		dump(t, ldb, ret, leftGas, err)
	}
	time.Sleep(time.Second * 3)
	//rt.Tracer.Write(os.Stdout)

	//logrus.Info("callcode")
	////vm.WriteTrace(os.Stdout, tracer.Logs)
	//{
	//	params := EncodeParams([]interface{}{
	//		addrs["Callee.bin"],
	//		0xC0C0,
	//	})
	//	ret, leftGas, err := CallContract(addrs["Caller.bin"], from2, coinBase, rt, math.NewBigInt(0), "0eebdd95", params)
	//	dump(t, ldb, ret, leftGas, err)
	//}

	//vm.WriteTrace(os.Stdout, tracer.Logs)

	ldb.MergeChanges()
	fmt.Println("Merged")
	fmt.Println(ldb.String())

}
