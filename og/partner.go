package og

import (
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/og/communication"
	"github.com/annchain/OG/og/message"
	"github.com/sirupsen/logrus"
	"sync"
)

type OgPartner struct {
	PeerOutgoing communication.OgPeerCommunicatorOutgoing
	PeerIncoming communication.OgPeerCommunicatorIncoming

	quit   chan bool
	quitWg sync.WaitGroup
}

func (a *OgPartner) InitDefault() {
	a.quitWg = sync.WaitGroup{}
	a.quit = make(chan bool)
}

func (o *OgPartner) Start() {
	go func() {

	}()
}

func (o *OgPartner) loop() {
	for {
		select {
		case <-o.quit:
			o.quitWg.Done()
			logrus.Debug("OgPartner quit")
			return
		case msgEvent := <-o.PeerIncoming.GetPipeOut():
			o.HandleOgMessage(msgEvent)
		}
	}
}

func (o *OgPartner) HandleOgMessage(msgEvent *communication.OgMessageEvent) {

	switch msgEvent.Message.GetType() {
	case message.OgMessageTypeStatus:
		o.HandleMessageStatus(msgEvent)
	case message.OgMessageTypePing:
		o.HandleMessagePing(msgEvent)
	case message.OgMessageTypePong:
		o.HandleMessagePong(msgEvent)
	case message.OgMessageTypeNewResource:
		o.HandleMessageNewResource(msgEvent)
	default:
		logrus.WithField("msg", msgEvent.Message).Warn("unsupported og message type")
	}
}

func (o *OgPartner) HandleMessagePing(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	logrus.Debugf("received ping from %d. Respond you a pong.", source.Id)
	o.PeerOutgoing.Unicast(&message.OgMessagePong{}, source)

}

func (o *OgPartner) HandleMessagePong(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	logrus.Debugf("received pong from %d.", source.Id)
}

func (o *OgPartner) HandleMessageNewResource(msgEvent *communication.OgMessageEvent) {
	msg := msgEvent.Message.(*message.OgMessageNewResource)
	// decode all resources and announce it to the receivers.
	for _, resource := range msg.Resources {
		logrus.Infof("Received resource: %s", hexutil.Encode(resource.ResourceContent))
	}
}

func (o *OgPartner) SendMessagePing(peer communication.OgPeer) {
	o.PeerOutgoing.Unicast(&message.OgMessagePing{}, peer)
}

func (a *OgPartner) HandleMessageStatus(msgEvent *communication.OgMessageEvent) {

}
