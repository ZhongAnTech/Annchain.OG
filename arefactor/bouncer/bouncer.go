package bouncer

import (
	"github.com/annchain/OG/arefactor/consts"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/go-eventbus"
	"github.com/sirupsen/logrus"
	"time"
)

// Bouncer maintain a list of nodes and pass the ball to the next
// Test p2p and event routing
// Bouncer listens to transport's new message event and consume it
type Bouncer struct {
	EventBus                      *eventbus.EventBus
	Id                            int
	Peers                         []string
	i                             int
	quit                          chan bool
	myNewIncomingMessageEventChan chan *transport_interface.IncomingLetter
	lastReceiveTime               time.Time
}

func (b *Bouncer) Receive(topic int, msg interface{}) error {
	switch consts.EventType(topic) {
	case consts.NewIncomingMessageEvent:
		b.myNewIncomingMessageEventChan <- msg.(*transport_interface.IncomingLetter)
	default:
		return eventbus.ErrNotSupported
	}
	return nil
}

func (b *Bouncer) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{"value": b.i}
}

func (b *Bouncer) InitDefault() {
	b.myNewIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter, consts.DefaultEventQueueSize)
	b.quit = make(chan bool)
}

func (b *Bouncer) loop() {
	for {
		logrus.Trace("bouncer loop round start")
		select {
		case <-b.quit:
			return
		case letter := <-b.myNewIncomingMessageEventChan:
			if letter.Msg.MsgType != 1 {
				panic("bad message")
			}
			bm := &BouncerMessage{}
			err := bm.FromBytes(letter.Msg.ContentBytes)
			if err != nil {
				panic(err)
			}

			// generate new message
			outgoingLetter := &transport_interface.OutgoingLetter{
				Msg: &BouncerMessage{
					Value: 1,
				},
				SendType:       transport_interface.SendTypeUnicast,
				CloseAfterSent: false,
				ExceptMyself:   true,
				EndReceivers:   []string{b.Peers[(b.Id+1)%len(b.Peers)]},
			}

			b.EventBus.Publish(int(consts.NewOutgoingMessageEvent), outgoingLetter)

			b.i = bm.Value + 1
			b.lastReceiveTime = time.Now()
		case <-time.Tick(time.Second * 10):
			if b.lastReceiveTime.Add(time.Second * 10).After(time.Now()) {
				break
			}
			if b.Id == 0 {
				outgoingLetter := &transport_interface.OutgoingLetter{
					Msg: &BouncerMessage{
						Value: 1,
					},
					SendType:       transport_interface.SendTypeUnicast,
					CloseAfterSent: false,
					ExceptMyself:   true,
					EndReceivers:   []string{b.Peers[1]},
				}
				b.EventBus.Publish(int(consts.NewOutgoingMessageEvent), outgoingLetter)
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
