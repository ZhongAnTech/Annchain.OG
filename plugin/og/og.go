package og

import (
	general_communication "github.com/annchain/OG/communication"
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/eventbus"
	general_message "github.com/annchain/OG/message"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/message"
	"github.com/annchain/OG/ogcore/pool"
	"github.com/sirupsen/logrus"
)

var supportedMessageTypes = []general_message.GeneralMessageType{
	MessageTypeOg,
}

type OgPlugin struct {
	messageHandler general_communication.GeneralMessageEventHandler
	OgPartner      *ogcore.OgPartner
	Communicator   *ProxyOgPeerCommunicator
	TxBuffer       *pool.TxBuffer
}

func NewOgPlugin() *OgPlugin {
	config := ogcore.OgProcessorConfig{}
	ogMessageAdapter := &DefaultOgMessageAdapter{
		unmarshaller: OgMessageUnmarshaller{},
	}

	communicator := &ProxyOgPeerCommunicator{
		OgMessageAdapter: ogMessageAdapter,
		GeneralOutgoing:  nil, // place for p2p peer outgoing (set later)
	}
	communicator.InitDefault()

	ogCore := &ogcore.OgCore{
		NodeLogger: debuglog.NodeLogger{
			Logger: logrus.StandardLogger(),
		},
		EventBus: nil,
	}

	ogPartner := &ogcore.OgPartner{
		NodeLogger: debuglog.NodeLogger{
			Logger: logrus.StandardLogger(),
		},
		Config:         config,
		PeerOutgoing:   communicator,
		PeerIncoming:   communicator,
		EventBus:       nil,
		StatusProvider: ogCore,
		OgCore:         ogCore,
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

func (o *OgPlugin) SupportedEventHandlers() []eventbus.EventHandlerRegisterInfo {
	return []eventbus.EventHandlerRegisterInfo{
		{
			Type:    events.TxReceivedEventType,
			Handler: o.TxBuffer,
		},
		{
			Type:    events.SequencerReceivedEventType,
			Handler: o.TxBuffer,
		},
	}
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

func (o *OgPlugin) SupportedMessageTypes() []general_message.GeneralMessageType {
	return supportedMessageTypes
}

func (o *OgPlugin) GetMessageEventHandler() general_communication.GeneralMessageEventHandler {
	return o.messageHandler
}

type OgMessageAdapter interface {
	AdaptGeneralMessage(incomingMsg general_message.GeneralMessage) (ogMessage message.OgMessage, err error)
	AdaptGeneralPeer(gnrPeer *general_message.GeneralPeer) (communication.OgPeer, error)
	AdaptOgMessage(outgoingMsg message.OgMessage) (msg general_message.GeneralMessage, err error)
	AdaptOgPeer(annPeer *communication.OgPeer) (general_message.GeneralPeer, error)
}
