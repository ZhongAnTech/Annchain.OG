package communicator

import (
	"errors"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
	"github.com/annchain/OG/types/msg"
)

type AnnsensusMessageUnmarshaller struct {
}

func (a AnnsensusMessageUnmarshaller) Unmarshal(messageType annsensus.AnnsensusMessageType, message []byte) (outMsg annsensus.AnnsensusMessage, err error) {
	switch messageType {
	case annsensus.AnnsensusMessageTypePlain:
		m := &annsensus.AnnsensusMessagePlain{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case annsensus.AnnsensusMessageTypeSigned:
		m := &annsensus.AnnsensusMessageSigned{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case annsensus.AnnsensusMessageTypeEncrypted:
		m := &annsensus.AnnsensusMessageEncrypted{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	default:
		err = errors.New("message type of Annsensus not supported")
	}
	return
}

type SimpleAnnsensusAdapter struct {
	annsensusMessageUnmarshaller *AnnsensusMessageUnmarshaller
}

func (s SimpleAnnsensusAdapter) AdaptOgMessage(incomingMsg msg.TransportableMessage) (msg annsensus.AnnsensusMessage, err error) {
	if incomingMsg.GetType() != archive.MessageTypeAnnsensus {
		err = errors.New("SimpleAnnsensusAdapter received a message of an unsupported type")
		return
	}
	// incomingMsg.GetType() == ogmessage.MessageTypeAnnsensus
	// incomingMsg.GetData
	return s.annsensusMessageUnmarshaller.Unmarshal(incomingMsg.GetType(), )
}

func (s SimpleAnnsensusAdapter) AdaptAnnsensusMessage(outgoingMsg annsensus.AnnsensusMessage) (msg.TransportableMessage, error) {
	panic("implement me")
}
