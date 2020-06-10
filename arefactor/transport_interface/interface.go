package transport_interface

type PeerEvent struct {
	PeerId string
}

type PeerConnectedEventSubscriber interface {
	Name() string
	GetPeerConnectedEventChannel() chan *PeerEvent
}

type PeerDisconnectedEventSubscriber interface {
	Name() string
	GetPeerDisconnectedEventChannel() chan *PeerEvent
}

type NewIncomingMessageEventSubscriber interface {
	Name() string
	NewIncomingMessageEventChannel() chan *IncomingLetter
}

type NewOutgoingMessageEventSubscriber interface {
	Name() string
	NewOutgoingMessageEventChannel() chan *OutgoingLetter
}
