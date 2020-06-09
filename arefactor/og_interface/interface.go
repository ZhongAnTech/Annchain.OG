package og_interface

type PeerJoinedEvent struct {
	PeerId string
}

type PeerJoinedEventSubscriber interface {
	EventChannelPeerJoined() chan *PeerJoinedEvent
}

type NodeInfoProvider interface {
	CurrentHeight() int64
	GetNetworkId() string
}

type NewHeightDetectedEvent struct {
	Height int64
	PeerId string
}

type NewHeightDetectedEventSubscriber interface {
	NewHeightDetectedEventChannel() chan *NewHeightDetectedEvent
}
