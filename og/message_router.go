package og

import (
	"github.com/annchain/OG/types"
)

// MessageRouter is a bridge between hub and components
type MessageRouter struct {
	Hub                        *Hub
	PingHandler                PingHandler
	PongHandler                PongHandler
	FetchByHashRequestHandler  FetchByHashHandlerRequest
	FetchByHashResponseHandler FetchByHashResponseHandler
	NewTxHandler               NewTxHandler
	NewTxsHandler              NewTxsHandler
	NewSequencerHandler        NewSequencerHandler
	GetMsgHandler              GetMsgHandler
	SequencerHeaderHandler     SequencerHeaderHandler
	BodiesRequestHandler       BodiesRequestHandler
	BodiesResponseHandler      BodiesResponseHandler
	TxsRequestHandler          TxsRequestHandler
	TxsResponseHandler         TxsResponseHandler
	HeaderRequestHandler       HeaderRequestHandler
	HeaderResponseHandler      HeaderResponseHandler
}

type ManagerConfig struct {
	AcquireTxQueueSize uint // length of the channel for tx acquiring
	BatchAcquireSize   uint // length of the buffer for batch tx acquire for a single node
}

type PingHandler interface {
	HandlePing(peerId string)
}
type PongHandler interface {
	HandlePong()
}

type FetchByHashHandlerRequest interface {
	HandleFetchByHashRequest(req *types.MessageSyncRequest, sourceID string)
}

type FetchByHashResponseHandler interface {
	HandleFetchByHashResponse(resp *types.MessageSyncResponse, sourceID string)
}

type NewTxHandler interface {
	HandleNewTx(msg *types.MessageNewTx, peerId string)
}

type GetMsgHandler interface {
	HandleGetMsg(msg *types.MessageGetMsg, peerId string)
}

type NewTxsHandler interface {
	HandleNewTxs(newTxs *types.MessageNewTxs, peerId string)
}

type NewSequencerHandler interface {
	HandleNewSequencer(msg *types.MessageNewSequencer, peerId string)
}

type SequencerHeaderHandler interface {
	HandleSequencerHeader(msgHeader *types.MessageSequencerHeader, peerId string)
}

type BodiesRequestHandler interface {
	HandleBodiesRequest(msgReq *types.MessageBodiesRequest, peerID string)
}

type BodiesResponseHandler interface {
	HandleBodiesResponse(request *types.MessageBodiesResponse, peerId string)
}

type TxsRequestHandler interface {
	HandleTxsRequest(msgReq *types.MessageTxsRequest, peerID string)
}

type TxsResponseHandler interface {
	HandleTxsResponse(request *types.MessageTxsResponse)
}

type HeaderRequestHandler interface {
	HandleHeaderRequest(request *types.MessageHeaderRequest, peerID string)
}

type HeaderResponseHandler interface {
	HandleHeaderResponse(headerMsg *types.MessageHeaderResponse, peerID string)
}

func (m *MessageRouter) Start() {
	m.Hub.BroadcastMessage(MessageTypePing, &types.MessagePing{Data: []byte{}})
}

func (m *MessageRouter) Stop() {

}

func (m *MessageRouter) Name() string {
	return "MessageRouter"
}

func (m *MessageRouter) RoutePing(msg *P2PMessage) {
	m.PingHandler.HandlePing(msg.SourceID)
}

func (m *MessageRouter) RoutePong(*P2PMessage) {
	m.PongHandler.HandlePong()
}
func (m *MessageRouter) RouteFetchByHashRequest(msg *P2PMessage) {
	m.FetchByHashRequestHandler.HandleFetchByHashRequest(msg.Message.(*types.MessageSyncRequest), msg.SourceID)
}

func (m *MessageRouter) RouteFetchByHashResponse(msg *P2PMessage) {
	m.FetchByHashResponseHandler.HandleFetchByHashResponse(msg.Message.(*types.MessageSyncResponse), msg.SourceID)
}

func (m *MessageRouter) RouteNewTx(msg *P2PMessage) {
	m.NewTxHandler.HandleNewTx(msg.Message.(*types.MessageNewTx), msg.SourceID)
}

func (m *MessageRouter) RouteNewTxs(msg *P2PMessage) {
	//maybe received more transactions
	m.NewTxsHandler.HandleNewTxs(msg.Message.(*types.MessageNewTxs), msg.SourceID)
}

func (m *MessageRouter) RouteNewSequencer(msg *P2PMessage) {
	m.NewSequencerHandler.HandleNewSequencer(msg.Message.(*types.MessageNewSequencer), msg.SourceID)
}

func (m *MessageRouter) RouteGetMsg(msg *P2PMessage) {
	m.GetMsgHandler.HandleGetMsg(msg.Message.(*types.MessageGetMsg), msg.SourceID)
}

func (m *MessageRouter) RouteSequencerHeader(msg *P2PMessage) {
	m.SequencerHeaderHandler.HandleSequencerHeader(msg.Message.(*types.MessageSequencerHeader), msg.SourceID)
}
func (m *MessageRouter) RouteBodiesRequest(msg *P2PMessage) {

	m.BodiesRequestHandler.HandleBodiesRequest(msg.Message.(*types.MessageBodiesRequest), msg.SourceID)
}

func (m *MessageRouter) RouteBodiesResponse(msg *P2PMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.BodiesResponseHandler.HandleBodiesResponse(msg.Message.(*types.MessageBodiesResponse), msg.SourceID)
}

func (m *MessageRouter) RouteTxsRequest(msg *P2PMessage) {
	// Decode the retrieval message
	m.TxsRequestHandler.HandleTxsRequest(msg.Message.(*types.MessageTxsRequest), msg.SourceID)

}
func (m *MessageRouter) RouteTxsResponse(msg *P2PMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.TxsResponseHandler.HandleTxsResponse(msg.Message.(*types.MessageTxsResponse))

}
func (m *MessageRouter) RouteHeaderRequest(msg *P2PMessage) {
	// Decode the complex header query
	m.HeaderRequestHandler.HandleHeaderRequest(msg.Message.(*types.MessageHeaderRequest), msg.SourceID)
}

func (m *MessageRouter) RouteHeaderResponse(msg *P2PMessage) {
	// A batch of headers arrived to one of our previous requests
	m.HeaderResponseHandler.HandleHeaderResponse(msg.Message.(*types.MessageHeaderResponse), msg.SourceID)
}

// BroadcastMessage send message to all peers
func (m *MessageRouter) BroadcastMessage(messageType MessageType, message types.Message) {
	m.Hub.BroadcastMessage(messageType, message)
}

// MulticastMessage send message to a randomly chosen peer
func (m *MessageRouter) MulticastMessage(messageType MessageType, message types.Message) {
	m.Hub.MulticastMessage(messageType, message)
}

func (m *MessageRouter) MulticastToSource(messageType MessageType, message types.Message, sourceMsgHash *types.Hash) {
	m.Hub.MulticastToSource(messageType, message, sourceMsgHash)
}

func (m *MessageRouter) BroadcastMessageWithLink(messageType MessageType, message types.Message) {
	m.Hub.BroadcastMessageWithLink(messageType, message)
}
