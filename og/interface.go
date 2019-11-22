package og

import (
	"github.com/annchain/OG/og/router"
	"github.com/annchain/OG/types/msg"
)

type Plugin interface {
	SupportedMessageTypes() []msg.BinaryMessageType
	GetMessageHandler() router.MessageHandler
}
