package og

import (
	"errors"
	general_message "github.com/annchain/OG/message"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/message"
)

type OgMessageUnmarshaller struct {
}

func (a *OgMessageUnmarshaller) Unmarshal(
	messageType message.OgMessageType, messageBytes []byte) (
	outMsg message.OgMessage, err error) {
	switch messageType {
	case message.OgMessageTypePing:
		m := &message.OgMessagePing{}
		_, err = m.UnmarshalMsg(messageBytes)
		outMsg = m
	case message.OgMessageTypePong:
		m := &message.OgMessagePong{}
		_, err = m.UnmarshalMsg(messageBytes)
		outMsg = m
	case message.OgMessageTypeBatchSyncRequest:
		m := &message.OgMessageBatchSyncRequest{}
		_, err = m.UnmarshalMsg(messageBytes)
		outMsg = m
	case message.OgMessageTypeSyncResponse:
		m := &message.OgMessageSyncResponse{}
		_, err = m.UnmarshalMsg(messageBytes)
		outMsg = m
	case message.OgMessageTypeNewResource:
		m := &message.OgMessageNewResource{}
		_, err = m.UnmarshalMsg(messageBytes)
		outMsg = m
	case message.OgMessageTypeHeightSyncRequest:
		m := &message.OgMessageHeightSyncRequest{}
		_, err = m.UnmarshalMsg(messageBytes)
		outMsg = m
	case message.OgMessageTypeHeaderRequest:
		m := &message.OgMessageHeaderRequest{}
		_, err = m.UnmarshalMsg(messageBytes)
		outMsg = m
	default:
		err = errors.New("message type of OG not supported")
	}
	return
}

type DefaultOgMessageAdapter struct {
	unmarshaller OgMessageUnmarshaller
}

func (d *DefaultOgMessageAdapter) AdaptGeneralMessage(incomingMsg general_message.GeneralMessage) (ogMessage message.OgMessage, err error) {
	mssageType := incomingMsg.GetType()
	if mssageType != MessageTypeOg {
		err = errors.New("SimpleOgAdapter received a message of an unsupported type")
		return
	}
	gma := incomingMsg.(*GeneralMessageOg)

	return d.unmarshaller.Unmarshal(gma.InnerMessageType, gma.InnerMessage)
}

func (d *DefaultOgMessageAdapter) AdaptGeneralPeer(gnrPeer general_message.GeneralPeer) (communication.OgPeer, error) {
	return communication.OgPeer{
		Id:             gnrPeer.Id,
		PublicKey:      gnrPeer.PublicKey,
		Address:        gnrPeer.Address,
		PublicKeyBytes: gnrPeer.PublicKeyBytes,
	}, nil
}

func (d *DefaultOgMessageAdapter) AdaptOgMessage(outgoingMsg message.OgMessage) (msg general_message.GeneralMessage, err error) {
	msg = &GeneralMessageOg{
		InnerMessageType: outgoingMsg.GetType(),
		InnerMessage:     outgoingMsg.GetBytes(),
	}
	return
}

func (d *DefaultOgMessageAdapter) AdaptOgPeer(annPeer communication.OgPeer) (general_message.GeneralPeer, error) {
	return general_message.GeneralPeer{
		Id:             annPeer.Id,
		PublicKey:      annPeer.PublicKey,
		Address:        annPeer.Address,
		PublicKeyBytes: annPeer.PublicKeyBytes,
	}, nil
}
