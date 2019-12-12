package annsensus

import (
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/message"
)

//go:generate msgp

var MessageTypeAnnsensus message.GeneralMessageType = 2

//msgp:tuple GeneralMessageAnnsensus
type GeneralMessageAnnsensus struct {
	InnerMessageType annsensus.AnnsensusMessageType
	InnerMessage     []byte
}

func (m *GeneralMessageAnnsensus) GetBytes() []byte {
	b, err := m.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (m *GeneralMessageAnnsensus) GetType() message.GeneralMessageType {
	return MessageTypeAnnsensus
}

func (m *GeneralMessageAnnsensus) String() string {
	return "GeneralMessageAnnsensus"
}
