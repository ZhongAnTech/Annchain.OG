package ogcore

import "github.com/annchain/OG/eventbus"

type EventBus interface {
	Route(eventbus.Event)
}