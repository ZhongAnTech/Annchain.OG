package communicator

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types/p2p_message"
)

type P2PSender interface {
	BroadcastMessage(messageType p2p_message.MessageType, message p2p_message.Message)
	SendToAnynomous(messageType p2p_message.MessageType, msg p2p_message.Message, anonymousPubKey *crypto.PublicKey)
	SendToPeer(peerId string, messageType p2p_message.MessageType, msg p2p_message.Message) error
}

// ConsensusAccountProvider provides public key and private key for signing consensus messages
type ConsensusAccountProvider interface {
	Account() *account.Account
}
