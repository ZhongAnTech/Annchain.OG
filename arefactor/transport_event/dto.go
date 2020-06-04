package transport_event

import (
	"fmt"
)

type SendType int

func (s SendType) String() string {
	switch s {
	case SendTypeUnicast:
		return "Unicast"
	case SendTypeMulticast:
		return "Multicast"
	case SendTypeBroadcast:
		return "Broadcast"
	default:
		panic("unknown send type")
	}
}

const (
	SendTypeUnicast   SendType = iota // send to only one
	SendTypeMulticast                 // send to multiple receivers
	SendTypeBroadcast                 // send to all peers in the network
)

type OutgoingRequest struct {
	Msg            OutgoingMsg
	SendType       SendType
	CloseAfterSent bool
	EndReceivers   []string // may be the relayer
}

func (o OutgoingRequest) String() string {
	return fmt.Sprintf("sendtype=%s receivers=%s message=%s", o.SendType, PrettyIds(o.EndReceivers), o.Msg)
}

type Bytable interface {
	ToBytes() []byte
	FromBytes(bts []byte) error
}

type OutgoingMsg interface {
	Bytable
	GetType() int
	String() string
}

type RelayableOutgoingMsg interface {
	OutgoingMsg
	GetReceiverId() string // final receiver.
	GetOriginId() string   // original sender
}
