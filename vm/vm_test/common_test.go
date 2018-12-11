package vm_test

import (
	"math/big"
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/vm/ovm"
	"github.com/annchain/OG/vm/eth/core/vm"
	"fmt"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"io/ioutil"
	"encoding/hex"
	"github.com/annchain/OG/common"
	vmtypes "github.com/annchain/OG/vm/types"
	"os"
	"bytes"
	"testing"
	"encoding/binary"
	"reflect"
)

func readFile(filename string) []byte {
	bytes, err := ioutil.ReadFile(Root + filename)
	if err != nil {
		panic(err)
	}
	bytes, err = hex.DecodeString(string(bytes))
	return bytes
}

type Runtime struct {
	Context vmtypes.Context
	Tracer  vm.Tracer
}

func DefaultLDB(from types.Address, coinBase types.Address) *ovm.LayerStateDB {
	mmdb := ovm.NewMemoryStateDB()
	ldb := ovm.NewLayerDB(mmdb)
	ldb.NewLayer()
	ldb.CreateAccount(from)
	ldb.AddBalance(from, big.NewInt(10000000))
	ldb.CreateAccount(coinBase)
	ldb.AddBalance(coinBase, big.NewInt(10000000))
	logrus.Info("Init accounts done")
	return ldb
}

func DefaultOVM(runtime *Runtime) *ovm.OVM {
	evmInterpreter := vm.NewEVMInterpreter(&runtime.Context, &vm.InterpreterConfig{
		Debug:  true,
		Tracer: runtime.Tracer,
	})

	return ovm.NewOVM(runtime.Context, []ovm.Interpreter{evmInterpreter}, &ovm.OVMConfig{NoRecursion: false})
}

