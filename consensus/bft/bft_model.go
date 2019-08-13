package bft

import (
	"fmt"
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp

//msgp:tuple BftMessageType
type BftMessageType int

func (m BftMessageType) String() string {
	switch m {
	case BftMessageTypeProposal:
		return "BFTProposal"
	case BftMessageTypePreVote:
		return "BFTPreVote"
	case BftMessageTypePreCommit:
		return "BFTPreCommit"
	default:
		return "BFTUnknown"
	}
}

const (
	BftMessageTypeProposal BftMessageType = iota
	BftMessageTypePreVote
	BftMessageTypePreCommit
)

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
