package engine

import (
	"github.com/annchain/OG/og/communication"
	"github.com/annchain/OG/types/msg"
)

type OgMessageEventHandler interface {
	Handle(msgEvent *communication.OgMessageEvent)
}

type OgPlugin interface {
	SupportedMessageTypes() []msg.OgMessageType
	GetMessageEventHandler() OgMessageEventHandler
}
