package hotstuff_event

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
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
	Send(msg *Msg, id string, pos string)
	SendToAllButMe(msg *Msg, myId string, pos string)
	SendToAllIncludingMe(msg *Msg, pos string)
	GetChannel(id string) chan *Msg
}

type LocalCommunicator struct {
	Channels map[string]chan *Msg
}

func (h *LocalCommunicator) GetChannel(id string) chan *Msg {
	return h.Channels[id]
}

func (h *LocalCommunicator) Send(msg *Msg, id string, pos string) {
	logrus.WithField("message", msg).WithField("to", id).Trace(fmt.Sprintf("[%d] sending [%s] to [%d]", msg.SenderId, pos, id))
	//defer logrus.WithField("msg", msg).WithField("to", id).Info("sent")
	//if id > 2 { // Byzantine test
	h.Channels[id] <- msg
	//}
}

func (h *LocalCommunicator) SendToAllButMe(msg *Msg, myId string, pos string) {
	for id := range h.Channels {
		if id != myId {
			h.Send(msg, id, pos)
		}
	}
}
func (h *LocalCommunicator) SendToAllIncludingMe(msg *Msg, pos string) {
	for id := range h.Channels {
		h.Send(msg, id, pos)
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
}

func (hub *LogicalCommunicator) InitDefault() {

}

// Deliver send the message to a specific peer
// Let PhysicalCommunicator decide how to reach this peer.
// Either send it directly, or let others relay the message.
func (hub *LogicalCommunicator) Deliver(msg *Msg, targetPeerId string, why string) {
	// get channel from communicatorManager
	peerId, err := peer.Decode(id)
	if err != nil {
		panic(err)
	}

	peer := hub.PhysicalCommunicator.GetNeighbour(peerId)
	if peer == nil {
		// peer not found in Neighbour Management, request connection
		hub.PhysicalCommunicator.SuggestConnection(peerId)
		// since the connection is not built yet, we return.
		return
	}
	peer.outgoingChannel <- &OutgoingRequest{
		Msg:          msg,
		SendType:     SendTypeUnicast,
		EndReceivers: []string{id},
	}
}

func (hub *LogicalCommunicator) SendToThemButMe(msg *Msg, peerIds []string, why string) {
	var newPeerIds []string
	for _, id := range peerIds {
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

func (hub *LogicalCommunicator) SendToThemIncludingMe(msg *Msg, peerIds []string, why string) {
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
