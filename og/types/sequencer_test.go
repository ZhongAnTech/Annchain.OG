package types

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"testing"
)

func TestSampleSequencer(t *testing.T) {
	s := &Sequencer{
		//Hash:           common.Hash{},
		ParentsHash:  types.Hashes{types.HexToHash("0xCCDD"), types.HexToHash("0xEEFF")},
		Height:       12,
		MineNonce:    23,
		Weight:       4,
		AccountNonce: 0,
		Issuer:       common.HexToAddress("0x33"),
		//Signature:    nil,
		//PublicKey: nil,
		StateRoot: types.Hash{},
		//Proposing:      false,
	}
	fmt.Println(s)
}

func TestSigning(t *testing.T) {
	// TODO
}
