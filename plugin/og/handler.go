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
	ogPeer, err := a.OgMessageAdapter.AdaptGeneralPeer(msgEvent.Sender)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt general peer to og")
		return
	}
	ogMsgEvent := &communication.OgMessageEvent{
		Message: annsensusMessage,
		Peer:    ogPeer,
	}
	a.OgPartner.PeerIncoming.GetPipeIn() <- ogMsgEvent

	//<-ffchan.NewTimeoutSenderShort(
	//	a.OgPartner.PeerIncoming.GetPipeIn(), ogMsgEvent, "og handler").C

}
