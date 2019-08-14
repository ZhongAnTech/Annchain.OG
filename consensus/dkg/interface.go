package dkg

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types/p2p_message"
)

type P2PSender interface {
	BroadcastMessage(messageType p2p_message.MessageType, message p2p_message.Message)
	AnonymousSendMessage(messageType p2p_message.MessageType, msg p2p_message.Message, anonymousPubKey *crypto.PublicKey)
	SendToPeer(messageType p2p_message.MessageType, msg p2p_message.Message, peerId string) error
}
