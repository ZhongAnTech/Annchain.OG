package bft

import (
	"fmt"
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp

type Signable interface {
	msg.MsgpMember
	SignatureTargets() []byte
}

//msgp:tuple BftMessage
type BftMessage struct {
	Type    BftMessageType
	Payload Signable
}

func (m *BftMessage) String() string {
	return fmt.Sprintf("%s %+v", m.Type.String(), m.Payload)
}
