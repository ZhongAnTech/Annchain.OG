package ogsyncer

import (
	"github.com/annchain/OG/arefactor/consts"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/go-eventbus"
	"github.com/sirupsen/logrus"
)

// OgReceiver Listens to all incoming resource events.
type OgReceiver struct {
	EventBus                      *eventbus.EventBus
	myNewIncomingMessageEventChan chan *transport_interface.IncomingLetter // subscribe to NewIncomingMessageEvent

	quit chan bool
}

func (o *OgReceiver) Start() {
	go o.eventLoop()
}

func (o *OgReceiver) Stop() {
	close(o.quit)
}

func (o *OgReceiver) InitDefault() {
	o.myNewIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter)
	o.quit = make(chan bool)
}

func (o *OgReceiver) Name() string {
	return "OgReceiver"
}

func (o *OgReceiver) Receive(topic int, msg interface{}) error {
	switch consts.EventType(topic) {
	case consts.NewIncomingMessageEvent:
		o.myNewIncomingMessageEventChan <- msg.(*transport_interface.IncomingLetter)

	}
}

func (o *OgReceiver) eventLoop() {
	for {
		select {
		case event := <-o.myNewIncomingMessageEventChan:
			o.handleIncomingMessage(event)
		case <-o.quit:
			return
		}
	}
}

func (o *OgReceiver) handleIncomingMessage(msg *transport_interface.IncomingLetter) {
	switch msg.Msg.MsgType {
	case ogsyncer_interface.OgAnnouncementTypeNewSequencer:
		panic("not implemented")
	case ogsyncer_interface.OgAnnouncementTypeNewTx:
		panic("not implemented")
	case ogsyncer_interface.OgAnnouncementTypeNewInt:
		ni := &ogsyncer_interface.OgAnnouncementNewInt{}
		err := ni.FromBytes(msg.Msg.ContentBytes)
		if err != nil {
			logrus.Warn("failed to deserialize OgAnnouncementTypeNewInt")
		}
		o.EventBus.Publish(int(consts.IntsReceivedEvent), &og_interface.IntsReceivedEvent{
			Ints: ni.Ints,
			From: msg.From,
		})

	}
}
