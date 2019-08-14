package bft

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/model"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"testing"
)

func TestMarshal(t *testing.T) {
	sp := model.SequencerProposal{
		tx_types.Sequencer{
			TxBase: types.TxBase{
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

	mp := MessageProposal{
		MessageConsensus: MessageConsensus{
			SourceId: 777,
			HeightRound: HeightRound{
				Height: 22,
				Round:  33,
			},
		},
		Value:      &sp,
		ValidRound: 0,
	}
	var buffer []byte
	_, err := mp.MarshalMsg(buffer)
	if err != nil{
		panic(err)
	}
}
