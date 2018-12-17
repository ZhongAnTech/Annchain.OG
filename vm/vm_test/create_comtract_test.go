package vm

import (
	"testing"
	"github.com/annchain/OG/types"
)

func TestContractCreation(t *testing.T) {
	from := types.HexToAddress("0xABCDEF88")
	coinBase := types.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")

	tracer := vm.NewStructLogger(&vm.LogConfig{
		Debug: true,
	})
	ldb := DefaultLDB(from, coinBase)
}