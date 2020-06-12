package transport

import (
	"context"
	"errors"
	"fmt"
	"github.com/annchain/OG/arefactor/performance"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/goffchan"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

//var BackoffConnect = time.Second * 5
var IpLayerProtocol = "tcp"

type IoEvent struct {
	Neighbour *Neighbour
	Err       error
	Reason    string
}

// PhysicalCommunicator
type PhysicalCommunicator struct {
	Port            int // listening port
	PrivateKey      core.PrivKey
	ProtocolId      string
	NetworkReporter *performance.SoccerdashReporter

	node            host.Host                                // p2p host to receive new streams
	activePeers     map[peer.ID]*Neighbour                   // active peers that will be reconnect if error
	tryingPeers     map[peer.ID]bool                         // peers that is trying to connect.
	incomingChannel chan *transport_interface.IncomingLetter // incoming message channel
	outgoingChannel chan *transport_interface.OutgoingLetter // universal outgoing channel to collect send requests
	ioEventChannel  chan *IoEvent                            // receive discard when peer disconnects

	peerConnectedSubscribers      []transport_interface.PeerConnectedEventSubscriber
	newIncomingMessageSubscribers []transport_interface.NewIncomingMessageEventSubscriber

	initWait sync.WaitGroup
	quit     chan bool
	mu       sync.RWMutex
}

func (c *PhysicalCommunicator) NewOutgoingMessageEventChannel() chan *transport_interface.OutgoingLetter {
	return c.outgoingChannel
}

func (c *PhysicalCommunicator) AddSubscriberNewIncomingMessageEvent(sub transport_interface.NewIncomingMessageEventSubscriber) {
	c.newIncomingMessageSubscribers = append(c.newIncomingMessageSubscribers, sub)
}

func (c *PhysicalCommunicator) notifyNewIncomingMessage(incomingLetter *transport_interface.IncomingLetter) {
	for _, sub := range c.newIncomingMessageSubscribers {
		//sub.NewIncomingMessageEventChannel() <- message
		<-goffchan.NewTimeoutSenderShort(sub.NewIncomingMessageEventChannel(), incomingLetter, "receive message "+sub.Name()).C
		//sub.NewIncomingMessageEventChannel() <- message
	}
}

func (c *PhysicalCommunicator) AddSubscriberPeerConnectedEvent(sub transport_interface.PeerConnectedEventSubscriber) {
	c.peerConnectedSubscribers = append(c.peerConnectedSubscribers, sub)
}

func (c *PhysicalCommunicator) notifyPeerConnected(event *transport_interface.PeerEvent) {
	for _, sub := range c.peerConnectedSubscribers {
		//sub.NewIncomingMessageEventChannel() <- message
		<-goffchan.NewTimeoutSenderShort(sub.GetPeerConnectedEventChannel(), event, "peer connected"+sub.Name()).C
		//sub.NewIncomingMessageEventChannel() <- message
	}
}

func (c *PhysicalCommunicator) Name() string {
	return "PhysicalCommunicator"
}

func (c *PhysicalCommunicator) InitDefault() {
	c.activePeers = make(map[peer.ID]*Neighbour)
	c.tryingPeers = make(map[peer.ID]bool)
	c.outgoingChannel = make(chan *transport_interface.OutgoingLetter)
	c.incomingChannel = make(chan *transport_interface.IncomingLetter)
	c.ioEventChannel = make(chan *IoEvent)
	c.initWait.Add(1)
	c.peerConnectedSubscribers = []transport_interface.PeerConnectedEventSubscriber{}
	c.newIncomingMessageSubscribers = []transport_interface.NewIncomingMessageEventSubscriber{}
	c.quit = make(chan bool)
}

func (c *PhysicalCommunicator) Start() {
	// start consuming queue
	go c.Listen()
	go c.mainLoopSend()
	go c.mainLoopReceive()
	go c.peerDiscovery()
}

func (c *PhysicalCommunicator) peerDiscovery() {
	c.initWait.Wait()
	for {
		select {
		case <-c.quit:
			return
		default:
			time.Sleep(time.Second * 1) // at least sleep one second to prevent flood
			c.pickOneAndConnect()
		}
	}
}

func (c *PhysicalCommunicator) mainLoopSend() {
	c.initWait.Wait()
	for {
		select {
		case <-c.quit:
			// TODO: close all peers
			return
		case outgoingLetter := <-c.outgoingChannel:
			logrus.WithField("outgoingLetter", outgoingLetter).Trace("physical communicator got a request from outgoing channel")
			c.handleOutgoing(outgoingLetter)
		case event := <-c.ioEventChannel:
			logrus.WithField("reason", event.Reason).WithError(event.Err).Trace("physical got neighbour down event")
			c.handlePeerError(event.Neighbour)
		}
	}
}
func (c *PhysicalCommunicator) mainLoopReceive() {
	c.initWait.Wait()
	for {
		select {
		case <-c.quit:
			// TODO: close all peers
			return
		case incomingLetter := <-c.incomingChannel:
			c.NetworkReporter.Report("receive", incomingLetter)
			c.notifyNewIncomingMessage(incomingLetter)
		}
	}
}

func (c *PhysicalCommunicator) Stop() {
	close(c.quit)
	// shut the node down
	if err := c.node.Close(); err != nil {
		panic(err)
	}
}

func (c *PhysicalCommunicator) makeHost(priv core.PrivKey) (host.Host, error) {
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/%s/%d", IpLayerProtocol, c.Port)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}
	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	return basicHost, nil
}

func (c *PhysicalCommunicator) printHostInfo(basicHost host.Host) {

	// print the node's listening addresses
	// protocol is always p2p
	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", basicHost.ID().Pretty()))
	if err != nil {
		panic(err)
	}

	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	logrus.WithField("addr", fullAddr).Info("my address")
}

func (c *PhysicalCommunicator) Listen() {
	// start a libp2p node with default settings
	host, err := c.makeHost(c.PrivateKey)
	if err != nil {
		panic(err)
	}
	c.node = host

	c.printHostInfo(c.node)

	c.node.SetStreamHandler(protocol.ID(c.ProtocolId), c.HandlePeerStream)
	c.initWait.Done()
	logrus.Info("waiting for connection...")
	select {
	case <-c.quit:
		err := c.node.Close()
		if err != nil {
			logrus.WithError(err).Warn("closing communicator")
		}
	}
}

func (c *PhysicalCommunicator) HandlePeerStream(s network.Stream) {
	// must lock to prevent double channel
	c.mu.Lock()
	defer c.mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"peerId":  s.Conn().RemotePeer().String(),
		"address": s.Conn().RemoteMultiaddr().String(),
	}).Info("Peer connection established")
	peerId := s.Conn().RemotePeer()

	// deregister in the trying list
	delete(c.tryingPeers, s.Conn().RemotePeer())

	// prevent double channel
	if _, ok := c.activePeers[peerId]; ok {
		// already established. close
		err := s.Close()
		if err != nil {
			logrus.WithError(err).Warn("closing peer")
		}
		return
	}

	neighbour := &Neighbour{
		Id:              peerId,
		PrettyId:        peerId.String(),
		Stream:          s,
		IoEventChannel:  c.ioEventChannel,
		IncomingChannel: c.incomingChannel,
	}
	neighbour.InitDefault()
	c.activePeers[peerId] = neighbour
	logrus.WithField("peerId", peerId).Debug("peer becomes active")

	c.notifyPeerConnected(&transport_interface.PeerEvent{
		PeerId: neighbour.PrettyId,
	})

	neighbour.Start()
}

