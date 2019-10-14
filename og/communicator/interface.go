package communicator

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types/general_message"
)

type P2PSender interface {
	BroadcastMessage(msg general_message.TransportableMessage)
	AnonymousSendMessage(msg general_message.TransportableMessage, anonymousPubKey *crypto.PublicKey)
	SendToPeer(msg general_message.TransportableMessage, peerId string) error
}

// P2PReceiver provides a channel for consumer to receive messages from p2p
type P2PReceiver interface {
	GetMessageChannel() chan general_message.TransportableMessage
}
