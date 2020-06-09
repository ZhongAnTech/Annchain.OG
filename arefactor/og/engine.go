package og

import (
	"github.com/annchain/OG/arefactor/og/message"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/sirupsen/logrus"
	"time"
)

type OgEngine struct {
	Ledger           Ledger
	CommunityManager CommunityManager
	NetworkId        string
	quit             chan bool

	// receive events
	myNewIncomingMessageEventChan chan *transport_event.IncomingLetter // subscribe to NewIncomingMessageEvent
	myPeerJoinedEventChan         chan *og_interface.PeerJoinedEvent

	// publish events
	newOutgoingMessageSubscribers []transport_event.NewOutgoingMessageEventSubscriber // a message need to be sent
	newHeightDetectedSubscribers  []og_interface.NewHeightDetectedEventSubscriber     // a message need to be sent

}

func (o *OgEngine) PeerJoinedEventChannel() chan *og_interface.PeerJoinedEvent {
	return o.myPeerJoinedEventChan
}

func (o *OgEngine) CurrentHeight() int64 {
	return o.Ledger.CurrentHeight()
}

func (o *OgEngine) GetNetworkId() string {
	return o.NetworkId
}

func (o *OgEngine) InitDefault() {
	o.myPeerJoinedEventChan = make(chan *og_interface.PeerJoinedEvent)
	o.myNewIncomingMessageEventChan = make(chan *transport_event.IncomingLetter)
	o.newOutgoingMessageSubscribers = []transport_event.NewOutgoingMessageEventSubscriber{}
	o.quit = make(chan bool)
}

func (o *OgEngine) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{}
}

func (o *OgEngine) AddSubscriberNewOutgoingMessageEvent(sub transport_event.NewOutgoingMessageEventSubscriber) {
	o.newOutgoingMessageSubscribers = append(o.newOutgoingMessageSubscribers, sub)
}

func (o *OgEngine) notifyNewOutgoingMessage(event *transport_event.OutgoingLetter) {
	for _, subscriber := range o.newOutgoingMessageSubscribers {
		//goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing")
		subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (o *OgEngine) AddSubscriberNewHeightDetectedEvent(sub og_interface.NewHeightDetectedEventSubscriber) {
	o.newHeightDetectedSubscribers = append(o.newHeightDetectedSubscribers, sub)
}

func (o *OgEngine) notifyNewHeightDetected(event *og_interface.NewHeightDetectedEvent) {
	for _, subscriber := range o.newHeightDetectedSubscribers {
		//goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing")
		subscriber.NewHeightDetectedEventChannel() <- event
	}
}

func (o *OgEngine) GetNewIncomingMessageEventChannel() chan *transport_event.IncomingLetter {
	return o.myNewIncomingMessageEventChan
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

	for {
		logrus.Trace("ogEngine loop round start")
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
		case <-time.Tick(time.Second * 10):

		}
	}
}

func (o *OgEngine) StaticSetup() {

}

func (o *OgEngine) handleHeightRequest(letter *transport_event.IncomingLetter) {
	// report my height info
	resp := message.OgMessageHeightResponse{
		Height: o.CurrentHeight(),
	}
	o.notifyNewOutgoingMessage(&transport_event.OutgoingLetter{
		Msg:            &resp,
		SendType:       transport_event.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{letter.From},
	})
}

func (o *OgEngine) handleHeightResponse(letter *transport_event.IncomingLetter) {
	// if we get a pong with close flag, the target peer is dropping us.
	m := &message.OgMessageHeightResponse{}
	_, err := m.UnmarshalMsg(letter.Msg.ContentBytes)
	if err != nil {
		logrus.WithField("type", m.GetType()).WithError(err).Warn("bad message")
	}
	if m.Height <= o.CurrentHeight() {
		// already known that\
		logrus.WithField("theirHeight", m.Height).
			WithField("myHeight", o.CurrentHeight()).
			WithField("from", letter.From).Debug("detected a new height but is not higher than mine")
		return
	}
	// get their height response, announce a new height received event
	o.notifyNewHeightDetected(&og_interface.NewHeightDetectedEvent{
		Height: m.Height,
		PeerId: letter.From,
	})
	logrus.WithField("height", m.Height).WithField("from", letter.From).Debug("detected a new height")
}

func (o *OgEngine) handlePeerJoined(event *og_interface.PeerJoinedEvent) {
	// detect height
	m := &message.OgMessageHeightRequest{}
	o.notifyNewOutgoingMessage(&transport_event.OutgoingLetter{
		Msg:            m,
		SendType:       transport_event.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{event.PeerId},
	})
}
