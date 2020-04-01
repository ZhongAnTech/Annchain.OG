package hotstuff_event

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
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
	IncomingChannel chan *Msg
	rw              *bufio.ReadWriter
	event           chan bool
	outgoingChannel chan *Msg
	quit            chan bool
}

func (c *Neighbour) InitDefault() {
	c.event = make(chan bool)
	c.quit = make(chan bool)
	c.outgoingChannel = make(chan *Msg) // messages already dispatched
	c.rw = bufio.NewReadWriter(bufio.NewReader(c.Stream), bufio.NewWriter(c.Stream))
}

func (c *Neighbour) StartRead() {
	var err error
	for {

		bytes, err := c.rw.ReadBytes('\n')
		if err != nil {
			logrus.WithError(err).Warn("read err")
			break
		}
		msg := &WireMessage{}
		msg.
			_, err = msg.UnmarshalMsg(bytes)
		if err != nil {
			logrus.WithError(err).Warn("read err")
			break
		}
		fmt.Println("Received: " + msg.Content.String())
		c.IncomingChannel <- msg
	}
	// neighbour disconnected, notify the communicator
	c.IoEventChannel <- &IoEvent{
		Neighbour: c,
		Err:       err,
	}

}

func (c *Neighbour) StartWrite() {
	var err error
loop:
	for {
		select {
		case req := <-c.outgoingChannel:

			contentBytes, err := req.Content.MarshalMsg([]byte{})
			if err != nil {
				panic(err)
			}

			wireMessage := WireMessage{
				MsgType:      int(req.Typev),
				ContentBytes: contentBytes,
				SenderId:     req.SenderId,
			}

			bytes, err := wireMessage.MarshalMsg([]byte{})
			fmt.Println("Sent: " + hex.EncodeToString(bytes))

			if err != nil {
				panic(err)
			}
			_, err = c.rw.Write(bytes)
			if err != nil {
				logrus.WithError(err).Warn("write err")
				break loop
			}
			err = c.rw.WriteByte(byte('\n'))
			if err != nil {
				logrus.WithError(err).Warn("write err")
				break loop
			}

			err = c.rw.Flush()
			if err != nil {
				logrus.WithError(err).Warn("write err")
				break loop
			}
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
	c.outgoingChannel <- req
}

// PhysicalCommunicator
type PhysicalCommunicator struct {
	node            host.Host              // p2p host to receive new streams
	Port            int                    // listening port
	ActivePeers     map[peer.ID]*Neighbour // active peers that will be reconnect if error
	outgoingChannel chan *OutgoingRequest  // universal outgoing channel to collect send requests
	initWait        sync.WaitGroup
	quit            chan bool
	ioEventChannel  chan *IoEvent // receive event when peer disconnects
	incomingChannel chan *Msg     // incoming message channel
}

func (c *PhysicalCommunicator) InitDefault() {
	c.ActivePeers = make(map[peer.ID]*Neighbour)
	c.outgoingChannel = make(chan *OutgoingRequest)
	c.incomingChannel = make(chan *Msg)
	c.ioEventChannel = make(chan *IoEvent)
	c.initWait.Add(1)
	c.quit = make(chan bool)
}

func (c *PhysicalCommunicator) Start() {
	c.initWait.Wait()
	// start consuming queue
	go c.consumeQueue()
}

func (c *PhysicalCommunicator) consumeQueue() {
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

func (c *PhysicalCommunicator) MakeHost(priv core.PrivKey) (host.Host, error) {
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
	privKey := c.LoadPrivateKey()
	host, err := c.MakeHost(privKey)
	if err != nil {
		panic(err)
	}
	c.node = host

	c.printHostInfo(c.node)

	c.node.SetStreamHandler(ProtocolId, c.HandlePeerStream)
	c.initWait.Done()
	select {}
}

func (hub *LogicalCommunicator) LoadPrivateKey() core.PrivKey {
	// read key file
	keyFile := viper.GetString("file")
	bytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		panic(err)
	}

	pi := &PrivateInfo{}
	err = json.Unmarshal(bytes, pi)
	if err != nil {
		panic(err)
	}

	privb, err := hex.DecodeString(pi.PrivateKey)
	if err != nil {
		panic(err)
	}

	priv, err := core.UnmarshalPrivateKey(privb)
	if err != nil {
		panic(err)
	}
	return priv
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
	c.ActivePeers[peerId] = neightbour

	go neightbour.StartRead()
	go neightbour.StartWrite()

}

func (c *PhysicalCommunicator) ClosePeer(id peer.ID) {

}

func (c *PhysicalCommunicator) MustGetNeighbour(id peer.ID) *Neighbour {
	return c.ActivePeers[id]
}

// SuggestConnection takes a peerId and try to connect to it.
func (c *PhysicalCommunicator) SuggestConnection(address string) {
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

	fmt.Println("p2pAddr:" + p2pAddr)
	fmt.Println("peerIdxxx:" + peerId.String())

	// save address and peer info
	c.node.Peerstore().AddAddr(peerId, connectionAddr, peerstore.PermanentAddrTTL)

	go c.keepTryingToConnect(peerId)
}

func (c *PhysicalCommunicator) Enqueue(req *OutgoingRequest) {
	c.outgoingChannel <- req
}

// we use direct connection currently so let's build a connection if not exists.
func (c *PhysicalCommunicator) handleRequest(req *OutgoingRequest) {
	// find neighbour first
	for _, peerIdEncoded := range req.EndReceivers {
		peerId, err := peer.Decode(peerIdEncoded)
		if err != nil {
			logrus.WithError(err).WithField("peerIdEncoded", peerIdEncoded).Warn("decoding peer")
		}
		// get active neighbour
		neighbour, ok := c.ActivePeers[peerId]
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
			logrus.WithField("stream", s).WithError(err).Warn("error on starting stream")
			time.Sleep(time.Second * 5)
			continue
		}
		// stream built

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
