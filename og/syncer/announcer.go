package syncer

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
)

type Announcer struct {
	messageSender MessageSender
}

func NewAnnouncer(messageSender MessageSender) *Announcer {
	return &Announcer{
		messageSender: messageSender,
	}
}

//BroadcastNewTx brodcast newly created txi message
func (m *Announcer) BroadcastNewTx(txi types.Txi) {
	txType := txi.GetType()
	if txType == types.TxBaseTypeNormal {
		tx := txi.(*types.Tx)
		msgTx := types.MessageNewTx{RawTx: tx.RawTx()}
		m.messageSender.BroadcastMessage(og.MessageTypeNewTx, &msgTx)
	} else if txType == types.TxBaseTypeSequencer {
		seq := txi.(*types.Sequencer)
		msgTx := types.MessageNewSequencer{RawSequencer: seq.RawSequencer()}
		m.messageSender.BroadcastMessage(og.MessageTypeNewSequencer, &msgTx)
	} else {
		log.Warn("never come here, unknown tx type", txType)
	}
}
