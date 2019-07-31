package p2p_message

import (
	"fmt"
	"github.com/annchain/OG/types/tx_types"
	"testing"
)

func TestMessageNewActionTx_MarshalMsg(t *testing.T) {
	of := tx_types.NewPublicOffering()
	of.TokenId = 32
	of.TokenName = "haha"
	tx := MessageNewActionTx{
		ActionTx: &tx_types.ActionTx{
			ActionData: of,
			Action:     tx_types.ActionTxActionSPO,
		},
	}
	data, _ := tx.MarshalMsg(nil)
	var newTx = &MessageNewActionTx{}
	o, err := newTx.UnmarshalMsg(data)
	if err != nil {
		t.Fatal(err, o, data)
	}
	fmt.Println(newTx, newTx.ActionTx, newTx.ActionTx.ActionData)
}

func TestMessageType_GetMsg(t *testing.T) {
	var m = MessageTypePreVote
	msg := m.GetMsg()
	fmt.Println(msg)
}
