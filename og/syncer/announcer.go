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
	switch tx := txi.(type) {
	case *types.Tx:
		msgTx := types.MessageNewTx{RawTx: tx.RawTx()}
		m.messageSender.BroadcastMessageWithLink(og.MessageTypeNewTx, &msgTx)
	case *types.Sequencer:
		msgTx := types.MessageNewSequencer{RawSequencer: tx.RawSequencer()}
		m.messageSender.BroadcastMessageWithLink(og.MessageTypeNewSequencer, &msgTx)
	case *types.Campaign:
		msg := types.MessageCampaign{
			RawCampaign: tx.RawCampaign(),
		}
		m.messageSender.BroadcastMessage(og.MessageTypeCampaign, &msg)
	case *types.TermChange:
		msg := types.MessageTermChange{
			RawTermChange: tx.RawTermChange(),
		}
		m.messageSender.BroadcastMessage(og.MessageTypeTermChange, &msg)
	default:
		log.Warn("never come here, unknown tx type ", tx)
	}
}
