package annsensus

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
)

type MessageSender interface {
	BroadcastMessage(messageType og.MessageType, message types.Message)
	SendToAnynomous(messageType og.MessageType, msg types.Message, anyNomousPubKey *crypto.PublicKey)
}
