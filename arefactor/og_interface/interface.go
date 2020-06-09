package og_interface

type HeightProvider interface {
	Height() int
}

type PeerConnectedEvent struct {
	peerId string
}

type PeerConnectedEventSubscriber interface {
	PeerConnectedEventChannel() chan *PeerConnectedEvent
}
