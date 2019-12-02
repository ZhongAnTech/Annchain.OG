package router

import (
	"github.com/annchain/OG/types/msg"
)



type MessageRouter struct {
	CallbackRegistgry map[msg.BinaryMessageType]OgMessageHandler // All message handlers
}

func NewMessageRouter() *MessageRouter {
	mr := &MessageRouter{CallbackRegistgry: make(map[msg.BinaryMessageType]OgMessageHandler)}
	return mr
}

func (m *MessageRouter) Register(binaryMessageType msg.BinaryMessageType, handler OgMessageHandler) {
	m.CallbackRegistgry[binaryMessageType] = handler
}

type NewOgMessageEvent