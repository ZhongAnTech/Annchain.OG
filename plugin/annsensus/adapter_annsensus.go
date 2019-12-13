package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/message"
)

type AnnsensusMessageUnmarshaller struct {
}

func (a *AnnsensusMessageUnmarshaller) Unmarshal(
	messageType annsensus.AnnsensusMessageType, message []byte) (
	outMsg annsensus.AnnsensusMessage, err error) {
	switch messageType {
	case annsensus.AnnsensusMessageTypeBftPlain:
		m := &annsensus.AnnsensusMessageBftPlain{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case annsensus.AnnsensusMessageTypeBftSigned:
		m := &annsensus.AnnsensusMessageBftSigned{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case annsensus.AnnsensusMessageTypeBftEncrypted:
		m := &annsensus.AnnsensusMessageBftEncrypted{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case annsensus.AnnsensusMessageTypeDkgPlain:
		m := &annsensus.AnnsensusMessageDkgPlain{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case annsensus.AnnsensusMessageTypeDkgSigned:
		m := &annsensus.AnnsensusMessageDkgSigned{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	case annsensus.AnnsensusMessageTypeDkgEncrypted:
		m := &annsensus.AnnsensusMessageDkgEncrypted{}
		_, err = m.UnmarshalMsg(message)
		outMsg = m
	default:
		err = errors.New("message type of Annsensus not supported")
	}

	return
}

type DefaultAnnsensusMessageAdapter struct {
	unmarshaller AnnsensusMessageUnmarshaller
}

func (d *DefaultAnnsensusMessageAdapter) AdaptGeneralMessage(incomingMsg message.GeneralMessage) (annMessage annsensus.AnnsensusMessage, err error) {
	mssageType := incomingMsg.GetType()
	if mssageType != MessageTypeAnnsensus {
		err = errors.New("SimpleAnnsensusAdapter received a message of an unsupported type")
		return
	}
	gma := incomingMsg.(*GeneralMessageAnnsensus)

	return d.unmarshaller.Unmarshal(gma.InnerMessageType, gma.InnerMessage)
}

func (d *DefaultAnnsensusMessageAdapter) AdaptGeneralPeer(gnrPeer message.GeneralPeer) (annsensus.AnnsensusPeer, error) {
	return annsensus.AnnsensusPeer{
		Id:             gnrPeer.Id,
		PublicKey:      gnrPeer.PublicKey,
		Address:        gnrPeer.Address,
		PublicKeyBytes: gnrPeer.PublicKeyBytes,
	}, nil
}

func (d *DefaultAnnsensusMessageAdapter) AdaptAnnsensusMessage(outgoingMsg annsensus.AnnsensusMessage) (msg message.GeneralMessage, err error) {
	msg = &GeneralMessageAnnsensus{
		InnerMessageType: outgoingMsg.GetType(),
		InnerMessage:     outgoingMsg.GetBytes(),
	}
	return
}

func (d *DefaultAnnsensusMessageAdapter) AdaptAnnsensusPeer(annPeer annsensus.AnnsensusPeer) (message.GeneralPeer, error) {
	return message.GeneralPeer{
		Id:             annPeer.Id,
		PublicKey:      annPeer.PublicKey,
		Address:        annPeer.Address,
		PublicKeyBytes: annPeer.PublicKeyBytes,
	}, nil
}

