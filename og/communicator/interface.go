package communicator

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/types/msg"
)

type P2PSender interface {
	BroadcastMessage(msg msg.OgMessage)
	AnonymousSendMessage(msg msg.OgMessage, anonymousPubKey *crypto.PublicKey)
	SendToPeer(msg msg.OgMessage, peerId PeerIdentifier) error
}

// P2PReceiver provides a channel for consumer to receive messages from p2p
type P2PReceiver interface {
	GetMessageChannel() chan msg.OgMessage
}

type AnnsensusMessageAdapter interface {
	AdaptMessage(incomingMsg msg.OgMessage) (annsensus.AnnsensusMessage, error)
	AdaptAnnsensusMessage(outgoingMsg annsensus.AnnsensusMessage) (msg.OgMessage, error)
}

type PeerIdentifier struct {
	Id int
}

type MessageEvent struct {
	Msg    msg.OgMessage
	Source PeerIdentifier
}

type OgPeerCommunicatorOutgoing interface {
	Broadcast(msg msg.OgMessage, peers []PeerIdentifier)
	Unicast(msg msg.OgMessage, peer PeerIdentifier)
}
type OgPeerCommunicatorIncoming interface {
	GetPipeIn() chan *MessageEvent
	GetPipeOut() chan *MessageEvent
}
