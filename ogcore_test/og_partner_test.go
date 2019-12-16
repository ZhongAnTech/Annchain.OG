package og_test

import (
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/message"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

type DummyOgPeerCommunicator struct {
	Myid        int
	PeerPipeIns []chan *communication.OgMessageEvent
	pipeIn      chan *communication.OgMessageEvent
	pipeOut     chan *communication.OgMessageEvent
}

func (o DummyOgPeerCommunicator) GetPipeIn() chan *communication.OgMessageEvent {
	return o.pipeIn
}

func (o DummyOgPeerCommunicator) GetPipeOut() chan *communication.OgMessageEvent {
	return o.pipeOut
}

func NewDummyOgPeerCommunicator(myid int, incoming chan *communication.OgMessageEvent, peers []chan *communication.OgMessageEvent) *DummyOgPeerCommunicator {
	d := &DummyOgPeerCommunicator{
		PeerPipeIns: peers,
		Myid:        myid,
		pipeIn:      incoming,
		pipeOut:     make(chan *communication.OgMessageEvent, 100), // must be big enough to avoid blocking issue
	}
	return d
}

func (o DummyOgPeerCommunicator) Broadcast(msg message.OgMessage, peers []communication.OgPeer) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("me", o.Myid).Debug("broadcasting message")
		go func(peer communication.OgPeer) {
			ffchan.NewTimeoutSenderShort(o.PeerPipeIns[peer.Id], msg, "dkg")
			//d.PeerPipeIns[peer.Id] <- msg
		}(peer)
	}
}

func (o DummyOgPeerCommunicator) Unicast(msg message.OgMessage, peer communication.OgPeer) {
	logrus.Debug("unicasting by DummyOgPeerCommunicator")
	go func() {
		//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
		o.PeerPipeIns[peer.Id] <- &communication.OgMessageEvent{
			Message: msg,
			Peer:    communication.OgPeer{Id: o.Myid},
		}
	}()
}

func (d *DummyOgPeerCommunicator) Run() {
	logrus.Info("DummyOgPeerCommunicator running")
	go func() {
		for {
			v := <-d.pipeIn
			//vv := v.Message.(bft.BftMessage)
			logrus.WithField("type", v.Message.GetType()).Debug("DummyOgPeerCommunicator received a message")
			d.pipeOut <- v
		}
	}()
}

func TestPingPong(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	total := 2
	// init two OG peers's In channel
	peerChans := make([]chan *communication.OgMessageEvent, total)
	peerInfos := make([]communication.OgPeer, total)

	// build communication channels
	for i := 0; i < total; i++ {
		peerInfos[i] = communication.OgPeer{Id: i}
		peerChans[i] = make(chan *communication.OgMessageEvent, 10)
	}

	processors := make([]*ogcore.OgPartner, total)

	// build peer communicator
	for i := 0; i < total; i++ {
		communicator := NewDummyOgPeerCommunicator(i, peerChans[i], peerChans)
		communicator.Run()

		processor := &ogcore.OgPartner{
			PeerOutgoing: communicator,
			PeerIncoming: communicator,
		}

		processors[i] = processor
		processors[i].Start()
	}

	// send ping
	logrus.Debug("Sending ping")
	processors[0].SendMessagePing(peerInfos[1])
	time.Sleep(time.Minute * 4)

}