func (c *PhysicalCommunicator) ClosePeer(id string) {
	idt, err := peer.Decode(id)
	if err != nil {
		logrus.WithError(err).Warn("Id format not recognized")
		return
	}
	c.closePeer(idt)
}

func (c *PhysicalCommunicator) closePeer(id peer.ID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	neighbour, ok := c.activePeers[id]
	if !ok {
		logrus.WithField("peerId", id).Trace("peer already removed from activePeers")
		return
	}

	delete(c.activePeers, neighbour.Id)
	c.tryingPeers[neighbour.Id] = true
	logrus.Trace("physical is closing neighbour stream")
	err := neighbour.Stream.Close()
	if err != nil {
		logrus.WithError(err).Warn("physical closing stream")
	}
	neighbour.CloseOutgoing()
}

func (c *PhysicalCommunicator) GetNeighbour(id string) (neighbour *Neighbour, err error) {
	idp, err := peer.Decode(id)
	if err != nil {
		return
	}
	neighbour, ok := c.activePeers[idp]
	if !ok {
		err = errors.New("peer not active")
	}
	return
}

func (c *PhysicalCommunicator) GetPeerId(address string) (peerIds string, err error) {
	fullAddr, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
		return
	}
	// p2p layer address
	p2pAddr, err := fullAddr.ValueForProtocol(multiaddr.P_P2P)

	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
		return
	}

	// recover peerId from Base58 Encoded p2pAddr
	peerId, err := peer.Decode(p2pAddr)
	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
		return
	}
	peerIds = peerId.String()
	return
}

