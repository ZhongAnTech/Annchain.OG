package transport_event

type NewIncomingMessageEventSubscriber interface {
	Name() string
	GetNewIncomingMessageEventChannel() chan *IncomingLetter
}

type NewOutgoingMessageEventSubscriber interface {
	Name() string
	GetNewOutgoingMessageEventChannel() chan *OutgoingLetter
}
