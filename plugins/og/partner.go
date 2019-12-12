package og

import (
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/message"
	"github.com/annchain/OG/og/communication"
	"github.com/sirupsen/logrus"
	"sync"
)

type OgPartner struct {
	incoming communication.GeneralPeerCommunicatorIncoming
	outgoing communication.GeneralPeerCommunicatorOutgoing
	quit     chan bool
	quitWg   sync.WaitGroup
}

func NewOgProcessor(
	incoming communication.GeneralPeerCommunicatorIncoming,
	outgoing communication.GeneralPeerCommunicatorOutgoing) *OgPartner {
	return &OgPartner{
		incoming: incoming,
		outgoing: outgoing,
		quit:     make(chan bool),
		quitWg:   sync.WaitGroup{},
	}
}

func (o OgPartner) Handle(msgEvent *message.GeneralMessageEvent) {
	o.incoming.GetPipeIn() <- msgEvent
}

func (o OgPartner) Run() {
	go func() {
		for {
			select {
			case <-o.quit:
				o.quitWg.Done()
				logrus.Debug("OgPartner quit")
				return
			case msgEvent := <-o.incoming.GetPipeOut():
				o.HandleOgMessage(msgEvent.Msg, msgEvent.Source)
			}
		}
	}()
}

func (o OgPartner) HandleOgMessage(message msg.OgMessage, source communication.OgPeer) {
	switch message.OgMessageType(message.GetType()) {
	case message.MessageTypePing:
		o.HandleMessagePing(source)
	case message.MessageTypePong:
		o.HandleMessagePong(source)
	case message.MessageTypeNewResource:
		o.HandleMessageNewResource(message, source)
	}
}

func (o OgPartner) HandleMessagePing(source communication.OgPeer) {
	logrus.Debugf("received ping from %d. Respond you a pong.", source.Id)
	o.outgoing.Unicast(&OgMessagePong{}, source)

}

func (o OgPartner) HandleMessagePong(source communication.OgPeer) {
	logrus.Debugf("received pong from %d.", source.Id)
}

func (o OgPartner) HandleMessageNewResource(message msg.OgMessage, identifier communication.OgPeer) {
	msg := message.(*message.MessageNewResource)
	// decode all resources and announce it to the receivers.
	for _, resource := range msg.Resources {
		logrus.Infof("Received resource: %s", hexutil.Encode(resource.ResourceContent))
	}
}

func (o OgPartner) SendMessagePing(peer communication.OgPeer) {
	o.outgoing.Unicast(&OgMessagePing{}, peer)
}