// SuggestConnection takes a peer address and try to connect to it.
func (c *PhysicalCommunicator) SuggestConnection(address string) (peerIds string) {
	c.initWait.Wait()
	logrus.WithField("address", address).Info("registering address")
	fullAddr, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
		return
	}

	// p2p layer address
	p2pAddr, err := fullAddr.ValueForProtocol(multiaddr.P_P2P)

	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
	}

	protocolAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", p2pAddr))

	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
	}
	// keep only the connection info, wipe out the p2p layer
	connectionAddr := fullAddr.Decapsulate(protocolAddr)
	//fmt.Println("connectionAddr:" + connectionAddr.String())

	// recover peerId from Base58 Encoded p2pAddr
	peerId, err := peer.Decode(p2pAddr)
	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
	}
	peerIds = peerId.String()

	//fmt.Println("peerId:" + p2pAddr)
	// check if it is a self connection.
	if peerId == c.node.ID() {
		return
	}

	// save address and peer info
	c.node.Peerstore().AddAddr(peerId, connectionAddr, peerstore.PermanentAddrTTL)

	// reg in the trying list
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.tryingPeers[peerId]; ok {
		return
	}
	c.tryingPeers[peerId] = true
	return
}

func (c *PhysicalCommunicator) Enqueue(req *transport_interface.OutgoingLetter) {
	logrus.WithField("req", req).Info("Sending message")
	c.initWait.Wait()
	<-goffchan.NewTimeoutSenderShort(c.outgoingChannel, req, "enqueue").C
	//c.outgoingChannel <- req
}

// we use direct connection currently so let's build a connection if not exists.
func (c *PhysicalCommunicator) handleOutgoing(req *transport_interface.OutgoingLetter) {
	logrus.Trace("physical is handling send request")
	if req.SendType == transport_interface.SendTypeBroadcast {
		for _, neighbour := range c.activePeers {
			neighbour.EnqueueSend(req)
		}
		return
	}
	// find neighbour first
	for _, peerIdEncoded := range req.EndReceivers {
		peerId, err := peer.Decode(peerIdEncoded)
		if err != nil {
			logrus.WithError(err).WithField("peerIdEncoded", peerIdEncoded).Warn("decoding peer")
		}
		// get active neighbour
		neighbour, ok := c.activePeers[peerId]
		if !ok {
			// wait for node to be connected. currently node address are pre-located and connections are built ahead.
			logrus.WithField("peerId", peerId).Trace("connection not in active peers")
			continue
		}
		logrus.WithField("peerId", peerId).Trace("go send")
		c.NetworkReporter.Report("send", req)
		neighbour.EnqueueSend(req)
	}
}

func (c *PhysicalCommunicator) pickOneAndConnect() {
	c.mu.RLock()

	if len(c.tryingPeers) == 0 {
		c.mu.RUnlock()
		return
	}
	peerIds := []peer.ID{}
	for k, _ := range c.tryingPeers {
		peerIds = append(peerIds, k)
	}
	c.mu.RUnlock()

	peerId := peerIds[rand.Intn(len(peerIds))]

	// start a stream
	logrus.WithField("address", c.node.Peerstore().PeerInfo(peerId).String()).
		WithField("peerId", peerId).
		Trace("connecting peer")

	s, err := c.node.NewStream(context.Background(), peerId, protocol.ID(c.ProtocolId))
	if err != nil {
		if err != swarm.ErrDialBackoff {
			logrus.WithField("stream", s).WithError(err).Warn("error on starting stream")
		}
		return
	}
	// stream built
	// release the lock
	c.HandlePeerStream(s)

	//hub.handleStream(s)
	//// Create a buffered stream so that read and writes are non blocking.
	//rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	//
	//// Create a thread to read and write data.
	//go hub.writeData(rw)
	//go hub.readData(rw)
	//logrus.WithField("s", info).WithError(err).Warn("connection established")
}

func (c *PhysicalCommunicator) handlePeerError(neighbour *Neighbour) {
	c.closePeer(neighbour.Id)
}
