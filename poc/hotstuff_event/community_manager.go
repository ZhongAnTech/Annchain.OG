package hotstuff_event

import (
	"context"
	"errors"
	"fmt"
	"github.com/annchain/OG/ffchan"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"github.com/tinylib/msgp/msgp"
	"log"
	"sync"
	"time"
)

var BackoffConnect = time.Second * 5

type IoEvent struct {
	Neighbour *Neighbour
	Err       error
}

type Neighbour struct {
	Id              peer.ID
	Stream          network.Stream
	IoEventChannel  chan *IoEvent
	IncomingChannel chan *WireMessage
	msgpReader      *msgp.Reader
	msgpWriter      *msgp.Writer
	event           chan bool
	outgoingChannel chan *Msg
	quit            chan bool
}

func (c *Neighbour) InitDefault() {
	c.event = make(chan bool)
	c.quit = make(chan bool)
	c.outgoingChannel = make(chan *Msg) // messages already dispatched
}

func (c *Neighbour) StartRead() {
	var err error
	c.msgpReader = msgp.NewReader(c.Stream)
	for {
		msg := &WireMessage{}
		err = msg.DecodeMsg(c.msgpReader)
		if err != nil {
			// bad message, drop
			logrus.WithError(err).Warn("bad message")
			break
		}

		fmt.Println("WiredReceived: " + msg.String())
		ffchan.NewTimeoutSenderShort(c.IncomingChannel, msg, "read")
		//c.IncomingChannel <- msg
	}
	// neighbour disconnected, notify the communicator
	c.IoEventChannel <- &IoEvent{
		Neighbour: c,
		Err:       err,
	}

}

func (c *Neighbour) StartWrite() {
	var err error
	c.msgpWriter = msgp.NewWriter(c.Stream)
loop:
	for {
		select {
		case req := <-c.outgoingChannel:
			logrus.Info("neighbour got send request")
			contentBytes, err := req.Content.MarshalMsg([]byte{})
			if err != nil {
				panic(err)
			}

			wireMessage := WireMessage{
				MsgType:      int(req.Typev),
				ContentBytes: contentBytes,
				SenderId:     req.SenderId,
			}

			err = wireMessage.EncodeMsg(c.msgpWriter)
			if err != nil {
				break
			}
			err = c.msgpWriter.Flush()
			if err != nil {
				break
			}
			logrus.Info("neighbour sent")

		case <-c.quit:
			break loop
		}
	}
	// neighbour disconnected, notify the communicator
	c.IoEventChannel <- &IoEvent{
		Neighbour: c,
		Err:       err,
	}
}

func (c *Neighbour) Send(req *Msg) {
	<-ffchan.NewTimeoutSenderShort(c.outgoingChannel, req, "send").C
	//c.outgoingChannel <- req
}

// PhysicalCommunicator
type PhysicalCommunicator struct {
	Port       int // listening port
	PrivateKey core.PrivKey

	node            host.Host              // p2p host to receive new streams
	activePeers     map[peer.ID]*Neighbour // active peers that will be reconnect if error
	outgoingChannel chan *OutgoingRequest  // universal outgoing channel to collect send requests
	initWait        sync.WaitGroup
	quit            chan bool
	ioEventChannel  chan *IoEvent     // receive event when peer disconnects
	incomingChannel chan *WireMessage // incoming message channel
}

func (c *PhysicalCommunicator) InitDefault() {
	c.activePeers = make(map[peer.ID]*Neighbour)
	c.outgoingChannel = make(chan *OutgoingRequest)
	c.incomingChannel = make(chan *WireMessage)
	c.ioEventChannel = make(chan *IoEvent)
	c.initWait.Add(1)
	c.quit = make(chan bool)
}

func (c *PhysicalCommunicator) Start() {
	// start consuming queue
	go c.Listen()
	go c.consumeQueue()
}

