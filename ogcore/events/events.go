package events

import "github.com/annchain/OG/eventbus"

const (
	PingReceivedEventType eventbus.EventType = iota
	PongReceivedEventType
)

type PingReceivedEvent struct{}

func (m *PingReceivedEvent) GetEventType() eventbus.EventType {
	return PingReceivedEventType
}

type PongReceivedEvent struct{}

func (m *PongReceivedEvent) GetEventType() eventbus.EventType {
	return PongReceivedEventType
}
