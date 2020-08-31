package consts

type EventType int

const (
	NewIncomingMessageEvent EventType = iota
	NewOutgoingMessageEvent
	PeerConnectedEvent
	PeerJoinedEvent
	PeerLeftEvent
	UnknownNeededEvent
	NewHeightDetectedEvent
	NewHeightBlockSyncedEvent
	LocalHeightUpdatedEvent
)

var EventCodeTextMap = map[EventType]string{
	NewIncomingMessageEvent:   "NewIncomingMessageEvent",
	NewOutgoingMessageEvent:   "NewOutgoingMessageEvent",
	PeerConnectedEvent:        "PeerConnectedEvent",
	PeerJoinedEvent:           "PeerJoinedEvent",
	PeerLeftEvent:             "PeerLeftEvent",
	UnknownNeededEvent:        "UnknownNeededEvent",
	NewHeightDetectedEvent:    "NewHeightDetectedEvent",
	NewHeightBlockSyncedEvent: "NewHeightBlockSyncedEvent",
	LocalHeightUpdatedEvent:   "LocalHeightUpdatedEvent",
}
