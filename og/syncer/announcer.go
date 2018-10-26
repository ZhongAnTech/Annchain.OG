package syncer

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
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
		msgTx := types.MessageNewTx{Tx: tx}
		data, _ := msgTx.MarshalMsg(nil)
		m.messageSender.BroadcastMessage(og.MessageTypeNewTx, data)
	} else if txType == types.TxBaseTypeSequencer {
		seq := txi.(*types.Sequencer)
		msgTx := types.MessageNewSequencer{seq}
		data, _ := msgTx.MarshalMsg(nil)
		m.messageSender.BroadcastMessage(og.MessageTypeNewSequencer, data)
	} else {
		logrus.Warn("never come here ,unkown tx type", txType)
	}
}
