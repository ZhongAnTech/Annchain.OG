package processor

import (
	"fmt"
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/types/msg"
)

type OgMessageAdapter interface {
	AdaptMessage(incomingMsg msg.BinaryMessage) (msg.TransportableMessage, error)
}

type DefaultOgMessageAdapter struct {
}

func (d DefaultOgMessageAdapter) AdaptMessage(incomingMsg msg.BinaryMessage) (msg.TransportableMessage, error) {
	var m msg.TransportableMessage
	switch ogmessage.OgMessageType(incomingMsg.Type) {
	case ogmessage.MessageTypePing:
		m = &ogmessage.MessagePing{}
	case ogmessage.MessageTypePong:
		m = &ogmessage.MessagePong{}
	case ogmessage.MessageTypeBatchSyncRequest:
		m = &ogmessage.MessageBatchSyncRequest{}
	case ogmessage.MessageTypeSyncResponse:
		m = &ogmessage.MessageSyncResponse{}
	case ogmessage.MessageTypeNewResource:
		m = &ogmessage.MessageNewResource{}
	case ogmessage.MessageTypeHeightSyncRequest:
		m = &ogmessage.MessageHeightSyncRequest{}
	case ogmessage.MessageTypeHeaderRequest:
		m = &ogmessage.MessageHeaderRequest{}
	case ogmessage.MessageTypeAnnsensus:
		m = &ogmessage.MessageAnnsensus{}
	default:
		return nil, fmt.Errorf("Binary message type not supported: %d", incomingMsg.Type)
	}
	err := m.FromBinary(incomingMsg.Data)
	return m, err
}
