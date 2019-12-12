package ogcore

import (
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/message"
)

type OgMessageEventHandler interface {
	Handle(msgEvent *communication.OgMessageEvent)
}

type OgPlugin interface {
	SupportedMessageTypes() []message.OgMessageType
	GetMessageEventHandler() OgMessageEventHandler
}
