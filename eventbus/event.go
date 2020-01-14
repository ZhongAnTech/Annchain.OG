package eventbus

import (
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
)

type EventType uint8

type Event interface {
	GetEventType() EventType
}
type EventHandler interface {
	HandlerDescription(EventType) string
	HandleEvent(Event)
	Name() string
}

type EventHandlerRegisterInfo struct {
	Type    EventType
	Name    string
	Handler EventHandler
}

type DefaultEventBus struct {
	ID         int
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

// Eventbus must be built before events are to be received.
// This is an commit from programmer, showing that all modules are inited and well-prepared to receive events.
func (e *DefaultEventBus) Build() {
	e.inited = true
}

func (e *DefaultEventBus) Route(ev Event) {
	if !e.inited {
		panic("bad code. build eventbus before routing")
	}
	name, ok := e.knownNames[ev.GetEventType()]
	if !ok {
		name = strconv.Itoa(int(ev.GetEventType()))
	}
	logrus.WithField("me", e.ID).WithField("type", name).WithField("v", ev).Debug("router received event")
	handlers, ok := e.listeners[ev.GetEventType()]
	if !ok {
		logrus.WithField("me", e.ID).WithField("type", name).WithField("typecode", ev.GetEventType()).Warn("no event handler to handle event type")
		return
	}
	for _, handler := range handlers {
		logrus.WithFields(logrus.Fields{
			"me":      e.ID,
			"handler": handler.Name(),
			"desc":    handler.HandlerDescription(ev.GetEventType()),
		}).Debug("handling")
		handler.HandleEvent(ev)
	}

	logrus.WithField("me", e.ID).WithField("type", name).WithField("v", ev).Debug("router handled event")
}
