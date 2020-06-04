package bouncer

import (
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/annchain/OG/ffchan"
	"github.com/sirupsen/logrus"
	"time"
)

// Bouncer maintain a list of nodes and pass the ball to the next
// Test p2p and event routing
// Bouncer listens to transport's new message event and consume it
type Bouncer struct {
	Id                            int
	Peers                         []string
	i                             int
	quit                          chan bool
	myNewIncomingMessageEventChan chan *transport_event.WireMessage
	newOutgoingMessageSubscribers []transport_event.NewOutgoingMessageEventSubscriber
	lastReceiveTime               time.Time
}

func (b *Bouncer) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{"value": b.i}
}

func (b *Bouncer) InitDefault() {
	b.myNewIncomingMessageEventChan = make(chan *transport_event.WireMessage)
	b.newOutgoingMessageSubscribers = []transport_event.NewOutgoingMessageEventSubscriber{}
	b.quit = make(chan bool)
}

func (b *Bouncer) RegisterSubscriberNewOutgoingMessageEvent(sub transport_event.NewOutgoingMessageEventSubscriber) {
	b.newOutgoingMessageSubscribers = append(b.newOutgoingMessageSubscribers, sub)
}

func (b *Bouncer) GetNewIncomingMessageEventChannel() chan *transport_event.WireMessage {
	return b.myNewIncomingMessageEventChan
}

func (b *Bouncer) loop() {
	for {
		logrus.Trace("bouncer loop round start")
		select {
		case <-b.quit:
			return
		case msg := <-b.myNewIncomingMessageEventChan:
			if msg.MsgType != 1 {
				panic("bad message")
			}
			bm := &BouncerMessage{}
			_, err := bm.UnmarshalMsg(msg.ContentBytes)
			if err != nil {
				panic(err)
			}

			// generate new message
			or := &transport_event.OutgoingRequest{
				Msg: &BouncerMessage{
					Value: 1,
				},
				SendType:     transport_event.SendTypeUnicast,
				EndReceivers: []string{b.Peers[(b.Id+1)%len(b.Peers)]},
			}
			for _, c := range b.newOutgoingMessageSubscribers {
				<-ffchan.NewTimeoutSender(c.GetNewOutgoingMessageEventChannel(), or, "bouncer send", 3000).C

			}
			b.i = bm.Value + 1
			b.lastReceiveTime = time.Now()
		case <-time.Tick(time.Second * 10):
			if b.lastReceiveTime.Add(time.Second * 10).After(time.Now()) {
				break
			}
			if b.Id == 0 {
				or := &transport_event.OutgoingRequest{
					Msg: &BouncerMessage{
						Value: 1,
					},
					SendType:     transport_event.SendTypeUnicast,
					EndReceivers: []string{b.Peers[1]},
				}
				for _, c := range b.newOutgoingMessageSubscribers {
					<-ffchan.NewTimeoutSender(c.GetNewOutgoingMessageEventChannel(), or, "bouncer send", 3000).C
				}
				b.i += 10
			}
		}
	}
}

func (b *Bouncer) Start() {
	go b.loop()
}

func (b *Bouncer) Stop() {
	close(b.quit)
}

func (b *Bouncer) Name() string {
	return "Bouncer"
}
