package og_interface

type PeerJoinedEvent struct {
	PeerId string
}
type PeerLeftEvent struct {
	PeerId string
}

type PeerJoinedEventSubscriber interface {
	EventChannelPeerJoined() chan *PeerJoinedEvent
}

type PeerLeftEventSubscriber interface {
	EventChannelPeerLeft() chan *PeerLeftEvent
}

type NodeInfoProvider interface {
	CurrentHeight() int64
	GetNetworkId() string
}

type NewHeightDetectedEvent struct {
	Height int64
	PeerId string
}

type AccountHolder interface {
	ProvidePrivateKey(createIfMissing bool) ([]byte, error)
}

type NewHeightDetectedEventSubscriber interface {
	Name() string
	NewHeightDetectedEventChannel() chan *NewHeightDetectedEvent
}
