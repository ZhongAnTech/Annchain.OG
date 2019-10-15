package communicator

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types/msg"
)

type P2PSender interface {
	BroadcastMessage(msg msg.TransportableMessage)
	AnonymousSendMessage(msg msg.TransportableMessage, anonymousPubKey *crypto.PublicKey)
	SendToPeer(msg msg.TransportableMessage, peerId string) error
}

// P2PReceiver provides a channel for consumer to receive messages from p2p
type P2PReceiver interface {
	GetMessageChannel() chan msg.TransportableMessage
}
