package ogcore

import (
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/message"
	"github.com/annchain/OG/ogcore/model"
	"github.com/sirupsen/logrus"
	"sync"
)

type OgPartner struct {
	Config       OgProcessorConfig
	PeerOutgoing communication.OgPeerCommunicatorOutgoing
	PeerIncoming communication.OgPeerCommunicatorIncoming
	EventBus     EventBus

	// og protocols
	StatusProvider OgStatusProvider
	OgCore         *OgCore

	// partner should connect the backend service by announce events
	quit   chan bool
	quitWg sync.WaitGroup
}

func (a *OgPartner) InitDefault() {
	a.quitWg = sync.WaitGroup{}
	a.quit = make(chan bool)
}

func (o *OgPartner) Start() {
	go o.loop()
	logrus.Info("OgPartner Started")

}

func (ap *OgPartner) Stop() {
	ap.quit <- true
	ap.quitWg.Wait()
	logrus.Debug("OgPartner Stopped")
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
	//case message.OgMessageTypeStatus:
	//	o.HandleMessageStatus(msgEvent)
	case message.OgMessageTypePing:
		o.HandleMessagePing(msgEvent)
	case message.OgMessageTypePong:
		o.HandleMessagePong(msgEvent)
	case message.OgMessageTypeQueryStatusRequest:
		o.HandleMessageQueryStatusRequest(msgEvent)
	case message.OgMessageTypeQueryStatusResponse:
		o.HandleMessageQueryStatusResponse(msgEvent)
	//case message.OgMessageTypeNewResource:
	//	o.HandleMessageNewResource(msgEvent)
	default:
		logrus.WithField("msg", msgEvent.Message).Warn("unsupported og message type")
	}
}

func (o *OgPartner) FireEvent(event eventbus.Event) {
	if o.EventBus != nil {
		o.EventBus.Route(event)
	}
}

func (o *OgPartner) HandleMessagePing(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	logrus.Debugf("received ping from %d. Respond you a pong.", source.Id)
	o.FireEvent(&events.PingReceivedEvent{})
	o.PeerOutgoing.Unicast(&message.OgMessagePong{}, source)

}

func (o *OgPartner) HandleMessagePong(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	logrus.Debugf("received pong from %d.", source.Id)
	o.FireEvent(&events.PongReceivedEvent{})
	o.PeerOutgoing.Unicast(&message.OgMessagePing{}, source)
}

func (o *OgPartner) HandleMessageQueryStatusRequest(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	logrus.Debugf("received QueryStatusRequest from %d.", source.Id)
	status := o.StatusProvider.GetCurrentOgStatus()
	o.PeerOutgoing.Unicast(&message.OgMessageQueryStatusResponse{
		ProtocolVersion: status.ProtocolVersion,
		NetworkId:       status.NetworkId,
		CurrentBlock:    status.CurrentBlock,
		GenesisBlock:    status.GenesisBlock,
		CurrentHeight:   status.CurrentHeight,
	}, source)
}

func (o *OgPartner) HandleMessageQueryStatusResponse(msgEvent *communication.OgMessageEvent) {
	// must be a response message
	resp, ok := msgEvent.Message.(*message.OgMessageQueryStatusResponse)
	if !ok {
		logrus.Warn("bad format: OgMessageQueryStatusResponse")
		return
	}
	source := msgEvent.Peer
	logrus.Debugf("received QueryStatusRequest from %d.", source.Id)
	statusData := model.OgStatusData{
		ProtocolVersion: resp.ProtocolVersion,
		NetworkId:       resp.NetworkId,
		CurrentBlock:    resp.CurrentBlock,
		GenesisBlock:    resp.GenesisBlock,
		CurrentHeight:   resp.CurrentHeight,
	}

	o.OgCore.HandleStatusData(statusData)
	o.FireEvent(&events.QueryStatusResponseReceivedEvent{
		StatusData: statusData,
	})
}

//func (o *OgPartner) HandleMessageNewResource(msgEvent *communication.OgMessageEvent) {
//	msg := msgEvent.Message.(*message.OgMessageNewResource)
//	// decode all resources and announce it to the receivers.
//	for _, resource := range msg.Resources {
//		logrus.Infof("Received resource: %s", hexutil.Encode(resource.ResourceContent))
//	}
//}

func (o *OgPartner) SendMessagePing(peer communication.OgPeer) {
	o.PeerOutgoing.Unicast(&message.OgMessagePing{}, peer)
}

func (o *OgPartner) SendMessageQueryStatusRequest(peer communication.OgPeer) {
	o.PeerOutgoing.Unicast(&message.OgMessageQueryStatusRequest{}, peer)
}

type OgProcessorConfig struct {
}
