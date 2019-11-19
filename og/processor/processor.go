package processor

import (
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/types/msg"
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

func (o OgProcessor) UnmarshalMessage(message msg.BinaryMessage) msg.TransportableMessage {

}

func (o OgProcessor) Handle(msgEvent *communicator.MessageEvent) {
	o.incoming.GetPipeIn() <- msgEvent
}

func (o OgProcessor) Run() {
	for {
		select {
		case <-o.quit:
			o.quitWg.Done()
			return
		case msgEvent := <-o.incoming.GetPipeIn():
			o.HandleOgMessage(msgEvent.Msg, msgEvent.Source)
		}
	}
}

func (o OgProcessor) HandleOgMessage(message msg.TransportableMessage, source communicator.PeerIdentifier) {
	switch ogmessage.OgMessageType(message.GetType()) {
	case ogmessage.MessageTypePing:
		o.outgoing.Unicast(&ogmessage.MessagePong{}, source)
	}
}
