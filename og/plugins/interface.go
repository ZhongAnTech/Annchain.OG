package plugins

import (
	"github.com/annchain/OG/og/communication"
	"github.com/annchain/OG/types/msg"
)

type OgMessageHandler interface {
	Handle(message msg.OgMessage, identifier communication.OgPeer)
}

type OgPlugin interface {
	SupportedMessageTypes() []msg.OgMessageType
	GetMessageHandler() OgMessageHandler
}
