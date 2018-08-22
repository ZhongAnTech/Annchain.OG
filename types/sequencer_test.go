package types

import (
	"testing"
	"fmt"
	"github.com/annchain/OG/common/math"
)

func TestSequencer(t *testing.T) {
	seq1 := Sequencer{TxBase: TxBase{ParentsHash: []Hash{HexToHash("0x0")}}}
	seq2 := Tx{TxBase: TxBase{ParentsHash: []Hash{HexToHash("0x0")}},
		To: HexToAddress("0x1"),
		From: HexToAddress("0x1"),
		Value: math.NewBigInt(0),
	}

	seq3 := Sequencer{TxBase: TxBase{ParentsHash: []Hash{seq1.BlockHash(), seq2.BlockHash()}}}

	fmt.Println(seq1.BlockHash().String())
	fmt.Println(seq2.BlockHash().String())
	fmt.Println(seq3.BlockHash().String())

}
