package hotstuff_event

import (
	"bufio"
	"bytes"
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
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

var ProtocolId protocol.ID = "/og/1.0.0"

type Hub interface {
	Send(msg *Msg, id int, pos string)
	SendToAllButMe(msg *Msg, myId int, pos string)
	SendToAllIncludingMe(msg *Msg, pos string)
	GetChannel(id int) chan *Msg
}

type LocalHub struct {
	Channels map[int]chan *Msg
}

func (h *LocalHub) GetChannel(id int) chan *Msg {
	return h.Channels[id]
}

func (h *LocalHub) Send(msg *Msg, id int, pos string) {
	logrus.WithField("message", msg).WithField("to", id).Trace(fmt.Sprintf("[%d] sending [%s] to [%d]", msg.SenderId, pos, id))
	//defer logrus.WithField("msg", msg).WithField("to", id).Info("sent")
	//if id > 2 { // Byzantine test
	h.Channels[id] <- msg
	//}
}

func (h *LocalHub) SendToAllButMe(msg *Msg, myId int, pos string) {
	for id := range h.Channels {
		if id != myId {
			h.Send(msg, id, pos)
		}
	}
}
func (h *LocalHub) SendToAllIncludingMe(msg *Msg, pos string) {
	for id := range h.Channels {
		h.Send(msg, id, pos)
	}
}

type OutgoingRequest struct {
	Msg      *Msg
	Receiver int
}

type RemoteHub struct {
	ipAddresses     map[int]peer.ID
	Port            int
	incomingChannel chan *Msg
	outgoingChannel chan *OutgoingRequest
	node            host.Host
	quit            chan bool
	initWait        sync.WaitGroup
}

func (hub *RemoteHub) InitDefault() {
	hub.ipAddresses = make(map[int]peer.ID)
	hub.incomingChannel = make(chan *Msg)
	hub.outgoingChannel = make(chan *OutgoingRequest)
	hub.quit = make(chan bool)
	hub.initWait.Add(1)
}

func (hub *RemoteHub) Start() {
}

func (hub *RemoteHub) Stop() {
	close(hub.quit)
	// shut the node down
	if err := hub.node.Close(); err != nil {
		panic(err)
	}
}

func (hub *RemoteHub) LoadPrivateKey() core.PrivKey {
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

func (hub *RemoteHub) MakeHost(priv core.PrivKey) (host.Host, error) {
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", hub.Port)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}
	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	return basicHost, nil
}

func (hub *RemoteHub) printHostInfo(basicHost host.Host) {

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

func (hub *RemoteHub) Listen() {
	// start a libp2p node with default settings
	privKey := hub.LoadPrivateKey()
	host, err := hub.MakeHost(privKey)
	if err != nil {
		panic(err)
	}
	hub.node = host

	hub.printHostInfo(hub.node)

	hub.node.SetStreamHandler(ProtocolId, hub.handleStream)
	hub.initWait.Done()
	select {}
}

func (hub *RemoteHub) handleStream(s network.Stream) {
	logrus.Info("Got a new stream!")

	// Create a buffered stream so that read and writes are non blocking.
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	// Create a thread to read and write data.
	go hub.writeData(rw)
	go hub.readData(rw)
}
func (hub *RemoteHub) InitPeers(ips []string) {
	hub.initWait.Wait()
	if hub.Port == 3300 {
		return
	}

	for _, v := range ips {
		logrus.WithField("ip", v).Info("processing")
		fullAddr, err := multiaddr.NewMultiaddr(v)
		if err != nil {
			logrus.WithField("v", v).WithError(err).Warn("bad address")
			continue
		}
		logrus.WithField("fullAddr", fullAddr).Info("processing address")

		// p2p layer address
		p2pAddr, err := fullAddr.ValueForProtocol(multiaddr.P_P2P)
		protocolAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", p2pAddr))

		// self checking
		if hub.node.ID().String() == p2pAddr {
			// myself, skip
			continue
		}

		if err != nil {
			log.Fatalln(err)
		}
		// keep only the connection info, wipe out the p2p layer
		connectionAddr := fullAddr.Decapsulate(protocolAddr)
		fmt.Println("connectionAddr:" + connectionAddr.String())

		// recover peerId from Base58 Encoded p2pAddr
		peerId, err := peer.Decode(p2pAddr)
		if err != nil {
			panic(err)
		}

		fmt.Println("p2pAddr:" + p2pAddr)
		fmt.Println("peerIdxxx:" + peerId.String())

		hub.node.Peerstore().AddAddr(peerId, connectionAddr, peerstore.PermanentAddrTTL)

		for {
			// start a stream
			s, err := hub.node.NewStream(context.Background(), peerId, ProtocolId)
			if err != nil {
				logrus.WithField("s", s).WithError(err).Warn("error on starting stream")
				time.Sleep(time.Second * 5)
				continue
			}
			hub.handleStream(s)
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
}

func (hub *RemoteHub) readData(rw *bufio.ReadWriter) {
	for {
		bytes, err := rw.ReadBytes('\n')
		if err != nil {
			logrus.WithError(err).Warn("read err")
			return
		}
		msg := &Msg{}
		_, err = msg.UnmarshalMsg(bytes)
		if err != nil {
			logrus.WithError(err).Warn("read err")
			return
		}
		hub.incomingChannel <- msg
	}
}

func (hub *RemoteHub) writeData(rw *bufio.ReadWriter) {
	for {
		select {
		case req := <-hub.outgoingChannel:
			bytes, err := req.Msg.MarshalMsg([]byte{})
			if err != nil {
				panic(err)
			}
			rw.Write(bytes)
			rw.WriteByte(byte('\n'))
			rw.Flush()
		case <-hub.quit:
			return

		}
	}
}

func (hub *RemoteHub) Send(msg *Msg, id int, pos string) {
	hub.outgoingChannel <- &OutgoingRequest{
		Msg:      msg,
		Receiver: id,
	}
}

func (hub *RemoteHub) SendToAllButMe(msg *Msg, myId int, pos string) {
	for id := range hub.ipAddresses {
		if id != myId {
			hub.Send(msg, id, pos)
		}
	}
}

func (hub *RemoteHub) SendToAllIncludingMe(msg *Msg, pos string) {
	for id := range hub.ipAddresses {
		hub.Send(msg, id, pos)
	}
}

func (hub *RemoteHub) GetChannel(id int) chan *Msg {
	return hub.incomingChannel
}

func isTransportOver(data []byte) (over bool) {
	over = bytes.HasSuffix(data, []byte{0})
	return
}
