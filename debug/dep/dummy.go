package dep

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/message"
	"github.com/sirupsen/logrus"
)

// LocalGeneralPeerCommunicator is the place for change GeneralMessage locally.
type LocalGeneralPeerCommunicator struct {
	Myid    int
	PeerIns []chan *message.GeneralMessageEvent
	pipe    chan *message.GeneralMessageEvent //pipeIn is the receiver of the outside messages
	me      message.GeneralPeer
}

func (d *LocalGeneralPeerCommunicator) HandleIncomingMessage(msgEvent *message.GeneralMessageEvent) {
	d.pipe <- msgEvent
}

func NewLocalGeneralPeerCommunicator(myid int, incoming chan *message.GeneralMessageEvent, peers []chan *message.GeneralMessageEvent) *LocalGeneralPeerCommunicator {
	d := &LocalGeneralPeerCommunicator{
		PeerIns: peers,
		Myid:    myid,
		pipe:    incoming,
		me: message.GeneralPeer{
			Id:             myid,
			PublicKey:      crypto.PublicKey{},
			Address:        common.Address{},
			PublicKeyBytes: nil,
		},
	}
	return d
}

func (d *LocalGeneralPeerCommunicator) Broadcast(msg message.GeneralMessage, peers []message.GeneralPeer) {
	for _, peer := range peers {
		go func(peer message.GeneralPeer) {
			logrus.WithFields(logrus.Fields{
				"from":    d.me.Id,
				"to":      peer.Id,
				"content": msg.String(),
				"type":    msg.GetType()}).Trace("Sending message")
			outMsg := &message.GeneralMessageEvent{
				Message: msg,
				Sender:  d.me,
			}
			//<- ffchan.NewTimeoutSenderShort(d.PeerIns[peer.Id], outMsg, "bft").C
			d.PeerIns[peer.Id] <- outMsg
		}(peer)

	}
}

func (d *LocalGeneralPeerCommunicator) Unicast(msg message.GeneralMessage, peer message.GeneralPeer) {
	go func() {
		logrus.WithFields(logrus.Fields{
			"from":    d.me.Id,
			"to":      peer.Id,
			"content": msg.String(),
			"type":    msg.GetType()}).Trace("Sending message")
		outMsg := &message.GeneralMessageEvent{
			Message: msg,
			Sender:  d.me,
		}
		//<- ffchan.NewTimeoutSenderShort(d.PeerIns[peer.Id], outMsg, "lgp").C
		d.PeerIns[peer.Id] <- outMsg
	}()
}

func (d *LocalGeneralPeerCommunicator) GetPipeIn() chan *message.GeneralMessageEvent {
	return d.pipe
}

func (d *LocalGeneralPeerCommunicator) GetPipeOut() chan *message.GeneralMessageEvent {
	return d.pipe
}
