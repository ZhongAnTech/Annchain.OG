package router

import "github.com/annchain/OG/types/msg"

type MessageHandler interface {
	UnmarshalMessage(message msg.BinaryMessage) msg.TransportableMessage
	Handle(message msg.TransportableMessage)
}

type MessageRouter struct {
	CallbackRegistgry map[msg.BinaryMessageType]MessageHandler // All message handlers
}

func NewMessageRouter() *MessageRouter {
	mr := &MessageRouter{CallbackRegistgry: make(map[msg.BinaryMessageType]MessageHandler)}
	return mr
}

func (m *MessageRouter) Register(binaryMessageType msg.BinaryMessageType, handler MessageHandler) {
	m.CallbackRegistgry[binaryMessageType] = handler
}
