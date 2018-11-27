package ovm

import (
	"testing"
	"github.com/annchain/OG/types"
	"math/big"
	"fmt"
)

func TestLayers(t *testing.T) {
	base := NewMemoryStateDB()
	ldb := NewLayerDB(base)

	addr1 := types.HexToAddress("0x01")
	addr2 := types.HexToAddress("0x02")
	addr3 := types.HexToAddress("0x03")

	ldb.CreateAccount(addr1)
	ldb.CreateAccount(addr2)

	ldb.AddBalance(addr1, big.NewInt(100))

	ldb.NewLayer()
	ldb.CreateAccount(addr3)
	ldb.AddBalance(addr1, big.NewInt(50))
	ldb.AddBalance(addr2, big.NewInt(30))

	ldb.NewLayer()
	ldb.SetNonce(addr3, 1)
	ldb.SubBalance(addr2, big.NewInt(10000))

	fmt.Println(ldb.String())
	fmt.Println(ldb.GetBalance(addr1))
	fmt.Println(ldb.GetBalance(addr2))
	fmt.Println(ldb.GetBalance(addr3))

	ldb.MergeChanges()
	fmt.Println(ldb.String())
	fmt.Println(ldb.GetBalance(addr1))
	fmt.Println(ldb.GetBalance(addr2))
	fmt.Println(ldb.GetBalance(addr3))
}

