package og

import (
	general_communication "github.com/annchain/OG/communication"
	"github.com/annchain/OG/eventbus"
	general_message "github.com/annchain/OG/message"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/message"
)

var supportedMessageTypes = []general_message.GeneralMessageType{
	MessageTypeOg,
}

type OgPlugin struct {
	messageHandler general_communication.GeneralMessageEventHandler
	OgPartner      *ogcore.OgPartner
	Communicator   *ProxyOgPeerCommunicator
}

func (o *OgPlugin) SupportedEvents() []eventbus.EventRegisterInfo {
	return []eventbus.EventRegisterInfo{}
}

func (o *OgPlugin) SetOutgoing(outgoing general_communication.GeneralPeerCommunicatorOutgoing) {
	o.Communicator.GeneralOutgoing = outgoing
}

func (o *OgPlugin) Start() {
	o.OgPartner.Start()
}

func (o *OgPlugin) Stop() {
	o.OgPartner.Stop()
}

func NewOgPlugin() *OgPlugin {
	config := ogcore.OgProcessorConfig{

	}
	ogMessageAdapter := &DefaultOgMessageAdapter{
		unmarshaller: OgMessageUnmarshaller{},
	}

	communicator := &ProxyOgPeerCommunicator{
		OgMessageAdapter: ogMessageAdapter,
		GeneralOutgoing:  nil, // place for p2p peer outgoing (set later)
	}
	communicator.InitDefault()

	ogPartner := &ogcore.OgPartner{
		Config:       config,
		PeerOutgoing: communicator,
		PeerIncoming: communicator,
	}

	return &OgPlugin{
		messageHandler: &OgGeneralMessageHandler{
			OgPartner:        ogPartner,
			OgMessageAdapter: ogMessageAdapter,
		},
		OgPartner:    ogPartner,
		Communicator: communicator,
	}
}

func (o *OgPlugin) SupportedMessageTypes() []general_message.GeneralMessageType {
	return supportedMessageTypes
}

func (o *OgPlugin) GetMessageEventHandler() general_communication.GeneralMessageEventHandler {
	return o.messageHandler
}

type OgMessageAdapter interface {
	AdaptGeneralMessage(incomingMsg general_message.GeneralMessage) (ogMessage message.OgMessage, err error)
	AdaptGeneralPeer(gnrPeer general_message.GeneralPeer) (communication.OgPeer, error)
	AdaptOgMessage(outgoingMsg message.OgMessage) (msg general_message.GeneralMessage, err error)
	AdaptOgPeer(annPeer communication.OgPeer) (general_message.GeneralPeer, error)
}
