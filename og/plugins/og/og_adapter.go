package og

import (
	"fmt"
	"github.com/annchain/OG/types/msg"
)

type OgMessageAdapter interface {
	AdaptMessage(incomingMsg msg.BinaryMessage) (msg.TransportableMessage, error)
}

type DefaultOgMessageAdapter struct {
}

func (d DefaultOgMessageAdapter) AdaptMessage(incomingMsg msg.BinaryMessage) (msg.TransportableMessage, error) {
	var m msg.TransportableMessage
	switch OgMessageType(incomingMsg.Type) {
	case MessageTypePing:
		m = &MessagePing{}
	case MessageTypePong:
		m = &MessagePong{}
	case MessageTypeBatchSyncRequest:
		m = &MessageBatchSyncRequest{}
	case MessageTypeSyncResponse:
		m = &MessageSyncResponse{}
	case MessageTypeNewResource:
		m = &MessageNewResource{}
	case MessageTypeHeightSyncRequest:
		m = &MessageHeightSyncRequest{}
	case MessageTypeHeaderRequest:
		m = &MessageHeaderRequest{}
	case MessageTypeAnnsensus:
		m = &MessageAnnsensus{}
	default:
		return nil, fmt.Errorf("Binary message type not supported: %d", incomingMsg.Type)
	}
	err := m.FromBinary(incomingMsg.Data)
	return m, err
}
