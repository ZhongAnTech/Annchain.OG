package og

import (
	"github.com/annchain/OG/message"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/sirupsen/logrus"
)

type OgGeneralMessageHandler struct {
	OgPartner        *ogcore.OgPartner
	OgMessageAdapter OgMessageAdapter
}

func (a OgGeneralMessageHandler) Handle(msgEvent *message.GeneralMessageEvent) {
	annsensusMessage, err := a.OgMessageAdapter.AdaptGeneralMessage(msgEvent.Message)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt og message to general")
		return
	}
	ogPeer, err := a.OgMessageAdapter.AdaptGeneralPeer(msgEvent.Peer)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt general peer to og")
		return
	}

	a.OgPartner.HandleOgMessage(&communication.OgMessageEvent{
		Message: annsensusMessage,
		Peer:    ogPeer,
	})
}
