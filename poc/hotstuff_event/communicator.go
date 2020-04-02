package hotstuff_event

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/sirupsen/logrus"
)

var ProtocolId protocol.ID = "/og/1.0.0"

type SendType int

const (
	SendTypeUnicast   SendType = iota // send to only one
	SendTypeMulticast                 // send to multiple receivers
	SendTypeBroadcast                 // send to all peers in the network
)

type Hub interface {
	Deliver(msg *Msg, id string, why string)
	DeliverToThemButMe(msg *Msg, targetPeerIds []string, why string)
	DeliverToThemIncludingMe(sg *Msg, peerIds []string, why string)
	Broadcast(msg *Msg, why string)
	GetChannel(myId string) (c chan *Msg, err error)
}

type LocalCommunicator struct {
	Channels map[string]chan *Msg
	myId     string
}

func (h *LocalCommunicator) Broadcast(msg *Msg, why string) {
	panic("implement me")
}

func (h *LocalCommunicator) GetChannel(id string) (c chan *Msg, err error) {
	return h.Channels[id], nil
}

func (h *LocalCommunicator) Deliver(msg *Msg, id string, pos string) {
	logrus.WithField("message", msg).WithField("to", id).Trace(fmt.Sprintf("[%d] sending [%s] to [%d]", msg.SenderId, pos, id))
	//defer logrus.WithField("msg", msg).WithField("to", id).Info("sent")
	//if id > 2 { // Byzantine test
	h.Channels[id] <- msg
	//}
}

func (h *LocalCommunicator) DeliverToThemButMe(msg *Msg, targetPeerIds []string, why string) {
	for _, id := range targetPeerIds {
		if id != h.myId {
			if _, ok := h.Channels[id]; !ok {
				panic("id not in channel list: " + id)
			}
			h.Deliver(msg, id, why)
		}
	}
}

func (h *LocalCommunicator) DeliverToThemIncludingMe(msg *Msg, targetPeerIds []string, why string) {
	for _, id := range targetPeerIds {
		if _, ok := h.Channels[id]; !ok {
			panic("id not in channel list: " + id)
		}
		h.Deliver(msg, id, why)
	}
}

type OutgoingRequest struct {
	Msg          *Msg
	SendType     SendType
	EndReceivers []string
}

// LogicalCommunicator is for logical send. LogicalCommunicator only specify receiver peerId and message.
// LogicalCommunicator does not known how to deliver the message. It only specify who to receive.
// Use some physical sending such as PhysicalCommunicator to either directly send or relay messages.
type LogicalCommunicator struct {
	PhysicalCommunicator *PhysicalCommunicator // do the real communication. connection management
	MyId                 string                // my peerId in the p2p network
	quit                 chan bool
	msgChan              chan *Msg
}

func (hub *LogicalCommunicator) GetChannel(myId string) (c chan *Msg, err error) {
	return hub.msgChan, nil
}

func (hub *LogicalCommunicator) InitDefault() {
	hub.quit = make(chan bool)
	hub.msgChan = make(chan *Msg)
}

func (hub *LogicalCommunicator) Start() {
	// consume incoming channel of pyhsicalCommunicator and relay it to upper
	go hub.pump()
}

func (hub *LogicalCommunicator) pump() {
	for {
		select {
		case <-hub.quit:
			return
		case wmsg := <-hub.PhysicalCommunicator.GetIncomingChannel():
			// decode msg
			switch MsgType(wmsg.MsgType) {
			case String:
				msg := &ContentString{}
				_, err := msg.UnmarshalMsg(wmsg.ContentBytes)
				if err != nil {
					logrus.WithError(err).Warn("unmarshal")
				}
				hub.msgChan <- &Msg{
					Typev:    String,
					Sig:      Signature{},
					SenderId: wmsg.SenderId,
					Content:  msg,
				}
			default:
				panic("unsupported type")
			}
		}
	}
}

// Deliver send the message to a specific peer
// Let PhysicalCommunicator decide how to reach this peer.
// Either send it directly, or let others relay the message.
func (hub *LogicalCommunicator) Deliver(msg *Msg, targetPeerId string, why string) {
	// get channel from communicatorManager
	hub.PhysicalCommunicator.Enqueue(&OutgoingRequest{
		Msg:          msg,
		SendType:     SendTypeUnicast,
		EndReceivers: []string{targetPeerId},
	})
}

func (hub *LogicalCommunicator) DeliverToThemButMe(msg *Msg, targetPeerIds []string, why string) {
	var newPeerIds []string
	for _, id := range targetPeerIds {
		if id != hub.MyId {
			newPeerIds = append(newPeerIds, id)
		}
	}
	if len(newPeerIds) == 0 {
		return
	}
	hub.PhysicalCommunicator.Enqueue(&OutgoingRequest{
		Msg:          msg,
		SendType:     SendTypeMulticast,
		EndReceivers: newPeerIds,
	})
}

func (hub *LogicalCommunicator) DeliverToThemIncludingMe(msg *Msg, peerIds []string, why string) {
	if len(peerIds) == 0 {
		return
	}
	hub.PhysicalCommunicator.Enqueue(&OutgoingRequest{
		Msg:          msg,
		SendType:     SendTypeMulticast,
		EndReceivers: peerIds,
	})
}

func (hub *LogicalCommunicator) Broadcast(msg *Msg, why string) {
	hub.PhysicalCommunicator.Enqueue(&OutgoingRequest{
		Msg:      msg,
		SendType: SendTypeBroadcast,
	})
}
