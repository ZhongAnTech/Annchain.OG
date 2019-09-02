package dkg

import (
	"github.com/annchain/OG/ffchan"
	"github.com/sirupsen/logrus"
)

var TestNodes = 70

type dummyDkgPeerCommunicator struct {
	Myid                   int
	Peers                  []chan DkgMessage
	ReceiverChannel        chan DkgMessage
	messageProviderChannel chan DkgMessage
}

func NewDummyDkgPeerCommunicator(myid int, incoming chan DkgMessage, peers []chan DkgMessage) *dummyDkgPeerCommunicator {
	d := &dummyDkgPeerCommunicator{
		Peers:                  peers,
		Myid:                   myid,
		ReceiverChannel:        incoming,
		messageProviderChannel: make(chan DkgMessage, 10000), // must be big enough to avoid blocking issue
	}
	return d
}

func (d *dummyDkgPeerCommunicator) Broadcast(msg DkgMessage, peers []PeerInfo) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("me", d.Myid).Debug("broadcasting message")
		go func(peer PeerInfo) {
			ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "dkg")
			//d.Peers[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyDkgPeerCommunicator) Unicast(msg DkgMessage, peer PeerInfo) {
	go func(peerId int) {
		ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "dkg")
		//d.Peers[peerId] <- msg
	}(peer.Id)
}

func (d *dummyDkgPeerCommunicator) GetIncomingChannel() chan DkgMessage {
	return d.messageProviderChannel
}

func (d *dummyDkgPeerCommunicator) Run() {
	go func() {
		for {
			v := <-d.ReceiverChannel
			ffchan.NewTimeoutSenderShort(d.messageProviderChannel, v,"pc")
			//d.messageProviderChannel <- v
		}
	}()
}
