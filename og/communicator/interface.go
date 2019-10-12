package communicator

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types/p2p_message"
)

type P2PSender interface {
	BroadcastMessage(messageType message.BinaryMessageType, message p2p_message.BinaryMessage)
	AnonymousSendMessage(messageType message.BinaryMessageType, msg p2p_message.BinaryMessage, anonymousPubKey *crypto.PublicKey)
	SendToPeer(messageType message.BinaryMessageType, msg p2p_message.BinaryMessage, peerId string) error
}

// P2PReceiver provides a channel for consumer to receive messages from p2p
type P2PReceiver interface {
	GetMessageChannel() chan p2p_message.Message
}