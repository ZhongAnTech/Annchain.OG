package transport

import (
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/latifrons/goffchan"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/sirupsen/logrus"
	"github.com/tinylib/msgp/msgp"
	"sync/atomic"
)

const (
	READ     = 0 // close by read error
	WRITE    = 1 // close by write error
	ACTIVELY = 2 // actively closed on demand

)

type Neighbour struct {
	Id              peer.ID
	PrettyId        string
	Stream          network.Stream
	IoEventChannel  chan *IoEvent
	IncomingChannel chan *transport_event.IncomingLetter
	msgpReader      *msgp.Reader
	msgpWriter      *msgp.Writer

	outgoingChannel chan *transport_event.OutgoingLetter

	closing int32
	quit    chan bool
}

func (c *Neighbour) InitDefault() {
	c.quit = make(chan bool)
	c.outgoingChannel = make(chan *transport_event.OutgoingLetter) // messages already dispatched
}

func (c *Neighbour) Start() {
	go c.loopRead()
	go c.loopWrite()
}

func (c *Neighbour) loopRead() {
	var err error
	c.msgpReader = msgp.NewReader(c.Stream)
	for {
		msg := &transport_event.WireMessage{}
		err = msg.DecodeMsg(c.msgpReader)
		if err != nil {
			// bad message, drop
			logrus.WithError(err).Warn("read error")
			c.peerError(err, "read")
			break
		}

		incoming := &transport_event.IncomingLetter{
			Msg:  msg,
			From: c.PrettyId,
		}

		<-goffchan.NewTimeoutSenderShort(c.IncomingChannel, incoming, "read").C
		//c.IncomingChannel <- message
	}
	logrus.Trace("peer read end")
}

func (c *Neighbour) write(wireMessage *transport_event.WireMessage) (err error) {
	err = wireMessage.EncodeMsg(c.msgpWriter)
	if err != nil {
		return
	}
	err = c.msgpWriter.Flush()
	if err != nil {
		return
	}
	return
}

func (c *Neighbour) loopWrite() {
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

			wireMessage := &transport_event.WireMessage{
				MsgType:      req.Msg.GetType(),
				ContentBytes: req.Msg.ToBytes(),
			}
			err = c.write(wireMessage)
			if err != nil {
				c.peerError(err, "write")
			}

			logrus.Trace("neighbour sent")

			if req.CloseAfterSent {
				logrus.Trace("close actively on demand")
				c.peerError(nil, "active")
			}
		}
	}
	logrus.Trace("peer write end")
}

func (c *Neighbour) EnqueueSend(req *transport_event.OutgoingLetter) {
	select {
	case c.outgoingChannel <- req:
	default:
		logrus.Trace("enqueue failed")
	}
}

func (c *Neighbour) Close() {
	if atomic.AddInt32(&c.closing, 1) != 1 {
		// cannot close twice
		logrus.Trace("forbid close twice")
		return
	}
	logrus.Trace("neighbour closing")
	close(c.outgoingChannel)
	logrus.Trace("neighbour closed")
}

func (c *Neighbour) peerError(err error, reason string) {
	// neighbour disconnected, notify the communicator
	logrus.Trace("notifying neighbour closed event")
	c.IoEventChannel <- &IoEvent{
		Neighbour: c,
		Err:       err,
		Reason:    reason,
	}
}
