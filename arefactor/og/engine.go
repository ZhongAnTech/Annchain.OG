package og

import (
	"github.com/annchain/OG/arefactor/og/message"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/OG/og_interface"
	"github.com/latifrons/goffchan"
	"github.com/sirupsen/logrus"
	"time"
)

const EngineCheckIntervalSeconds = 1

type OgEngine struct {
	Ledger           og_interface.Ledger
	CommunityManager CommunityManager
	NetworkId        string
	quit             chan bool

	// receive events
	myNewIncomingMessageEventChan chan *transport_interface.IncomingLetter // subscribe to NewIncomingMessageEvent
	myPeerJoinedEventChan         chan *og_interface.PeerJoinedEvent

	// publish events
	newOutgoingMessageSubscribers []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent
	newHeightDetectedSubscribers  []og_interface.NewHeightDetectedEventSubscriber         // a message need to be sent

}

func (o *OgEngine) InitDefault() {
	o.myPeerJoinedEventChan = make(chan *og_interface.PeerJoinedEvent)
	o.myNewIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter)
	o.newOutgoingMessageSubscribers = []transport_interface.NewOutgoingMessageEventSubscriber{}
	o.quit = make(chan bool)
}

func (o *OgEngine) EventChannelPeerJoined() chan *og_interface.PeerJoinedEvent {
	return o.myPeerJoinedEventChan
}

func (o *OgEngine) NewIncomingMessageEventChannel() chan *transport_interface.IncomingLetter {
	return o.myNewIncomingMessageEventChan
}

func (o *OgEngine) AddSubscriberNewOutgoingMessageEvent(sub transport_interface.NewOutgoingMessageEventSubscriber) {
	o.newOutgoingMessageSubscribers = append(o.newOutgoingMessageSubscribers, sub)
}

func (o *OgEngine) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range o.newOutgoingMessageSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing engine"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (o *OgEngine) AddSubscriberNewHeightDetectedEvent(sub og_interface.NewHeightDetectedEventSubscriber) {
	o.newHeightDetectedSubscribers = append(o.newHeightDetectedSubscribers, sub)
}

func (o *OgEngine) notifyNewHeightDetected(event *og_interface.NewHeightDetectedEvent) {
	for _, subscriber := range o.newHeightDetectedSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewHeightDetectedEventChannel(), event, "heightdetected"+subscriber.Name()).C
		//subscriber.NewHeightDetectedEventChannel() <- event
	}
}

func (o *OgEngine) Start() {
	go o.loop()
}

func (o *OgEngine) Stop() {
	close(o.quit)
}

func (o *OgEngine) Name() string {
	return "OgEngine"
}

func (o *OgEngine) loop() {
	// load committee
	//height := o.Ledger.CurrentHeight()
	//committee := o.Ledger.CurrentCommittee()
	timer := time.NewTimer(time.Second * time.Duration(EngineCheckIntervalSeconds))

	for {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(time.Second * time.Duration(EngineCheckIntervalSeconds))
		select {
		case <-o.quit:
			return
		case incomingLetter := <-o.myNewIncomingMessageEventChan:
			switch message.OgMessageType(incomingLetter.Msg.MsgType) {
			case message.OgMessageTypeHeightRequest:
				o.handleHeightRequest(incomingLetter)
			case message.OgMessageTypeHeightResponse:
				o.handleHeightResponse(incomingLetter)
			}
		case event := <-o.myPeerJoinedEventChan:
			o.handlePeerJoined(event)
		case <-timer.C:
			logrus.Trace("routing check in engine")
		}
	}
}

func (o *OgEngine) StaticSetup() {

}

func (o *OgEngine) CurrentHeight() int64 {
	return o.Ledger.CurrentHeight()
}

func (o *OgEngine) GetNetworkId() string {
	return o.NetworkId
}

func (o *OgEngine) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{}
}
