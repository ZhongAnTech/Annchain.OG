package transport_event

import (
	"fmt"
	"github.com/tinylib/msgp/msgp"
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
	Msg          *OutgoingMsg
	SendType     SendType
	EndReceivers []string
}

func (o OutgoingRequest) String() string {
	return fmt.Sprintf("sendtype=%s receivers=%s msg=%s", o.SendType, PrettyIds(o.EndReceivers), o.Msg)
}

// OutgoingMsg is NOT for transportation. It is only an internal structure
type OutgoingMsg struct {
	Typev    int
	SenderId string // no need to fill it when sending
	Content  msgp.Marshaler
}

func (m OutgoingMsg) String() string {
	return fmt.Sprintf("[type:%d sender=%s content=%s]", m.Typev, PrettyId(m.SenderId), m.Content)
}
