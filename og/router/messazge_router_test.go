package router

import (
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/types/msg"
	"testing"
)

func TestRouter(t *testing.T) {
	msgRouter := NewMessageRouter()

	ogProcessor := ogP

	msgRouter.Register(msg.BinaryMessageType(ogmessage.MessageTypePing), )
}
