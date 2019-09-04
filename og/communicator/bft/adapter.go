package bft

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/message"
)

type BftCommunicatorAdapter interface {
	AdaptOgMessage(incomingMsg *message.OGMessage) (bft.BftMessage, error)
}
