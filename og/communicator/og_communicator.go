package communicator

import (
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/types/msg"
	"github.com/sirupsen/logrus"
)

type DummyOgPeerCommunicator struct {
	Myid        int
	PeerPipeIns []chan msg.TransportableMessage
	pipeIn      chan msg.TransportableMessage
	pipeOut     chan msg.TransportableMessage
}

func NewDummyOgPeerCommunicator(myid int, incoming chan msg.TransportableMessage, peers []chan msg.TransportableMessage) *DummyOgPeerCommunicator {
	d := &DummyOgPeerCommunicator{
		PeerPipeIns: peers,
		Myid:        myid,
		pipeIn:      incoming,
		pipeOut:     make(chan msg.TransportableMessage, 10000), // must be big enough to avoid blocking issue
	}
	return d
}

func (o DummyOgPeerCommunicator) Broadcast(msg msg.TransportableMessage, peers []OgPeer) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("me", o.Myid).Debug("broadcasting message")
		go func(peer OgPeer) {
			ffchan.NewTimeoutSenderShort(o.PeerPipeIns[peer.Id], msg, "dkg")
			//d.PeerPipeIns[peer.Id] <- msg
		}(peer)
	}
}

func (o DummyOgPeerCommunicator) Unicast(msg msg.TransportableMessage, peer OgPeer) {
	logrus.Debug("unicasting by DummyOgPeerCommunicator")
	go func() {
		//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
		o.PeerPipeIns[peer.Id] <- msg
	}()
}

func (o DummyOgPeerCommunicator) GetPipeIn() chan msg.TransportableMessage {
	return o.pipeIn
}

func (o DummyOgPeerCommunicator) GetPipeOut() chan msg.TransportableMessage {
	return o.pipeOut
}
