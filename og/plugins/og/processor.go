package og

import (
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/types/msg"
	"github.com/sirupsen/logrus"
	"sync"
)

type OgProcessor struct {
	incoming communicator.OgPeerCommunicatorIncoming
	outgoing communicator.OgPeerCommunicatorOutgoing
	quit     chan bool
	quitWg   sync.WaitGroup
}

func NewOgProcessor(
	incoming communicator.OgPeerCommunicatorIncoming,
	outgoing communicator.OgPeerCommunicatorOutgoing) *OgProcessor {
	return &OgProcessor{
		incoming: incoming,
		outgoing: outgoing,
		quit:     make(chan bool),
		quitWg:   sync.WaitGroup{},
	}
}

func (o OgProcessor) Handle(msgEvent *communicator.MessageEvent) {
	o.incoming.GetPipeIn() <- msgEvent
}

func (o OgProcessor) Run() {
	go func() {
		for {
			select {
			case <-o.quit:
				o.quitWg.Done()
				logrus.Debug("OgProcessor quit")
				return
			case msgEvent := <-o.incoming.GetPipeOut():
				o.HandleOgMessage(msgEvent.Msg, msgEvent.Source)
			}
		}
	}()
}

func (o OgProcessor) HandleOgMessage(message msg.OgMessage, source communicator.PeerIdentifier) {
	switch message.OgMessageType(message.GetType()) {
	case message.MessageTypePing:
		o.HandleMessagePing(source)
	case message.MessageTypePong:
		o.HandleMessagePong(source)
	case message.MessageTypeNewResource:
		o.HandleMessageNewResource(message, source)
	}
}

func (o OgProcessor) HandleMessagePing(source communicator.PeerIdentifier) {
	logrus.Debugf("received ping from %d. Respond you a pong.", source.Id)
	o.outgoing.Unicast(&MessagePong{}, source)

}

func (o OgProcessor) HandleMessagePong(source communicator.PeerIdentifier) {
	logrus.Debugf("received pong from %d.", source.Id)
}

func (o OgProcessor) HandleMessageNewResource(message msg.OgMessage, identifier communicator.PeerIdentifier) {
	msg := message.(*message.MessageNewResource)
	// decode all resources and announce it to the receivers.
	for _, resource := range msg.Resources {
		logrus.Infof("Received resource: %s", hexutil.Encode(resource.ResourceContent))
	}
}

func (o OgProcessor) SendMessagePing(peer communicator.PeerIdentifier) {
	o.outgoing.Unicast(&MessagePing{}, peer)
}
