package transport_event

type NewIncomingMessageEventSubscriber interface {
	GetNewIncomingMessageEventChannel() chan *WireMessage
}

type NewOutgoingMessageEventSubscriber interface {
	GetNewOutgoingMessageEventChannel() chan *OutgoingRequest
}
