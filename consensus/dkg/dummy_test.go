package dkg

import "github.com/annchain/OG/ffchan"

type dummyDkgPeerCommunicator struct {
	Myid                   int
	Peers                  []chan DkgMessage
	ReceiverChannel        chan DkgMessage
	messageProviderChannel chan DkgMessage
}

func (d *dummyDkgPeerCommunicator) Broadcast(msg DkgMessage, peers []PeerInfo) {
	for _, peer := range peers {
		go func(peer PeerInfo) {
			ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "bft")
			//d.Peers[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyDkgPeerCommunicator) Unicast(msg DkgMessage, peer PeerInfo) {
	go func() {
		ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "bft")
		//d.Peers[peer.Id] <- msg
	}()
}

func (d *dummyDkgPeerCommunicator) GetIncomingChannel() chan DkgMessage {
	return d.messageProviderChannel
}

func NewDummyDkgPeerCommunicator(myid int, incoming chan DkgMessage, peers []chan DkgMessage) *dummyDkgPeerCommunicator {
	d := &dummyDkgPeerCommunicator{
		Peers:           peers,
		Myid:            myid,
		ReceiverChannel: incoming,
	}
	return d
}
