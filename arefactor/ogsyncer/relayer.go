package ogsyncer

import (
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/gcache"
	"github.com/latifrons/goffchan"
	"github.com/sirupsen/logrus"
	"time"
)

// OgRelayer listens to new message event and relay it to the neighbours.
// It tries to dedup the messages so that duplicate messages won't be sent twice.
// OgRelayer will listen to broadcast messages and new block qced event
type OgRelayer struct {
	notificationCache             gcache.Cache
	newOutgoingMessageSubscribers []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent

	newBlockToldChan chan *og_interface.NewBlockToldEvent
	quit             chan bool
}

func (o *OgRelayer) InitDefault() {
	o.notificationCache = gcache.New(100).LRU().Expiration(time.Minute).Build()
	o.newBlockToldChan = make(chan *og_interface.NewBlockToldEvent)
	o.quit = make(chan bool)
}

// subscribe mine
func (n *OgRelayer) AddSubscriberNewOutgoingMessageEvent(sub transport_interface.NewOutgoingMessageEventSubscriber) {
	n.newOutgoingMessageSubscribers = append(n.newOutgoingMessageSubscribers, sub)
}

func (n *OgRelayer) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range n.newOutgoingMessageSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing ogrelayer "+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (o *OgRelayer) Start() {
	go o.eventLoop()
}

func (o *OgRelayer) Stop() {
	o.quit <- true
}

func (o *OgRelayer) Name() string {
	return "OgRelayer"
}

func (o *OgRelayer) eventLoop() {
	for {
		select {
		case <-o.quit:
			return
		case event := <-o.newBlockToldChan:
			o.handleNewBlockToldEvent(event)
		}
	}
}

func (o *OgRelayer) handleNewBlockToldEvent(event *og_interface.NewBlockToldEvent) {
	// broadcast to all my neighbours if not in the cache
	key := event.BlockContent.GetHash().HashString()
	if o.notificationCache.Has(key) {
		// already told others, ignore
		logrus.WithField("key", key).Debug("I have already told my neighbours")
		return
	}
	// broadcast list
	o.notifyNewOutgoingMessage()

}
