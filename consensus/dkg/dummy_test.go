package dkg

import "github.com/annchain/OG/ffchan"

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
		messageProviderChannel: make(chan DkgMessage),
	}
	return d
}

func (d *dummyDkgPeerCommunicator) Broadcast(msg DkgMessage, peers []PeerInfo) {
	for _, peer := range peers {
		go func(peer PeerInfo) {
			ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "dkg")
			//d.Peers[peer.MyIndex] <- msg
		}(peer)
	}
}

func (d *dummyDkgPeerCommunicator) Unicast(msg DkgMessage, peer PeerInfo) {
	go func() {
		ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "dkg")
		//d.Peers[peer.MyIndex] <- msg
	}()
}

func (d *dummyDkgPeerCommunicator) GetIncomingChannel() chan DkgMessage {
	return d.messageProviderChannel
}

func (d *dummyDkgPeerCommunicator) Run() {
	go func() {
		for {
			v := <-d.ReceiverChannel
			d.messageProviderChannel <- v
		}
	}()
}
