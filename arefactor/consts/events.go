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
	NewBlockProducedEvent

	SequencerReceivedEvent
	TxReceivedEvent
	IntsReceivedEvent
	SequencerFulfilledEvent
	TxFulfilledEvent
	IntsFulfilledEvent
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
	NewBlockProducedEvent:     "NewBlockProducedEvent",
	IntsReceivedEvent:         "IntsReceivedEvent",
}
