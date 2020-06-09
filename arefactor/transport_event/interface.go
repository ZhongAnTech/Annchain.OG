package transport_event

type PeerConnectedEvent struct {
	PeerId string
}

type PeerConnectedEventSubscriber interface {
	Name() string
	GetPeerConnectedEventChannel() chan *PeerConnectedEvent
}

type NewIncomingMessageEventSubscriber interface {
	Name() string
	NewIncomingMessageEventChannel() chan *IncomingLetter
}

type NewOutgoingMessageEventSubscriber interface {
	Name() string
	NewOutgoingMessageEventChannel() chan *OutgoingLetter
}