func (c *PhysicalCommunicator) consumeQueue() {
	c.initWait.Wait()
	for {
		select {
		case req := <-c.outgoingChannel:
			go c.handleRequest(req)
		case <-c.quit:
			return
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

func (c *PhysicalCommunicator) GetIncomingChannel() chan *WireMessage {
	return c.incomingChannel
}

func (c *PhysicalCommunicator) makeHost(priv core.PrivKey) (host.Host, error) {
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", c.Port)),
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
	log.Printf("I am %s\n", fullAddr)
}

func (c *PhysicalCommunicator) Listen() {
	// start a libp2p node with default settings
	host, err := c.makeHost(c.PrivateKey)
	if err != nil {
		panic(err)
	}
	c.node = host

	c.printHostInfo(c.node)

	c.node.SetStreamHandler(ProtocolId, c.HandlePeerStream)
	c.initWait.Done()
	logrus.Info("waiting for connection...")
	select {}
}

func (c *PhysicalCommunicator) HandlePeerStream(s network.Stream) {
	logrus.Info("Got a new stream!")
	peerId := s.Conn().RemotePeer()
	neightbour := &Neighbour{
		Id:              peerId,
		Stream:          s,
		IoEventChannel:  c.ioEventChannel,
		IncomingChannel: c.incomingChannel,
	}
	neightbour.InitDefault()
	c.activePeers[peerId] = neightbour

	go neightbour.StartRead()
	go neightbour.StartWrite()

}

func (c *PhysicalCommunicator) ClosePeer(id string) {

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

// SuggestConnection takes a peerId and try to connect to it.
func (c *PhysicalCommunicator) SuggestConnection(address string) {
	c.initWait.Wait()
	logrus.WithField("address", address).Info("processing")
	fullAddr, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
		return
	}
	logrus.WithField("fullAddr", fullAddr).Info("processing address")

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
	fmt.Println("connectionAddr:" + connectionAddr.String())

	// recover peerId from Base58 Encoded p2pAddr
	peerId, err := peer.Decode(p2pAddr)
	if err != nil {
		logrus.WithField("address", address).WithError(err).Warn("bad address")
	}

	fmt.Println("peerId:" + p2pAddr)
	// check if it is a self connection.
	if peerId == c.node.ID() {
		return
	}

	// save address and peer info
	c.node.Peerstore().AddAddr(peerId, connectionAddr, peerstore.PermanentAddrTTL)

	go c.keepTryingToConnect(peerId)
}

func (c *PhysicalCommunicator) Enqueue(req *OutgoingRequest) {
	c.initWait.Wait()
	<-ffchan.NewTimeoutSenderShort(c.outgoingChannel, req, "enqueue").C
	//c.outgoingChannel <- req
}

// we use direct connection currently so let's build a connection if not exists.
func (c *PhysicalCommunicator) handleRequest(req *OutgoingRequest) {
	logrus.Info("handling send request")
	if req.SendType == SendTypeBroadcast {
		for _, neighbour := range c.activePeers {
			go neighbour.Send(req.Msg)
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
			return
		}
		go neighbour.Send(req.Msg)
	}
}

func (c *PhysicalCommunicator) keepTryingToConnect(peerId peer.ID) {
	for {
		// start a stream
		s, err := c.node.NewStream(context.Background(), peerId, ProtocolId)
		if err != nil {
			if err != swarm.ErrDialBackoff {
				logrus.WithField("stream", s).WithError(err).Warn("error on starting stream")
			}
			time.Sleep(time.Second * 5)
			continue
		}
		// stream built
		c.HandlePeerStream(s)

		//hub.handleStream(s)
		//// Create a buffered stream so that read and writes are non blocking.
		//rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
		//
		//// Create a thread to read and write data.
		//go hub.writeData(rw)
		//go hub.readData(rw)
		//logrus.WithField("s", info).WithError(err).Warn("connection established")

		break
	}
}
