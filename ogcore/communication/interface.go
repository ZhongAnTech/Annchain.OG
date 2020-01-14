package communication

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/ogcore/message"
)

type P2PSender interface {
	BroadcastMessage(msg message.OgMessage)
	AnonymousSendMessage(msg message.OgMessage, anonymousPubKey *crypto.PublicKey)
	SendToPeer(msg message.OgMessage, peerId OgPeer) error
}

// P2PReceiver provides a channel for consumer to receive messages from p2p
type P2PReceiver interface {
	GetMessageChannel() chan message.OgMessage
}

type OgPeerCommunicatorOutgoing interface {
	Broadcast(msg message.OgMessage)
	Multicast(msg message.OgMessage, peers []*OgPeer)
	Unicast(msg message.OgMessage, peer *OgPeer)
}
type OgPeerCommunicatorIncoming interface {
	GetPipeIn() chan *OgMessageEvent
	GetPipeOut() chan *OgMessageEvent
}
