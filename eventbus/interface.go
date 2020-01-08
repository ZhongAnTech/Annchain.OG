package eventbus

type EventBus interface {
	Route(Event)
}
