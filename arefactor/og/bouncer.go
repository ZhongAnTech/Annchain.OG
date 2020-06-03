package og

import (
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/annchain/OG/ffchan"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Bouncer maintain a list of nodes and pass the ball to the next
// Test p2p and event routing
// Bouncer listens to transport's new message event and consume it
type Bouncer struct {
	Id                           int
	Peers                        []string
	i                            int
	quit                         chan bool
	transportEventReceiver       chan *transport_event.WireMessage
	newOutgoingMessageSubscriber []transport_event.NewOutgoingMessageEventSubscriber
}

func (b *Bouncer) InitDefault() {
	b.transportEventReceiver = make(chan *transport_event.WireMessage)
	b.newOutgoingMessageSubscriber = []transport_event.NewOutgoingMessageEventSubscriber{}
	b.quit = make(chan bool)
}

func (b *Bouncer) RegisterSubscriberNewOutgoingMessageEvent(sub transport_event.NewOutgoingMessageEventSubscriber) {
	b.newOutgoingMessageSubscriber = append(b.newOutgoingMessageSubscriber, sub)
}

func (b *Bouncer) GetNewIncomingMessageEventChannel() chan *transport_event.WireMessage {
	return b.transportEventReceiver
}

func (b *Bouncer) Start() {
	for {
		logrus.Trace("bouncer loop round start")
		select {
		case <-b.quit:
			return
		case msg := <-b.transportEventReceiver:
			if msg.MsgType != 1 {
				panic("bad msg")
			}
			bm := &BouncerMessage{}
			_, err := bm.UnmarshalMsg(msg.ContentBytes)
			if err != nil {
				panic(err)
			}

			// generate new msg
			or := &transport_event.OutgoingRequest{
				Msg: &transport_event.OutgoingMsg{
					Typev:    1,
					SenderId: strconv.Itoa(b.Id),
					Content: &BouncerMessage{
						Value: bm.Value + 1,
					},
				},
				SendType:     transport_event.SendTypeUnicast,
				EndReceivers: []string{b.Peers[(b.Id+1)%len(b.Peers)]},
			}
			for _, c := range b.newOutgoingMessageSubscriber {
				ffchan.NewTimeoutSender(c.GetNewOutgoingMessageEventChannel(), or, "bouncer send", 3000)
			}
		case <-time.Tick(time.Second * 10):
			if b.Id == 0 {
				or := &transport_event.OutgoingRequest{
					Msg: &transport_event.OutgoingMsg{
						Typev:    1,
						SenderId: strconv.Itoa(b.Id),
						Content: &BouncerMessage{
							Value: b.i * 10,
						},
					},
					SendType:     transport_event.SendTypeUnicast,
					EndReceivers: []string{b.Peers[1]},
				}
				for _, c := range b.newOutgoingMessageSubscriber {
					ffchan.NewTimeoutSender(c.GetNewOutgoingMessageEventChannel(), or, "bouncer send", 3000)
				}
			}
		}
	}
}

func (b *Bouncer) Stop() {
	close(b.quit)
}

func (b *Bouncer) Name() string {
	return "Bouncer"
}
