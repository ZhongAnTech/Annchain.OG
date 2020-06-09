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
	GetNewIncomingMessageEventChannel() chan *IncomingLetter
}

type NewOutgoingMessageEventSubscriber interface {
	Name() string
	NewOutgoingMessageEventChannel() chan *OutgoingLetter
}
