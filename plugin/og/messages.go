package og

import (
	general_message "github.com/annchain/OG/message"
	"github.com/annchain/OG/ogcore/message"
)

//go:generate msgp

var MessageTypeOg general_message.GeneralMessageType = 1

//msgp:tuple GeneralMessageOg
type GeneralMessageOg struct {
	InnerMessageType message.OgMessageType
	InnerMessage     []byte
}

func (g *GeneralMessageOg) GetType() general_message.GeneralMessageType {
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
