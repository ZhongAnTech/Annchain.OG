package message_archive

import (
	"fmt"
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/types"

	"testing"
)

func TestMarshal(t *testing.T) {
	sp := SequencerProposal{
		types.Sequencer{
			TxBase: types.TxBase{
				Type:         0,
				Hash:         types2.Hash{},
				ParentsHash:  nil,
				AccountNonce: 0,
				Height:       0,
				PublicKey:    nil,
				Signature:    nil,
				MineNonce:    0,
				Weight:       0,
				Version:      0,
			},
			Issuer:         nil,
			BlsJointSig:    nil,
			BlsJointPubKey: nil,
			StateRoot:      types2.Hash{},
			Proposing:      false,
		},
	}

	mp := bft.BftMessageProposal{
		BftBasicInfo: bft.BftBasicInfo{
			SourceId: 777,
			HeightRound: bft.HeightRound{
				Height: 22,
				Round:  33,
			},
		},
		Value:      &sp,
		ValidRound: 0,
	}
	var buffer []byte
	buffer, err := mp.MarshalMsg(buffer)
	if err != nil{
		panic(err)
	}
	fmt.Println(hexutil.Encode(buffer))

}
