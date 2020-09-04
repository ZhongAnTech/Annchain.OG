package og

import (
	"github.com/annchain/OG/arefactor/consts"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/go-eventbus"
	"github.com/sirupsen/logrus"
	"time"
)

const EngineCheckIntervalSeconds = 1

type OgEngine struct {
	EventBus         *eventbus.EventBus
	Ledger           og_interface.Ledger
	CommunityManager CommunityManager
	NetworkId        string
	quit             chan bool

	// receive events
	myNewIncomingMessageEventChan chan *transport_interface.IncomingLetter // subscribe to NewIncomingMessageEvent
	myPeerJoinedEventChan         chan *og_interface.PeerJoinedEventArg
}

func (o *OgEngine) Receive(topic int, msg interface{}) error {
	switch consts.EventType(topic) {
	case consts.NewIncomingMessageEvent:
		o.myNewIncomingMessageEventChan <- msg.(*transport_interface.IncomingLetter)
	case consts.PeerJoinedEvent:
		o.myPeerJoinedEventChan <- msg.(*og_interface.PeerJoinedEventArg)
	default:
		return eventbus.ErrNotSupported
	}
	return nil
}

func (o *OgEngine) InitDefault() {
	o.myPeerJoinedEventChan = make(chan *og_interface.PeerJoinedEventArg, consts.DefaultEventQueueSize)
	o.myNewIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter, consts.DefaultEventQueueSize)
	o.quit = make(chan bool)
}

func (o *OgEngine) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	o.EventBus.Publish(int(consts.NewOutgoingMessageEvent), event)
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
		case <-o.myNewIncomingMessageEventChan:
			continue
			//switch message.OgMessageType(incomingLetter.Msg.MsgType) {
			//case message.OgMessageTypeHeightRequest:
			//	o.handleHeightRequest(incomingLetter)
			//case message.OgMessageTypeHeightResponse:
			//	o.handleHeightResponse(incomingLetter)
			//}
		case <-o.myPeerJoinedEventChan:
			continue
		//	o.handlePeerJoined(event)
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
