package og

import (
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/og/message"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/goffchan"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

var Protocol = "og/1.0.0"
var PingCheckIntervalSeconds = 1

type CommunityManager interface {
}

// DefaultCommunityManager manages relationship with other peers.
// It keeps the community with a stable scale。
// It also tries to balance the whole network to prevent some node being too heavy loaded
type DefaultCommunityManager struct {
	NodeInfoProvider      og_interface.NodeInfoProvider
	PhysicalCommunicator  *transport.PhysicalCommunicator
	KnownPeerListFilePath string

	// TODO: peer management. currently all peers are inited at the beginning.
	// active peer list should be
	peers             []string
	knownPeersAddress []string

	myNewIncomingMessageEventChannel chan *transport_interface.IncomingLetter // handle incoming messages
	myPeerConnectedEventChannel      chan *transport_interface.PeerEvent
	myPeerDisconnectedEventChannel   chan *transport_interface.PeerEvent
	newOutgoingMessageSubscribers    []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent
	peerJoinedEventSubscribers       []og_interface.PeerJoinedEventSubscriber                // a peer is connected after protocol verification
	peerLeftEventSubscribers         []og_interface.PeerLeftEventSubscriber                  // a peer is left due to disconnection

	quit chan bool
}

func (d *DefaultCommunityManager) GetPeerDisconnectedEventChannel() chan *transport_interface.PeerEvent {
	return d.myPeerDisconnectedEventChannel
}

func (d *DefaultCommunityManager) GetPeerConnectedEventChannel() chan *transport_interface.PeerEvent {
	return d.myPeerConnectedEventChannel
}

func (d *DefaultCommunityManager) InitDefault() {
	d.quit = make(chan bool)
	d.myPeerConnectedEventChannel = make(chan *transport_interface.PeerEvent)
	d.myPeerDisconnectedEventChannel = make(chan *transport_interface.PeerEvent)
	d.myNewIncomingMessageEventChannel = make(chan *transport_interface.IncomingLetter)
	d.newOutgoingMessageSubscribers = []transport_interface.NewOutgoingMessageEventSubscriber{}
	d.peerJoinedEventSubscribers = []og_interface.PeerJoinedEventSubscriber{}
	d.peerLeftEventSubscribers = []og_interface.PeerLeftEventSubscriber{}
}

// event handlers begin

// my subscriptions
func (d *DefaultCommunityManager) NewIncomingMessageEventChannel() chan *transport_interface.IncomingLetter {
	return d.myNewIncomingMessageEventChannel
}

// subscribe mine
func (d *DefaultCommunityManager) AddSubscriberNewOutgoingMessageEvent(sub transport_interface.NewOutgoingMessageEventSubscriber) {
	d.newOutgoingMessageSubscribers = append(d.newOutgoingMessageSubscribers, sub)
}

func (d *DefaultCommunityManager) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range d.newOutgoingMessageSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing community manager"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (d *DefaultCommunityManager) AddSubscriberPeerJoinedEvent(sub og_interface.PeerJoinedEventSubscriber) {
	d.peerJoinedEventSubscribers = append(d.peerJoinedEventSubscribers, sub)
}

func (d *DefaultCommunityManager) notifyPeerJoined(event *og_interface.PeerJoinedEvent) {
	for _, subscriber := range d.peerJoinedEventSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.EventChannelPeerJoined(), event, "peerjoined").C
		//subscriber.EventChannelPeerJoined() <- event
	}
}

func (d *DefaultCommunityManager) AddSubscriberPeerLeftEvent(sub og_interface.PeerLeftEventSubscriber) {
	d.peerLeftEventSubscribers = append(d.peerLeftEventSubscribers, sub)
}

func (d *DefaultCommunityManager) notifyPeerLeft(event *og_interface.PeerLeftEvent) {
	for _, subscriber := range d.peerLeftEventSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.EventChannelPeerLeft(), event, "peerleft").C
		//subscriber.EventChannelPeerLeft() <- event
	}
}

// event handlers over

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
	// check random peer every 5 seconds
	ticker := time.NewTicker(time.Duration(PingCheckIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.quit:
			return
		case <-ticker.C:
			// pickup random peer and senc ping
			d.SendPing("")
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
		case event := <-d.myPeerConnectedEventChannel:
			// send ping
			d.handlePeerConnected(event)
		case event := <-d.myPeerDisconnectedEventChannel:
			d.handlePeerDisconnected(event)

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

	oreq := &transport_interface.OutgoingLetter{
		Msg:            ping,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{peer},
	}
	d.notifyNewOutgoingMessage(oreq)
}

func (d *DefaultCommunityManager) StaticSetup() {
	knownPeersAddress, err := transport_interface.LoadKnownPeers(d.KnownPeerListFilePath)
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

func (d *DefaultCommunityManager) handleMsgPing(letter *transport_interface.IncomingLetter) {
	m := &message.OgMessagePing{}
	err := m.FromBytes(letter.Msg.ContentBytes)
	if err != nil {
		logrus.WithField("type", "OgMessagePing").WithError(err).Warn("bad message")
	}

	closeFlag := !d.protocolMatch(Protocol, m.Protocol)

	resp := &message.OgMessagePong{
		Protocol: Protocol,
		Close:    closeFlag,
	}

	oreq := &transport_interface.OutgoingLetter{
		Msg:            resp,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: closeFlag,
		EndReceivers:   []string{letter.From},
	}
	d.notifyNewOutgoingMessage(oreq)
}

func (d *DefaultCommunityManager) handleMsgPong(letter *transport_interface.IncomingLetter) {
	// if we get a pong with close flag, the target peer is dropping us.
	m := &message.OgMessagePong{}
	err := m.FromBytes(letter.Msg.ContentBytes)
	if err != nil {
		logrus.WithField("type", "OgMessagePong").WithError(err).Warn("bad message")
	}
	// remove the peer if not removed
	if m.Close || !d.protocolMatch(Protocol, m.Protocol) {
		logrus.WithField("peer", letter.From).Debug("closing neighbour because the target is closing")
		d.PhysicalCommunicator.ClosePeer(letter.From)
	}
	peerJoinedEvent := &og_interface.PeerJoinedEvent{
		PeerId: letter.From,
	}
	d.notifyPeerJoined(peerJoinedEvent)
}

func (d *DefaultCommunityManager) protocolMatch(mine string, theirs string) bool {
	// TODO: adapt multiple protocols
	return mine == theirs
}

func (d *DefaultCommunityManager) handlePeerConnected(event *transport_interface.PeerEvent) {
	// send ping
	resp := &message.OgMessagePing{
		Protocol: Protocol,
	}

	oreq := &transport_interface.OutgoingLetter{
		Msg:            resp,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{event.PeerId},
	}
	d.notifyNewOutgoingMessage(oreq)
}

func (d *DefaultCommunityManager) handlePeerDisconnected(event *transport_interface.PeerEvent) {
	// TODO: connect another potential node if there is spare node
	peerLeftEvent := &og_interface.PeerLeftEvent{
		PeerId: event.PeerId,
	}

	d.notifyPeerLeft(peerLeftEvent)
}
