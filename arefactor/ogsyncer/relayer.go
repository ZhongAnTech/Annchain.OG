package ogsyncer

import (
	"github.com/annchain/OG/arefactor/consts"
	"github.com/annchain/OG/arefactor/dummy"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/gcache"
	"github.com/latifrons/go-eventbus"
	"github.com/sirupsen/logrus"
	"time"
)

// OgRelayer listens to new message event and relay it to the neighbours.
// It tries to dedup the messages so that duplicate messages won't be sent twice.
// OgRelayer will listen to broadcast messages and new block qced event
type OgRelayer struct {
	EventBus          *eventbus.EventBus
	notificationCache gcache.Cache

	newBlockProducedEventChan chan *og_interface.NewBlockProducedEventArg
	intsReceivedEventChan     chan *ogsyncer_interface.IntsReceivedEventArg
	quit                      chan bool
}

func (o *OgRelayer) Receive(topic int, msg interface{}) error {
	switch topic {
	case int(consts.NewBlockProducedEvent):
		o.newBlockProducedEventChan <- msg.(*og_interface.NewBlockProducedEventArg)
	case int(consts.IntsReceivedEvent):
		o.intsReceivedEventChan <- msg.(*ogsyncer_interface.IntsReceivedEventArg)
	default:
		return eventbus.ErrNotSupported
	}
	return nil
}

func (o *OgRelayer) InitDefault() {
	o.notificationCache = gcache.New(100).LRU().Expiration(time.Minute).Build()
	o.newBlockProducedEventChan = make(chan *og_interface.NewBlockProducedEventArg, consts.DefaultEventQueueSize)
	o.intsReceivedEventChan = make(chan *ogsyncer_interface.IntsReceivedEventArg, consts.DefaultEventQueueSize)
	o.quit = make(chan bool)
}

func (n *OgRelayer) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	n.EventBus.Publish(int(consts.NewOutgoingMessageEvent), event)
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
		case event := <-o.newBlockProducedEventChan:
			o.handleNewBlockProducedEvent(event)
		case event := <-o.intsReceivedEventChan:
			o.handleIntsReceivedEvent(event)
		}
	}
}

func (o *OgRelayer) handleNewBlockProducedEvent(event *og_interface.NewBlockProducedEventArg) {
	key := event.Block.GetHash().HashString()
	block := event.Block.(*dummy.IntArrayBlockContent)
	o.relayInts(key, ogsyncer_interface.MessageContentInt{
		Height:      block.Height,
		Step:        block.Step,
		PreviousSum: block.PreviousSum,
		MySum:       block.MySum,
		Submitter:   block.Submitter,
		Ts:          block.Ts,
	}, nil)
}

func (o *OgRelayer) relayInts(key string, ints ogsyncer_interface.MessageContentInt, excepts []string) {
	// broadcast to all my neighbours if not in the cache
	if o.notificationCache.Has(key) {
		// already told others, ignore
		logrus.WithField("key", key).Debug("I have already told my neighbours")
		return
	}
	_ = o.notificationCache.Set(key, true)
	// broadcast list
	msg := &ogsyncer_interface.OgAnnouncementNewInt{
		Ints: ints,
	}

	letter := &transport_interface.OutgoingLetter{
		Msg:             msg,
		SendType:        transport_interface.SendTypeBroadcast,
		CloseAfterSent:  false,
		ExceptMyself:    true,
		EndReceivers:    nil,
		ExceptReceivers: excepts,
	}

	o.notifyNewOutgoingMessage(letter)
}

func (o *OgRelayer) handleIntsReceivedEvent(event *ogsyncer_interface.IntsReceivedEventArg) {
	block := event.Ints
	v := dummy.IntArrayBlockContent{
		Height:      block.Height,
		Step:        block.Step,
		PreviousSum: block.PreviousSum,
		MySum:       block.MySum,
		Submitter:   block.Submitter,
		Ts:          block.Ts,
	}

	key := v.GetHash().HashString()
	o.relayInts(key, event.Ints, []string{event.From})
}
