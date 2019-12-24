package events

import (
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/model"
)

const (
	PingReceivedEventType eventbus.EventType = iota
	PongReceivedEventType
	QueryStatusRequestReceivedEventType
	QueryStatusResponseReceivedEventType
	HeightBehindEventType

	TxReceivedEventType
	SequencerReceivedEventType
	ArchiveReceivedEventType
	ActionReceivedEventType
)

type PingReceivedEvent struct{}

func (m *PingReceivedEvent) GetEventType() eventbus.EventType {
	return PingReceivedEventType
}

type PongReceivedEvent struct{}

func (m *PongReceivedEvent) GetEventType() eventbus.EventType {
	return PongReceivedEventType
}

type QueryStatusReceivedEvent struct {
}

func (m *QueryStatusReceivedEvent) GetEventType() eventbus.EventType {
	return QueryStatusRequestReceivedEventType
}

type QueryStatusResponseReceivedEvent struct {
	StatusData model.OgStatusData
}

func (m *QueryStatusResponseReceivedEvent) GetEventType() eventbus.EventType {
	return QueryStatusResponseReceivedEventType
}

type HeightBehindEvent struct {
	LatestKnownHeight uint64
}

func (m *HeightBehindEvent) GetEventType() eventbus.EventType {
	return HeightBehindEventType
}

type TxReceivedEvent struct {
	Tx *types.Tx
}

func (m *TxReceivedEvent) GetEventType() eventbus.EventType {
	return TxReceivedEventType
}

type SequencerReceivedEvent struct {
	Sequencer *types.Sequencer
}

func (m *SequencerReceivedEvent) GetEventType() eventbus.EventType {
	return SequencerReceivedEventType
}