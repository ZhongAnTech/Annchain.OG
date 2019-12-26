package events

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/model"
)

const (
	PingReceivedEventType eventbus.EventType = iota
	PongReceivedEventType
	QueryStatusRequestReceivedEventType  // global status query request is got
	QueryStatusResponseReceivedEventType // global status query response is got
	HeightBehindEventType                // my height is lower than others.
	TxReceivedEventType                  // a new tx list is received.
	SequencerReceivedEventType           // a new seq is received.
	ArchiveReceivedEventType
	ActionReceivedEventType
	NewTxDependencyFulfilledEventType  // a new tx is fully resolved (thus can be broadcasted)
	NeedSyncEventType                  // a hash is needed but not found locally (thus need sync)
	HeightSyncRequestReceivedEventType // someone is requesting a height
	TxsFetchedForResponseEventType     // txs are fetched from db and ready for response
	NewTxGeneratedEventType            // a new tx is generated from local
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

type NewTxDependencyFulfilledEvent struct {
	Tx types.Txi
}

func (m *NewTxDependencyFulfilledEvent) GetEventType() eventbus.EventType {
	return NewTxDependencyFulfilledEventType
}

type NeedSyncEvent struct {
	ParentHash      common.Hash
	ChildHash       common.Hash
	SendBloomfilter bool
}

func (m *NeedSyncEvent) GetEventType() eventbus.EventType {
	return NeedSyncEventType
}

type HeightSyncRequestReceivedEvent struct {
	Height    uint64
	Offset    uint32
	RequestId uint32
	Peer      communication.OgPeer
}

func (m *HeightSyncRequestReceivedEvent) GetEventType() eventbus.EventType {
	return HeightSyncRequestReceivedEventType
}

type TxsFetchedForResponseEvent struct {
	Txs       []types.Txi
	Height    uint64
	Offset    uint32
	RequestId uint32
	Peer      communication.OgPeer
}

func (m *TxsFetchedForResponseEvent) GetEventType() eventbus.EventType {
	return TxsFetchedForResponseEventType
}

type NewTxGeneratedEvent struct {
	Tx *types.Tx
}

func (m *NewTxGeneratedEvent) GetEventType() eventbus.EventType {
	return NewTxGeneratedEventType
}
