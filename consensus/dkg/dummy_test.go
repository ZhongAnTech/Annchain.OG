package dkg

import (
	"github.com/sirupsen/logrus"
)

type dummyDkgPeerCommunicator struct {
	Myid    int
	Peers   []chan *DkgMessageEvent
	pipeIn  chan *DkgMessageEvent
	pipeOut chan *DkgMessageEvent
}

func NewDummyDkgPeerCommunicator(myid int, incoming chan *DkgMessageEvent, peers []chan *DkgMessageEvent) *dummyDkgPeerCommunicator {
	d := &dummyDkgPeerCommunicator{
		Peers:   peers,
		Myid:    myid,
		pipeIn:  incoming,
		pipeOut: make(chan *DkgMessageEvent, 10000), // must be big enough to avoid blocking issue
	}
	return d
}

func (d *dummyDkgPeerCommunicator) Broadcast(msg DkgMessage, peers []DkgPeer) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("me", d.Myid).Debug("broadcasting message")
		go func(peer DkgPeer) {
			//<-ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "dkg").C
			d.Peers[peer.Id] <- &DkgMessageEvent{
				Message: msg,
				Peer: DkgPeer{
					Id: d.Myid,
				},
			}
			logrus.WithField("type", msg.GetType()).Info("broadcast")
			//d.PeerPipeIns[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyDkgPeerCommunicator) Unicast(msg DkgMessage, peer DkgPeer) {
	go func(peerId int) {
		//<-ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "dkg").C
		d.Peers[peer.Id] <- &DkgMessageEvent{
			Message: msg,
			Peer: DkgPeer{
				Id: d.Myid,
			},
		}
		logrus.WithField("type", msg.GetType()).Info("unicast")
		//d.PeerPipeIns[peerId] <- msg
	}(peer.Id)
}

func (d *dummyDkgPeerCommunicator) GetPipeOut() chan *DkgMessageEvent {
	return d.pipeOut
}

func (d *dummyDkgPeerCommunicator) GetPipeIn() chan *DkgMessageEvent {
	return d.pipeIn
}

func (d *dummyDkgPeerCommunicator) Run() {
	go func() {
		for {
			v := <-d.pipeIn
			//<-ffchan.NewTimeoutSenderShort(d.pipeOut, v, "pc").C
			d.pipeOut <- v
		}
	}()
}
