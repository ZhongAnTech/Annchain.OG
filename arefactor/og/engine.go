package og

import (
	"fmt"
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/sirupsen/logrus"
	"time"
)

type OgEngine struct {
	Ledger                        Ledger
	CommunityManager              CommunityManager
	quit                          chan bool
	myNewIncomingMessageEventChan chan *transport_event.IncomingLetter                // subscribe to NewIncomingMessageEvent
	newOutgoingMessageSubscribers []transport_event.NewOutgoingMessageEventSubscriber // publish OutgoingMessage
}

func (o *OgEngine) InitDefault() {
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
		case msg := <-o.myNewIncomingMessageEventChan:
			fmt.Println(msg)
		case <-time.Tick(time.Second * 10):

		}
	}
}

func (o *OgEngine) StaticSetup() {

}
