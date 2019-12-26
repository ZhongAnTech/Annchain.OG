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
	Name() string
}

type EventHandlerRegisterInfo struct {
	Type    EventType
	Name    string
	Handler EventHandler
}

type DefaultEventBus struct {
	knownNames map[EventType]string
	listeners  map[EventType][]EventHandler
	inited     bool       // do not use Mutex after initialization. It will downgrade performance
	mu         sync.Mutex // use only during initialization
}

func (e *DefaultEventBus) InitDefault() {
	e.listeners = make(map[EventType][]EventHandler)
	e.knownNames = make(map[EventType]string)
}

func (e *DefaultEventBus) ListenTo(regInfo EventHandlerRegisterInfo) {
	e.mu.Lock()
	defer e.mu.Unlock()
	l, ok := e.listeners[regInfo.Type]
	if !ok {
		l = []EventHandler{regInfo.Handler}
	} else {
		l = append(l, regInfo.Handler)
	}
	e.listeners[regInfo.Type] = l
	e.knownNames[regInfo.Type] = regInfo.Name
}
func (e *DefaultEventBus) Build() {
	e.inited = true
}

func (e *DefaultEventBus) Route(ev Event) {
	if !e.inited {
		panic("bad code. build eventbus before routing")
	}
	name := e.knownNames[ev.GetEventType()]
	logrus.WithField("type", name).WithField("v", ev).Debug("router received event")
	handlers, ok := e.listeners[ev.GetEventType()]
	if !ok {
		logrus.WithField("type", name).WithField("typecode", ev.GetEventType()).Warn("no event handler to handle event type")
		return
	}
	for _, handler := range handlers {
		logrus.WithField("handler", handler.Name()).Debug("handling")
		handler.HandleEvent(ev)
	}
	logrus.WithField("type", name).WithField("v", ev).Debug("router handled event")
}
