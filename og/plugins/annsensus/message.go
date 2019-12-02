package annsensus

import (
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp

var MessageTypeAnnsensus msg.OgMessageType = 101

//msgp:tuple MessageAnnsensus
type MessageAnnsensus struct {
	InnerMessageType annsensus.AnnsensusMessageType
	InnerMessage     []byte
}

func (m *MessageAnnsensus) GetType() msg.OgMessageType {
	return MessageTypeAnnsensus
}

func (m *MessageAnnsensus) String() string {
	return "MessageAnnsensus"
}

func (z *MessageAnnsensus) Marshal() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *MessageAnnsensus) Unmarshal(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}