func DeployContract(filename string, from types.Address, coinBase types.Address, rt *Runtime, params []byte) (ret []byte, contractAddr types.Address, leftOverGas uint64, err error) {
	txContext := &ovm.TxContext{
		From:     from,
		Value:    math.NewBigInt(0),
		Data:     readFile(filename),
		GasPrice: math.NewBigInt(1),
		GasLimit: DefaultGasLimit,
	}

	ovm := DefaultOVM(rt)
	if params != nil && len(params) != 0 {
		txContext.Data = append(txContext.Data, params...)
	}

	logrus.Info("Deploying contract")
	ret, contractAddr, leftOverGas, err = ovm.Create(&rt.Context, vmtypes.AccountRef(coinBase), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	// make duplicate
	//ovm.StateDB.SetNonce(coinBase, 0)
	//ret, contractAddr, leftOverGas, err = ovm.Create(&context, vmtypes.AccountRef(coinBase), txContext.Data, txContext.GasLimit, txContext.Value.Value)
	logrus.Info("Deployed contract")
	fmt.Println("CP1", common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)
	//fmt.Println(rt.Context.StateDB.String())
	//rt.Tracer.Write(os.Stdout)
	return
}

func CallContract(contractAddr types.Address, from types.Address, coinBase types.Address, rt *Runtime, value *math.BigInt, functionHash string, params []byte) (ret []byte, leftOverGas uint64, err error) {
	txContext := &ovm.TxContext{
		From:     from,
		Value:    value,
		GasPrice: math.NewBigInt(1),
		GasLimit: DefaultGasLimit,
	}

	logrus.Info("Calling contract")

	var input []byte
	contractAddress, err := hex.DecodeString(functionHash)
	if err != nil {
		return
	}
	input = append(input, contractAddress...)
	if params != nil && len(params) != 0 {
		input = append(input, params...)
	}

	ovm := DefaultOVM(rt)

	ret, leftOverGas, err = ovm.Call(&rt.Context, vmtypes.AccountRef(coinBase), contractAddr, input, txContext.GasLimit, txContext.Value.Value)
	logrus.Info("Called contract")
	fmt.Println("CP2", common.Bytes2Hex(ret), contractAddr.String(), leftOverGas, err)
	fmt.Println(rt.Context.StateDB.String())
	rt.Tracer.Write(os.Stdout)
	return
}

func pad(r []byte, baseLen int, padLeft bool) []byte {
	l := len(r)
	newl := baseLen
	if len(r) > baseLen {
		newl = ((l + baseLen - 1) / baseLen) * baseLen
	}
	bytes := make([]byte, newl)
	if padLeft {
		copy(bytes[newl-l:], r)
	} else {
		copy(bytes[0:], r)
	}

	return bytes
}

func EncodeParams(params []interface{}) []byte {
	//300000, test me   ,PPPPPPPPPP
	//00000000000000000000000000000000000000000000000000000000000493e0
	//0000000000000000000000000000000000000000000000000000000000000060
	//00000000000000000000000000000000000000000000000000000000000000a0
	//000000000000000000000000000000000000000000000000000000000000000a
	//74657374206d6520202000000000000000000000000000000000000000000000
	//000000000000000000000000000000000000000000000000000000000000000a
	//5050505050505050505000000000000000000000000000000000000000000000

	buf := bytes.Buffer{}
	pointer := len(params) * 32
	head := make([]byte, pointer)
	for i, obj := range params {
		var bs []byte
		switch obj.(type) {
		case int:
			bs = make([]byte, 4)
			binary.BigEndian.PutUint32(bs, uint32(obj.(int)))
		case uint:
			bs = make([]byte, 4)
			binary.BigEndian.PutUint32(bs, uint32(obj.(uint)))
		case bool:
			bs = make([]byte, 4)
			if obj.(bool) {
				binary.BigEndian.PutUint32(bs, 1)
			} else {
				binary.BigEndian.PutUint32(bs, 0)
			}
		case string:
			bs = make([]byte, 4)
			binary.BigEndian.PutUint32(bs, uint32(pointer))
			bsPayloadLength := make([]byte, 4)
			binary.BigEndian.PutUint32(bsPayloadLength, uint32(len(obj.(string))))
			buf.Write(pad(bsPayloadLength, 32, true))
			buf.Write(pad([]byte(obj.(string)), 32, false))
			pointer = buf.Len()
		default:
			panic(fmt.Sprintf("not supported: %s", reflect.TypeOf(obj).String()))
		}
		bsWrite := pad(bs, 32, true)
		copy(head[i*32:i*32+32], bsWrite)


	}
	result := bytes.Buffer{}
	result.Write(head)
	result.Write(buf.Bytes())

	return result.Bytes()
}

func TestPad(t *testing.T) {
	v, _ := hex.DecodeString("01")
	fmt.Printf("%x\n", pad(v, 4, true))
	v, _ = hex.DecodeString("0101")
	fmt.Printf("%x\n", pad(v, 4, true))
	v, _ = hex.DecodeString("010101")
	fmt.Printf("%x\n", pad(v, 4, true))
	v, _ = hex.DecodeString("01010101")
	fmt.Printf("%x\n", pad(v, 4, true))
	v, _ = hex.DecodeString("0101010101")
	fmt.Printf("%x\n", pad(v, 4, true))
	v, _ = hex.DecodeString("010101010101")
	fmt.Printf("%x\n", pad(v, 4, true))
	v, _ = hex.DecodeString("01010101010101")
	fmt.Printf("%x\n", pad(v, 4, true))
	v, _ = hex.DecodeString("0101010101010101")
	fmt.Printf("%x\n", pad(v, 4, true))
	v, _ = hex.DecodeString("010101010101010101")
	fmt.Printf("%x\n", pad(v, 4, true))

}

func TestEncodeParams(t *testing.T) {
	params := []interface{}{1024, "TTTTTTTTTTT", "PPPPPPPPPP"}
	bs := EncodeParams(params)
	fmt.Println(hex.Dump(bs))
}
