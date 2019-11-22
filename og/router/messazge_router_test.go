package router

import (
	"github.com/annchain/OG/og/plugins/og"
	"github.com/annchain/OG/types/msg"
	"testing"
)

func TestRouter(t *testing.T) {
	msgRouter := NewMessageRouter()

	msgRouter.Register(msg.BinaryMessageType(og.MessageTypePing), )
}
