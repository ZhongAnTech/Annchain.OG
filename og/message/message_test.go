package message

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/protocol_message"
	"testing"
)

func TestMarshal(t *testing.T) {
	sp := SequencerProposal{
		protocol_message.Sequencer{
			TxBase: protocol_message.TxBase{
				Type:         0,
				Hash:         common.Hash{},
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
			StateRoot:      common.Hash{},
			Proposing:      false,
		},
	}

	mp := bft.MessageProposal{
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
