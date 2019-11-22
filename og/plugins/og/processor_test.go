package og

import (
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/types/msg"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

type DummyOgPeerCommunicator struct {
	Myid        int
	PeerPipeIns []chan *communicator.MessageEvent
	pipeIn      chan *communicator.MessageEvent
	pipeOut     chan *communicator.MessageEvent
}

func (o DummyOgPeerCommunicator) GetPipeIn() chan *communicator.MessageEvent {
	return o.pipeIn
}

func (o DummyOgPeerCommunicator) GetPipeOut() chan *communicator.MessageEvent {
	return o.pipeOut
}

func NewDummyOgPeerCommunicator(myid int, incoming chan *communicator.MessageEvent, peers []chan *communicator.MessageEvent) *DummyOgPeerCommunicator {
	d := &DummyOgPeerCommunicator{
		PeerPipeIns: peers,
		Myid:        myid,
		pipeIn:      incoming,
		pipeOut:     make(chan *communicator.MessageEvent, 100), // must be big enough to avoid blocking issue
	}
	return d
}

func (o DummyOgPeerCommunicator) Broadcast(msg msg.TransportableMessage, peers []communicator.PeerIdentifier) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("me", o.Myid).Debug("broadcasting message")
		go func(peer communicator.PeerIdentifier) {
			ffchan.NewTimeoutSenderShort(o.PeerPipeIns[peer.Id], msg, "dkg")
			//d.PeerPipeIns[peer.Id] <- msg
		}(peer)
	}
}

func (o DummyOgPeerCommunicator) Unicast(msg msg.TransportableMessage, peer communicator.PeerIdentifier) {
	logrus.Debug("unicasting by DummyOgPeerCommunicator")
	go func() {
		//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
		o.PeerPipeIns[peer.Id] <- &communicator.MessageEvent{
			Msg:    msg,
			Source: communicator.PeerIdentifier{Id: o.Myid},
		}
	}()
}

func (d *DummyOgPeerCommunicator) Run() {
	logrus.Info("DummyOgPeerCommunicator running")
	go func() {
		for {
			v := <-d.pipeIn
			//vv := v.Message.(bft.BftMessage)
			logrus.WithField("type", v.Msg.GetType()).Debug("DummyOgPeerCommunicator received a message")
			d.pipeOut <- v
		}
	}()
}

func TestPingPong(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	total := 2
	// init two OG peers's In channel
	peerChans := make([]chan *communicator.MessageEvent, total)
	peerInfos := make([]communicator.PeerIdentifier, total)

	// build communication channels
	for i := 0; i < total; i++ {
		peerInfos[i] = communicator.PeerIdentifier{Id: i}
		peerChans[i] = make(chan *communicator.MessageEvent, 10)
	}

	processors := make([]*OgProcessor, total)

	// build peer communicator
	for i := 0; i < total; i++ {
		communicator := NewDummyOgPeerCommunicator(i, peerChans[i], peerChans)
		communicator.Run()
		processors[i] = NewOgProcessor(communicator, communicator)
		processors[i].Run()
	}

	// send ping
	logrus.Info("Sending ping")
	processors[0].SendMessagePing(peerInfos[1])
	time.Sleep(time.Minute * 4)

}
