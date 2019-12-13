package communication

import (
	"github.com/annchain/OG/message"
)

type GeneralPeerCommunicatorOutgoing interface {
	Broadcast(msg message.GeneralMessage, peers []message.GeneralPeer)
	Unicast(msg message.GeneralMessage, peer message.GeneralPeer)
}
type GeneralPeerCommunicatorIncoming interface {
	GetPipeIn() chan *message.GeneralMessageEvent
	GetPipeOut() chan *message.GeneralMessageEvent
}

type GeneralMessageEventHandler interface {
	Handle(msgEvent *message.GeneralMessageEvent)
}

type GeneralMessageHandlerPlugin interface {
	SupportedMessageTypes() []message.GeneralMessageType
	GetMessageEventHandler() GeneralMessageEventHandler
	SetOutgoing(GeneralPeerCommunicatorOutgoing)
	Start()
	Stop()
}