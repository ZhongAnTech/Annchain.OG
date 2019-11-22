package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/types/msg"
	"github.com/sirupsen/logrus"
)

type AnnsensusMessageHandler struct {
	peerCommunicator annsensus.AnnsensusPeerCommunicator
	//unmarshaller *AnnsensusMessageUnmarshaller
}

//func (a AnnsensusMessageHandler) UnmarshalMessage(message msg.BinaryMessage) (msg.OgMessage, error) {
//	return a.unmarshaller.Unmarshal(annsensus.AnnsensusMessageType(message.Type), message.Data)
//}

func (a AnnsensusMessageHandler) Handle(message msg.OgMessage, identifier communicator.PeerIdentifier) {
	messageType := annsensus.AnnsensusMessageType(message.GetType())

	switch messageType {
	case annsensus.AnnsensusMessageTypePlain:
		m := &annsensus.AnnsensusMessagePlain{}
		_, err = m.UnmarshalMsg(message.GetData())
		outMsg = m
	case annsensus.AnnsensusMessageTypeSigned:
		m := &annsensus.AnnsensusMessageSigned{}
		_, err = m.UnmarshalMsg(message.GetData())
		outMsg = m
	case annsensus.AnnsensusMessageTypeEncrypted:
		m := &annsensus.AnnsensusMessageEncrypted{}
		_, err = m.UnmarshalMsg(message.GetData())
		outMsg = m
	default:
		err = errors.New("message type of Annsensus not supported")
	}



	logrus.Info("I will handle this annsensus message")
}
