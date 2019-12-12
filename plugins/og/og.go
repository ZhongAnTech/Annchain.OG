package og

import (
	general_message "github.com/annchain/OG/message"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/message"
)

type OgMessageAdapter interface {
	AdaptGeneralMessage(incomingMsg general_message.GeneralMessage) (ogMessage message.OgMessage, err error)
	AdaptGeneralPeer(gnrPeer general_message.GeneralPeer) (communication.OgPeer, error)
	AdaptOgMessage(outgoingMsg message.OgMessage) (msg general_message.GeneralMessage, err error)
	AdaptOgPeer(annPeer communication.OgPeer) (general_message.GeneralPeer, error)
}
