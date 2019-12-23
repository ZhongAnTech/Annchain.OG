package eventbus

import (
	"github.com/sirupsen/logrus"
	"sync"
)

type EventType uint8

type Event interface {
	GetEventType() EventType
}
type EventHandler interface {
	HandleEvent(Event)
}

type EventBus struct {
	listeners map[EventType][]EventHandler
	inited    bool       // do not use Mutex after initialization. It will downgrade performance
	mu        sync.Mutex // use only during initialization
}

func (e *EventBus) InitDefault() {
	e.listeners = make(map[EventType][]EventHandler)
}

func (e *EventBus) ListenTo(t EventType, handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	l, ok := e.listeners[t]
	if !ok {
		l = []EventHandler{handler}
	} else {
		l = append(l, handler)
	}
	e.listeners[t] = l
}
func (e *EventBus) Build() {
	e.inited = true
}

func (e *EventBus) Route(ev Event) {
	if !e.inited {
		panic("bad code. build eventbus before routing")
	}
	handlers, ok := e.listeners[ev.GetEventType()]
	if !ok {
		logrus.WithField("type", ev.GetEventType()).Warn("no event handler to handle event type")
		return
	}
	for _, handler := range handlers {
		handler.HandleEvent(ev)
	}
}
