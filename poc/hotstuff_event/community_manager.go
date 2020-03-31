package hotstuff_event

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/sirupsen/logrus"
)

type Neighbour struct {
	Id              peer.ID
	Stream          network.Stream
	rw              *bufio.ReadWriter
	event           chan bool
	outgoingChannel chan *OutgoingRequest
	quit            chan bool
}

func (c *Neighbour) InitDefault() {
	c.event = make(chan bool)
	c.quit = make(chan bool)
	c.outgoingChannel = make(chan *OutgoingRequest) // messages already dispatched
}

func (c *Neighbour) StartRead() {
	for {
		bytes, err := c.rw.ReadBytes('\n')
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
		fmt.Println("Received: " + msg.Content.String())
		//hub.incomingChannel <- msg
	}
}

func (c *Neighbour) StartWrite() {
	for {
		select {
		case req := <-c.outgoingChannel:
			bytes, err := req.Msg.MarshalMsg([]byte{})
			fmt.Println("Sent: " + hex.EncodeToString(bytes))

			if err != nil {
				panic(err)
			}
			_, err = c.rw.Write(bytes)
			if err != nil {
				logrus.WithError(err).Warn("write err")
				return
			}
			err = c.rw.WriteByte(byte('\n'))
			if err != nil {
				logrus.WithError(err).Warn("write err")
				return
			}

			err = c.rw.Flush()
			if err != nil {
				logrus.WithError(err).Warn("write err")
				return
			}
		case <-c.quit:
			return

		}
	}
}

func (c *Neighbour) Send(req *OutgoingRequest) {
	c.outgoingChannel <- req
}

// CommunityManager
type CommunityManager struct {
	ActivePeers map[peer.ID]*Neighbour // active peers that will be reconnect if error
}

func (c *CommunityManager) InitDefault() {
	c.ActivePeers = make(map[peer.ID]*Neighbour)
}

func (c *CommunityManager) MaintainPeer(s network.Stream) {
	logrus.Info("Got a new stream!")
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go hub.writeData(rw)
	go hub.readData(rw)
}

func (c *CommunityManager) ClosePeer(id peer.ID) {

}
