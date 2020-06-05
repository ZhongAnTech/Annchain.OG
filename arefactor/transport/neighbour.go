package transport

import (
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/latifrons/goffchan"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/sirupsen/logrus"
	"github.com/tinylib/msgp/msgp"
)

type Neighbour struct {
	Id              peer.ID
	PrettyId        string
	Stream          network.Stream
	IoEventChannel  chan *IoEvent
	IncomingChannel chan *transport_event.IncomingLetter
	msgpReader      *msgp.Reader
	msgpWriter      *msgp.Writer
	event           chan bool
	outgoingChannel chan *transport_event.OutgoingLetter
	quit            chan bool
}

func (c *Neighbour) InitDefault() {
	c.event = make(chan bool)
	c.quit = make(chan bool)
	c.outgoingChannel = make(chan *transport_event.OutgoingLetter) // messages already dispatched
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

		incoming := &transport_event.IncomingLetter{
			Msg:  msg,
			From: c.PrettyId,
		}

		<-goffchan.NewTimeoutSenderShort(c.IncomingChannel, incoming, "read").C
		//c.IncomingChannel <- message
	}
	logrus.Trace("closing peer in read break")
	err = c.Disconnect()
	if err != nil {
		logrus.WithError(err).Warn("failed to close from read")
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
		case req, ok := <-c.outgoingChannel:
			if !ok {
				break loop
			}
			logrus.Trace("neighbour got send request")

			wireMessage := transport_event.WireMessage{
				MsgType:      req.Msg.GetType(),
				ContentBytes: req.Msg.ToBytes(),
			}

			err = wireMessage.EncodeMsg(c.msgpWriter)
			if err != nil {
				break loop
			}
			err = c.msgpWriter.Flush()
			if err != nil {
				break loop
			}
			logrus.Trace("neighbour sent")

			if req.CloseAfterSent {
				logrus.Trace("closing peer in active break")
				err := c.Stream.Close()
				if err != nil {
					logrus.WithError(err).Debug("error on closing peer")
				} else {
					logrus.Debug("peer closed actively")
				}
			}

		case <-c.quit:
			break loop
		}
	}
	logrus.Trace("closing peer in write break")
	err = c.Disconnect()
	if err != nil {
		logrus.WithError(err).Warn("failed to close from write")
	}
	// neighbour disconnected, notify the communicator
	c.IoEventChannel <- &IoEvent{
		Neighbour: c,
		Err:       err,
	}
}

func (c *Neighbour) Send(req *transport_event.OutgoingLetter) {
	<-goffchan.NewTimeoutSenderShort(c.outgoingChannel, req, "send").C
	//c.outgoingChannel <- req
}

func (c *Neighbour) Disconnect() error {
	close(c.outgoingChannel)
	return c.Stream.Close()
}