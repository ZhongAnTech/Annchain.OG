package plugin_test

import (
	"github.com/annchain/OG/message"
)

type LocalGeneralPeerCommunicator struct {
	Myid    int
	PeerIns []chan *message.GeneralMessageEvent
	pipe    chan *message.GeneralMessageEvent //pipeIn is the receiver of the outside messages
}

func (d *LocalGeneralPeerCommunicator) HandleIncomingMessage(msgEvent *message.GeneralMessageEvent) {
	d.pipe <- msgEvent
}

func NewLocalGeneralPeerCommunicator(myid int, incoming chan *message.GeneralMessageEvent, peers []chan *message.GeneralMessageEvent) *LocalGeneralPeerCommunicator {
	d := &LocalGeneralPeerCommunicator{
		PeerIns: peers,
		Myid:    myid,
		pipe:    incoming,
	}
	return d
}

func (d *LocalGeneralPeerCommunicator) Broadcast(msg message.GeneralMessage, peers []message.GeneralPeer) {
	for _, peer := range peers {
		go func(peer message.GeneralPeer) {
			//ffchan.NewTimeoutSenderShort(d.PeerIns[peer.Id], msg, "bft")
			d.PeerIns[peer.Id] <- &message.GeneralMessageEvent{
				Message: msg,
				Peer:    peer,
			}
		}(peer)
	}
}

func (d *LocalGeneralPeerCommunicator) Unicast(msg message.GeneralMessage, peer message.GeneralPeer) {
	go func() {
		//ffchan.NewTimeoutSenderShort(d.PeerIns[peer.Id], msg, "bft")
		d.PeerIns[peer.Id] <- &message.GeneralMessageEvent{
			Message: msg,
			Peer:    peer,
		}
	}()
}

func (d *LocalGeneralPeerCommunicator) GetPipeIn() chan *message.GeneralMessageEvent {
	return d.pipe
}

func (d *LocalGeneralPeerCommunicator) GetPipeOut() chan *message.GeneralMessageEvent {
	return d.pipe
}
