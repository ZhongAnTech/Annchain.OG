package communication

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types/msg"
)

type P2PSender interface {
	BroadcastMessage(msg msg.OgMessage)
	AnonymousSendMessage(msg msg.OgMessage, anonymousPubKey *crypto.PublicKey)
	SendToPeer(msg msg.OgMessage, peerId OgPeer) error
}

// P2PReceiver provides a channel for consumer to receive messages from p2p
type P2PReceiver interface {
	GetMessageChannel() chan msg.OgMessage
}

type OgPeer struct {
	Id             int
	PublicKey      crypto.PublicKey `json:"-"`
	Address        common.Address   `json:"address"`
	PublicKeyBytes hexutil.Bytes    `json:"public_key"`
}

type OgMessageEvent struct {
	Msg    msg.OgMessage
	Source OgPeer
}

type OgPeerCommunicatorOutgoing interface {
	Broadcast(msg msg.OgMessage, peers []OgPeer)
	Unicast(msg msg.OgMessage, peer OgPeer)
}
type OgPeerCommunicatorIncoming interface {
	GetPipeIn() chan *OgMessageEvent
	GetPipeOut() chan *OgMessageEvent
}
