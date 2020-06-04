package og

import (
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/og/message"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/annchain/OG/ffchan"
	"github.com/sirupsen/logrus"
)

var Protocol = "og/1.0.0"

type CommunityManager interface {
}

// DefaultCommunityManager manages relationship with other peers.
// It keeps the community with a stable scale。
// It also tries to balance the whole network to prevent some node being too heavy loaded
type DefaultCommunityManager struct {
	PhysicalCommunicator  *transport.PhysicalCommunicator
	KnownPeerListFilePath string

	peers             []string
	knownPeersAddress []string

	myNewIncomingMessageEventChannel chan *transport_event.WireMessage
	newOutgoingMessageSubscribers    []transport_event.NewOutgoingMessageEventSubscriber

	quit chan bool
}

func (c *DefaultCommunityManager) InitDefault() {
	c.quit = make(chan bool)
	c.myNewIncomingMessageEventChannel = make(chan *transport_event.WireMessage)
	c.newOutgoingMessageSubscribers = []transport_event.NewOutgoingMessageEventSubscriber{}
}

func (c *DefaultCommunityManager) RegisterSubscriberNewOutgoingMessageEvent(sub transport_event.NewOutgoingMessageEventSubscriber) {
	c.newOutgoingMessageSubscribers = append(c.newOutgoingMessageSubscribers, sub)
}

func (c *DefaultCommunityManager) GetNewIncomingMessageEventChannel() chan *transport_event.WireMessage {
	return c.myNewIncomingMessageEventChannel
}

func (c *DefaultCommunityManager) Start() {
	go c.loop()
}

func (c *DefaultCommunityManager) Stop() {
	close(c.quit)
}

func (c *DefaultCommunityManager) Name() string {
	return "DefaultCommunityManager"
}

func (c *DefaultCommunityManager) loop() {
	// maintain the peer list
	// very simple implementation: just let nodes connected each other
	for _, peerAddress := range c.knownPeersAddress {
		c.PhysicalCommunicator.SuggestConnection(peerAddress)
	}
	select {
	case <-c.quit:
		return
	case msg := <-c.myNewIncomingMessageEventChannel:
		switch message.OgMessageType(msg.MsgType) {
		case message.OgMessageTypePing:
			c.handleMsgPing(msg)
		}
	}
}

func (c *DefaultCommunityManager) StaticSetup() {
	knownPeersAddress, err := transport_event.LoadKnownPeers(c.KnownPeerListFilePath)
	if err != nil {
		logrus.WithError(err).Fatal("you need provide at least one known address to connect to the address network. Place them in config/peers.lst")
	}
	c.knownPeersAddress = knownPeersAddress

	// load init peers from disk
	for _, address := range c.knownPeersAddress {
		nodeId, err := c.PhysicalCommunicator.GetPeerId(address)
		utilfuncs.PanicIfError(err, "parse node address")
		c.peers = append(c.peers, nodeId)
	}
}

func (c *DefaultCommunityManager) handleMsgPing(msg *transport_event.WireMessage) {
	m := &message.OgMessagePing{}
	_, err := m.UnmarshalMsg(msg.ContentBytes)
	if err != nil {
		logrus.WithField("type", "OgMessagePing").WithError(err).Warn("bad message")
	}
	// TODO: adapt multiple protocols
	closeFlag := false
	if m.Protocol != Protocol {
		closeFlag = true
	}

	resp := &message.OgMessagePong{
		Protocol: Protocol,
	}

	oreq := &transport_event.OutgoingRequest{
		Msg:            resp,
		SendType:       transport_event.SendTypeBroadcast,
		CloseAfterSent: closeFlag,
		EndReceivers:   nil,
	}

	for _, subscriber := range c.newOutgoingMessageSubscribers {
		ffchan.NewTimeoutSenderShort(subscriber.GetNewOutgoingMessageEventChannel(), oreq, "send pong")
	}

}
