package message

import (
	"errors"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/types/p2p_message"
)

type OGMessageUnmarshaller interface {
	DoUnmarshal(message *OGMessage) error
}

type OGMessageUnmarshalManager struct {
	unmarshallers []OGMessageUnmarshaller
}

func (ma *OGMessageUnmarshalManager) RegisterUnmarshall(m OGMessageUnmarshaller) {
	ma.unmarshallers = append(ma.unmarshallers, m)
}

func (ma *OGMessageUnmarshalManager) Unmarshal(message *OGMessage) error {
	// try plugins
	for _, m := range ma.unmarshallers {
		err := m.DoUnmarshal(message)
		if err == nil {
			return nil
		}
	}
	return errors.New("message cannot be unmarshalled. check messageunmarshalmanager")

}

type OGBasicMessageUnmarshaller struct {
}

func (d OGBasicMessageUnmarshaller) DoUnmarshal(message *OGMessage) error {
	err := message.Unmarshal()
	return err
}


type MessageConsensusUnmarshaller struct {
}

func (m MessageConsensusUnmarshaller) DoUnmarshal(msg *OGMessage) error {
	var inner p2p_message.Message
	switch bft.BftMessageType(msg.MessageType) {
	case bft.BftMessageTypeProposal:
		inner = &bft.MessageProposal{}
	case bft.BftMessageTypePreVote:
		inner = &bft.MessagePreVote{}
	case bft.BftMessageTypePreCommit:
		inner = &bft.MessagePreCommit{}
	default:
		return errors.New("unsupported type")
	}
	_, err := inner.UnmarshalMsg(msg.Data)
	if err != nil {
		return err
	}
	msg.Message = inner
	return nil
}

