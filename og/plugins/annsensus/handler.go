package annsensus

import (
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/types/msg"
	"github.com/sirupsen/logrus"
)

type AnnsensusMessageHandler struct {
	unmarshaller *AnnsensusMessageUnmarshaller
}

func (a AnnsensusMessageHandler) UnmarshalMessage(message msg.BinaryMessage) (msg.TransportableMessage, error) {
	return a.unmarshaller.Unmarshal(annsensus.AnnsensusMessageType(message.Type), message.Data)
}

func (a AnnsensusMessageHandler) Handle(message msg.TransportableMessage, identifier communicator.PeerIdentifier) {
	logrus.Info("I will handle this annsensus message")
}
