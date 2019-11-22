package og

import (
	"github.com/annchain/OG/og/router"
	"github.com/annchain/OG/types/msg"
)

var supportedMessageTypes = []msg.BinaryMessageType{
	msg.BinaryMessageType(MessageTypePing),
	msg.BinaryMessageType(MessageTypePong),
	msg.BinaryMessageType(MessageTypeBatchSyncRequest),
	msg.BinaryMessageType(MessageTypeSyncResponse),
	msg.BinaryMessageType(MessageTypeNewResource),
	msg.BinaryMessageType(MessageTypeHeightSyncRequest),
	msg.BinaryMessageType(MessageTypeTxsRequest),
	msg.BinaryMessageType(MessageTypeHeaderRequest),
}

type OgPlugin struct {
}

func (o OgPlugin) SupportedMessageTypes() []msg.BinaryMessageType {
	return supportedMessageTypes
}

func (o OgPlugin) GetMessageHandler() router.MessageHandler {

}
