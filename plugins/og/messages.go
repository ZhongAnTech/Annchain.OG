package og

import (
	"github.com/annchain/OG/message"
)

//go:generate msgp

var MessageTypeOg message.GeneralMessageType = 1

var supportedMessageTypes = []message.GeneralMessageType{
	MessageTypeOg,
}

//msgp:tuple GeneralMessageOg
type GeneralMessageOg struct {
	InnerMessageType message.OgMessageType
	InnerMessage     []byte
}

func (g *GeneralMessageOg) GetType() message.GeneralMessageType {
	return MessageTypeOg
}

func (g *GeneralMessageOg) GetBytes() []byte {
	b, err := g.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (g *GeneralMessageOg) String() string {
	return "GeneralMessageOg"
}
