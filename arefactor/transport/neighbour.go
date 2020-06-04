package transport

import (
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/annchain/OG/ffchan"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/sirupsen/logrus"
	"github.com/tinylib/msgp/msgp"
)

type Neighbour struct {
	Id              peer.ID
	Stream          network.Stream
	IoEventChannel  chan *IoEvent
	IncomingChannel chan *transport_event.WireMessage
	msgpReader      *msgp.Reader
	msgpWriter      *msgp.Writer
	event           chan bool
	outgoingChannel chan transport_event.OutgoingMsg
	quit            chan bool
}

func (c *Neighbour) InitDefault() {
	c.event = make(chan bool)
	c.quit = make(chan bool)
	c.outgoingChannel = make(chan transport_event.OutgoingMsg) // messages already dispatched
}

func (c *Neighbour) StartRead() {
	var err error
	c.msgpReader = msgp.NewReader(c.Stream)
	for {
		msg := &transport_event.WireMessage{}
		err = msg.DecodeMsg(c.msgpReader)
		if err != nil {
			// bad message, drop
			logrus.WithError(err).Warn("read error")
			break
		}

		<-ffchan.NewTimeoutSenderShort(c.IncomingChannel, msg, "read").C
		//c.IncomingChannel <- message
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
			logrus.Trace("neighbour got send request")

			wireMessage := transport_event.WireMessage{
				MsgType:      req.GetType(),
				ContentBytes: req.ToBytes(),
			}

			err = wireMessage.EncodeMsg(c.msgpWriter)
			if err != nil {
				break
			}
			err = c.msgpWriter.Flush()
			if err != nil {
				break
			}
			logrus.Trace("neighbour sent")

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

func (c *Neighbour) Send(req transport_event.OutgoingMsg) {
	<-ffchan.NewTimeoutSenderShort(c.outgoingChannel, req, "send").C
	//c.outgoingChannel <- req
}
