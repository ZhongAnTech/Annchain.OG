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

type EventRegisterInfo struct {
	Type    EventType
	Handler EventHandler
}

type DefaultEventBus struct {
	listeners map[EventType][]EventHandler
	inited    bool       // do not use Mutex after initialization. It will downgrade performance
	mu        sync.Mutex // use only during initialization
}

func (e *DefaultEventBus) InitDefault() {
	e.listeners = make(map[EventType][]EventHandler)
}

func (e *DefaultEventBus) ListenTo(regInfo EventRegisterInfo) {
	e.mu.Lock()
	defer e.mu.Unlock()
	l, ok := e.listeners[regInfo.Type]
	if !ok {
		l = []EventHandler{regInfo.Handler}
	} else {
		l = append(l, regInfo.Handler)
	}
	e.listeners[regInfo.Type] = l
}
func (e *DefaultEventBus) Build() {
	e.inited = true
}

func (e *DefaultEventBus) Route(ev Event) {
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
