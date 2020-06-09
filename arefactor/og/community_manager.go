package og

import (
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/og/message"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/sirupsen/logrus"
	"math/rand"
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

	myNewIncomingMessageEventChannel chan *transport_event.IncomingLetter
	newOutgoingMessageSubscribers    []transport_event.NewOutgoingMessageEventSubscriber

	quit chan bool
}

func (d *DefaultCommunityManager) InitDefault() {
	d.quit = make(chan bool)
	d.myNewIncomingMessageEventChannel = make(chan *transport_event.IncomingLetter)
	d.newOutgoingMessageSubscribers = []transport_event.NewOutgoingMessageEventSubscriber{}
}

func (d *DefaultCommunityManager) AddSubscriberNewOutgoingMessageEvent(sub transport_event.NewOutgoingMessageEventSubscriber) {
	d.newOutgoingMessageSubscribers = append(d.newOutgoingMessageSubscribers, sub)
}

func (d *DefaultCommunityManager) GetNewIncomingMessageEventChannel() chan *transport_event.IncomingLetter {
	return d.myNewIncomingMessageEventChannel
}

func (d *DefaultCommunityManager) Start() {
	go d.loop()
}

func (d *DefaultCommunityManager) Stop() {
	close(d.quit)
}

func (d *DefaultCommunityManager) Name() string {
	return "DefaultCommunityManager"
}

func (d *DefaultCommunityManager) loop() {
	// maintain the peer list
	// very simple implementation: just let nodes connected each other
	for _, peerAddress := range d.knownPeersAddress {
		d.PhysicalCommunicator.SuggestConnection(peerAddress)
	}
	for {
		select {
		case <-d.quit:
			return
		case incomingLetter := <-d.myNewIncomingMessageEventChannel:
			logrus.WithField("c", "DefaultCommunityManager").
				WithField("from", incomingLetter.From).
				WithField("type", message.OgMessageType(incomingLetter.Msg.MsgType)).
				Trace("received message")
			switch message.OgMessageType(incomingLetter.Msg.MsgType) {
			case message.OgMessageTypePing:
				d.handleMsgPing(incomingLetter)
			case message.OgMessageTypePong:
				d.handleMsgPong(incomingLetter)
			}
		}
	}
}

func (d *DefaultCommunityManager) SendPing(peer string) {
	ping := &message.OgMessagePing{
		Protocol: Protocol,
	}

	if peer == "" {
		peer = d.peers[rand.Intn(len(d.peers))]
	}

	oreq := &transport_event.OutgoingLetter{
		Msg:            ping,
		SendType:       transport_event.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{peer},
	}
	d.notifyNewOutgoingMessageSubscribers(oreq)
}

func (d *DefaultCommunityManager) StaticSetup() {
	knownPeersAddress, err := transport_event.LoadKnownPeers(d.KnownPeerListFilePath)
	if err != nil {
		logrus.WithError(err).Fatal("you need provide at least one known address to connect to the address network. Place them in config/peers.lst")
	}
	d.knownPeersAddress = knownPeersAddress

	// load init peers from disk
	for _, address := range d.knownPeersAddress {
		nodeId, err := d.PhysicalCommunicator.GetPeerId(address)
		utilfuncs.PanicIfError(err, "parse node address")
		d.peers = append(d.peers, nodeId)
	}
}

func (d *DefaultCommunityManager) notifyNewOutgoingMessageSubscribers(req *transport_event.OutgoingLetter) {
	for _, subscriber := range d.newOutgoingMessageSubscribers {
		//goffchan.NewTimeoutSenderShort(subscriber.GetNewOutgoingMessageEventChannel(), req, "outgoing")
		subscriber.GetNewOutgoingMessageEventChannel() <- req
	}
}

func (d *DefaultCommunityManager) handleMsgPing(letter *transport_event.IncomingLetter) {
	m := &message.OgMessagePing{}
	_, err := m.UnmarshalMsg(letter.Msg.ContentBytes)
	if err != nil {
		logrus.WithField("type", "OgMessagePing").WithError(err).Warn("bad message")
	}

	closeFlag := !d.protocolMatch(Protocol, m.Protocol)

	resp := &message.OgMessagePong{
		Protocol: Protocol,
		Close:    closeFlag,
	}

	oreq := &transport_event.OutgoingLetter{
		Msg:            resp,
		SendType:       transport_event.SendTypeUnicast,
		CloseAfterSent: closeFlag,
		EndReceivers:   []string{letter.From},
	}
	d.notifyNewOutgoingMessageSubscribers(oreq)
}

func (d *DefaultCommunityManager) handleMsgPong(letter *transport_event.IncomingLetter) {
	// if we get a pong with close flag, the target peer is dropping us.
	m := &message.OgMessagePong{}
	_, err := m.UnmarshalMsg(letter.Msg.ContentBytes)
	if err != nil {
		logrus.WithField("type", "OgMessagePong").WithError(err).Warn("bad message")
	}
	// remove the peer if not removed
	if m.Close || !d.protocolMatch(Protocol, m.Protocol) {
		logrus.WithField("peer", letter.From).Debug("closing neighbour because the target is closing")
		d.PhysicalCommunicator.ClosePeer(letter.From)
	}
}

func (d *DefaultCommunityManager) protocolMatch(mine string, theirs string) bool {
	// TODO: adapt multiple protocols
	return mine == theirs
}
