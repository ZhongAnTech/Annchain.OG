package annsensus

import (
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/message"
	"github.com/sirupsen/logrus"
)

type AnnsensusOgMessageHandler struct {
	AnnsensusPartner        annsensus.AnnsensusPartner
	AnnsensusMessageAdapter AnnsensusMessageAdapter
}

func (a AnnsensusOgMessageHandler) Handle(msgEvent *message.GeneralMessageEvent) {
	annsensusMessage, err := a.AnnsensusMessageAdapter.AdaptOgMessage(msgEvent.Msg)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt og message to annsensus")
		return
	}
	annsensusPeer, err := a.AnnsensusMessageAdapter.AdaptOgPeer(msgEvent.Source)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt og peer to annsensus")
		return
	}

	a.AnnsensusPartner.HandleAnnsensusMessage(&annsensus.AnnsensusMessageEvent{
		Message: annsensusMessage,
		Peer:    annsensusPeer,
	})
}
