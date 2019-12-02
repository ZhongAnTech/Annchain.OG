package og

import (
	"github.com/annchain/OG/types/msg"
)

var supportedMessageTypes = []msg.OgMessageType{
	MessageTypePing,
	MessageTypePong,
	MessageTypeBatchSyncRequest,
	MessageTypeSyncResponse,
	MessageTypeNewResource,
	MessageTypeHeightSyncRequest,
	MessageTypeTxsRequest,
	MessageTypeHeaderRequest,
}

//type OgPlugin struct {
//
//}
//
//func (o OgPlugin) SupportedMessageTypes() []msg.BinaryMessageType {
//	return supportedMessageTypes
//}
//
//func (o OgPlugin) GetMessageHandler() router.OgMessageHandler {
//
//}
