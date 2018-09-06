package wserver

import (
	"testing"
	"github.com/annchain/OG/types"
	"fmt"
)

func TestConvertor(t *testing.T){
	tx := types.Tx{
		TxBase:types.TxBase{
			Hash: types.BytesToHash([]byte{1,2,3,4,5}),
			ParentsHash:[]types.Hash{types.BytesToHash([]byte{1,1,2,2,3,3})},
		},
		From:types.HexToAddress("0x12345"),
		To:types.HexToAddress("0x56789"),
		Value:nil,

	}
	fmt.Println(tx2UIData(tx))
}
