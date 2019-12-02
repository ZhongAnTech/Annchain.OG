package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og/communication"
	"github.com/annchain/OG/types/msg"
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

func (d *DefaultAnnsensusMessageAdapter) AdaptOgMessage(incomingMsg msg.OgMessage) (annMessage annsensus.AnnsensusMessage, err error) {
	mssageType := incomingMsg.GetType()
	if mssageType != MessageTypeAnnsensus {
		err = errors.New("SimpleAnnsensusAdapter received a message of an unsupported type")
		return
	}

	return d.unmarshaller.Unmarshal(annsensus.AnnsensusMessageType(incomingMsg.GetType()), incomingMsg.GetData())
}

func (d *DefaultAnnsensusMessageAdapter) AdaptOgPeer(ogPeer communication.OgPeer) (annsensus.AnnsensusPeer, error) {
	return annsensus.AnnsensusPeer{
		Id:             ogPeer.Id,
		PublicKey:      ogPeer.PublicKey,
		Address:        ogPeer.Address,
		PublicKeyBytes: ogPeer.PublicKeyBytes,
	}, nil
}

func (d *DefaultAnnsensusMessageAdapter) AdaptAnnsensusMessage(outgoingMsg annsensus.AnnsensusMessage) (msg.OgMessage, error) {
	panic("implement me")
}

func (d *DefaultAnnsensusMessageAdapter) AdaptAnnsensusPeer(annPeer annsensus.AnnsensusPeer) (communication.OgPeer, error) {
	return communication.OgPeer{
		Id:             annPeer.Id,
		PublicKey:      annPeer.PublicKey,
		Address:        annPeer.Address,
		PublicKeyBytes: annPeer.PublicKeyBytes,
	}, nil
}
