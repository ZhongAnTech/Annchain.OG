package communicator

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/annsensus"
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

type AnnsensusMessageAdapter interface {
	AdaptMessage(incomingMsg msg.TransportableMessage) (annsensus.AnnsensusMessage, error)
	AdaptAnnsensusMessage(outgoingMsg annsensus.AnnsensusMessage) (msg.TransportableMessage, error)
}

type PeerIdentifier string

type MessageEvent struct {
	Msg    msg.TransportableMessage
	Source PeerIdentifier
}

type OgPeerCommunicatorOutgoing interface {
	Broadcast(msg msg.TransportableMessage, peers []PeerIdentifier)
	Unicast(msg msg.TransportableMessage, peer PeerIdentifier)
}
type OgPeerCommunicatorIncoming interface {
	GetPipeIn() chan *MessageEvent
	GetPipeOut() chan *MessageEvent
}
