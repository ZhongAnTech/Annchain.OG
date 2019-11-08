package types

import (
	"fmt"
	"github.com/annchain/OG/common"
	"testing"
)

func TestSampleSequencer(t *testing.T) {
	s := &Sequencer{
		//Hash:           common.Hash{},
		ParentsHash:  common.Hashes{common.HexToHash("0xCCDD"), common.HexToHash("0xEEFF")},
		Height:       12,
		MineNonce:    23,
		Weight:       4,
		AccountNonce: 0,
		Issuer:       common.HexToAddress("0x33"),
		//Signature:    nil,
		//PublicKey: nil,
		StateRoot: common.Hash{},
		//Proposing:      false,
	}
	fmt.Println(s)
}

func TestSigning(t *testing.T) {
	// TODO
}
