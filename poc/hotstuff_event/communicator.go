package hotstuff_event

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"strconv"
)

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
}

func (hub *RemoteHub) InitDefault() {
	hub.ipAddresses = make(map[int]peer.ID)
	hub.incomingChannel = make(chan *Msg)
	hub.outgoingChannel = make(chan *OutgoingRequest)
	hub.quit = make(chan bool)
}

func (hub *RemoteHub) Start() {
	ctx := context.Background()
	// start a libp2p node with default settings
	node, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/"+strconv.Itoa(hub.Port)))
	//node, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		panic(err)
	}
	hub.node = node
	// print the node's listening addresses
	fmt.Println("Listen addresses:", node.Addrs())
}

func (hub *RemoteHub) Stop() {
	close(hub.quit)
	// shut the node down
	if err := hub.node.Close(); err != nil {
		panic(err)
	}
}

func (hub *RemoteHub) InitPeers(ips []string) {
	for i, v := range ips {
		maddr, err := multiaddr.NewMultiaddr(v)
		if err != nil {
			logrus.WithField("v", v).WithError(err).Warn("bad address")
			continue
		}
		logrus.WithField("maddr", maddr).WithError(err).Info("process address")
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			logrus.WithField("info", info).WithError(err).Warn("process address")
			continue
		}
		hub.node.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		// start a stream
		s, err := hub.node.NewStream(context.Background(), info.ID, "/libra/1.0")
		if err != nil {
			logrus.WithField("s", s).WithError(err).Warn("start stream")
			continue
		}
		hub.ipAddresses[i] = info.ID

		// Create a buffered stream so that read and writes are non blocking.
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		// Create a thread to read and write data.
		go hub.writeData(rw, i)
		go hub.readData(rw, i)
	}
}

func (hub *RemoteHub) readData(rw *bufio.ReadWriter, fromId int) {
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

func (hub *RemoteHub) writeData(rw *bufio.ReadWriter, fromId int) {
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
