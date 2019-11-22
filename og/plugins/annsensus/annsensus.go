package annsensus

import (
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og/router"
	"github.com/annchain/OG/types/msg"
)

var supportedMessageTypes = []msg.BinaryMessageType{
	msg.BinaryMessageType(annsensus.AnnsensusMessageTypePlain),
	msg.BinaryMessageType(annsensus.AnnsensusMessageTypeSigned),
	msg.BinaryMessageType(annsensus.AnnsensusMessageTypeEncrypted),
}

type AnnsensusPlugin struct {
	messageHandler router.MessageHandler
}

func NewAnnsensusPlugin() *AnnsensusPlugin {
	return &AnnsensusPlugin{
		messageHandler: &AnnsensusMessageHandler{
			annsensusAdapter: &SimpleAnnsensusAdapter{
				annsensusMessageUnmarshaller: &AnnsensusMessageUnmarshaller{

				},
			},
		},
	}
}

func (a AnnsensusPlugin) SupportedMessageTypes() []msg.BinaryMessageType {
	return supportedMessageTypes
}

func (a AnnsensusPlugin) GetMessageHandler() router.MessageHandler {
	return a.messageHandler
}
