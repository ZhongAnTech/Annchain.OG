package events

import (
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/model"
)

const (
	PingReceivedEventType eventbus.EventType = iota + 1
	PongReceivedEventType
	QueryStatusRequestReceivedEventType  // global status query request is got
	QueryStatusResponseReceivedEventType // global status query response is got
	HeightBehindEventType                // my height is lower than others.

	TxReceivedEventType        // a new tx list is received.
	SequencerReceivedEventType // a new seq is received.
	ArchiveReceivedEventType
	ActionReceivedEventType
	NewTxiDependencyFulfilledEventType    // a new tx is fully resolved (thus can be broadcasted)

	NeedSyncTxEventType                   // a hash is needed but not found locally (thus need sync)
	HeightSyncRequestReceivedEventType    // someone is requesting a height
	BatchSyncRequestReceivedEventType     // someone is requesting some txs by hash
	TxsFetchedForResponseEventType        // txs are fetched from db and ready for response
	NewTxLocallyGeneratedEventType        // a new tx is generated from local

	NewSequencerLocallyGeneratedEventType // a new seq is generated from local (by annsensus)
	NewTxReceivedInPoolEventType          // a new tx is received in the pool and to be processed. (including sequencer)
	SequencerBatchConfirmedEventType      // a sequencer and its txs are all confirmed in a batch
	SequencerConfirmedEventType           // a sequencer is received and validated to be on the graph
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

type NewTxiDependencyFulfilledEvent struct {
	Txi types.Txi
}

func (m *NewTxiDependencyFulfilledEvent) GetEventType() eventbus.EventType {
	return NewTxiDependencyFulfilledEventType
}

type NeedSyncTxEvent struct {
	Hash            types2.Hash
	SpecifiedSource *communication.OgPeer
	//ParentHash      common.Hash
	//ChildHash       common.Hash
	//SendBloomfilter bool
}

func (m *NeedSyncTxEvent) GetEventType() eventbus.EventType {
	return NeedSyncTxEventType
}

type HeightSyncRequestReceivedEvent struct {
	Height    uint64
	Offset    int
	RequestId uint32
	Peer      *communication.OgPeer
}

func (m *HeightSyncRequestReceivedEvent) GetEventType() eventbus.EventType {
	return HeightSyncRequestReceivedEventType
}

type BatchSyncRequestReceivedEvent struct {
	Hashes    types2.Hashes
	RequestId uint32
	Peer      *communication.OgPeer
}

func (m *BatchSyncRequestReceivedEvent) GetEventType() eventbus.EventType {
	return BatchSyncRequestReceivedEventType
}

type TxsFetchedForResponseEvent struct {
	Txs       types.Txis
	Height    uint64
	Offset    int
	RequestId uint32
	Peer      *communication.OgPeer
}

func (m *TxsFetchedForResponseEvent) GetEventType() eventbus.EventType {
	return TxsFetchedForResponseEventType
}

type NewTxLocallyGeneratedEvent struct {
	Tx               *types.Tx
	RequireBroadcast bool
}

func (m *NewTxLocallyGeneratedEvent) GetEventType() eventbus.EventType {
	return NewTxLocallyGeneratedEventType
}

type NewSequencerLocallyGeneratedEvent struct {
	Sequencer *types.Sequencer
}

func (m *NewSequencerLocallyGeneratedEvent) GetEventType() eventbus.EventType {
	return NewSequencerLocallyGeneratedEventType
}

type NewTxReceivedInPoolEvent struct {
	Tx types.Txi
}

func (m *NewTxReceivedInPoolEvent) GetEventType() eventbus.EventType {
	return NewTxReceivedInPoolEventType
}

type SequencerBatchConfirmedEvent struct {
	Elders map[types2.Hash]types.Txi
}

func (m *SequencerBatchConfirmedEvent) GetEventType() eventbus.EventType {
	return SequencerBatchConfirmedEventType
}

type SequencerConfirmedEvent struct {
	Sequencer *types.Sequencer
}

func (m *SequencerConfirmedEvent) GetEventType() eventbus.EventType {
	return SequencerConfirmedEventType
}
