package transport_event

import (
	"fmt"
	"github.com/annchain/OG/common/hexutil"
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

type IncomingLetter struct {
	Msg  *WireMessage
	From string
}

func (i IncomingLetter) String() string {
	return fmt.Sprintf("msgType=%d from=%s bytes=%s", i.Msg.MsgType, i.From, hexutil.Encode(i.Msg.ContentBytes))
}

type OutgoingLetter struct {
	Msg            OutgoingMsg
	SendType       SendType
	CloseAfterSent bool
	EndReceivers   []string // may be the relayer
}

func (o OutgoingLetter) String() string {
	return fmt.Sprintf("sendtype=%s receivers=%s message=%s close=%t", o.SendType, PrettyIds(o.EndReceivers), o.Msg, o.CloseAfterSent)
}

type Bytable interface {
	ToBytes() []byte
	FromBytes(bts []byte) error
}

type OutgoingMsg interface {
	Bytable
	msgp.Marshaler
	msgp.Unmarshaler
	GetTypeValue() int
	String() string
}

type RelayableOutgoingMsg interface {
	OutgoingMsg
	GetReceiverId() string // final receiver.
	GetOriginId() string   // original sender
}
