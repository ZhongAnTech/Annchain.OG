package og

import (
	"fmt"
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

	SequencerHeaderHandler SequencerHeaderHandler
	BodiesRequestHandler   BodiesRequestHandler
	BodiesResponseHandler  BodiesResponseHandler
	TxsRequestHandler      TxsRequestHandler
	TxsResponseHandler     TxsResponseHandler
	HeaderRequestHandler   HeaderRequestHandler
	HeaderResponseHandler  HeaderResponseHandler
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
	HandleFetchByHashRequest(req types.MessageSyncRequest, sourceID string)
}

type FetchByHashResponseHandler interface {
	HandleFetchByHashResponse(resp types.MessageSyncResponse, sourceID string)
}

type NewTxHandler interface {
	HandleNewTx(types.MessageNewTx)
}

type NewTxsHandler interface {
	HandleNewTxs(types.MessageNewTxs)
}

type NewSequencerHandler interface {
	HandleNewSequencer(types.MessageNewSequencer)
}

type SequencerHeaderHandler interface {
	HandleSequencerHeader(msgHeader types.MessageSequencerHeader, peerId string)
}

type BodiesRequestHandler interface {
	HandleBodiesRequest(msgReq types.MessageBodiesRequest, peerID string)
}

type BodiesResponseHandler interface {
	HandleBodiesResponse(request types.MessageBodiesResponse, peerId string)
}

type TxsRequestHandler interface {
	HandleTxsRequest(msgReq types.MessageTxsRequest, peerID string)
}

type TxsResponseHandler interface {
	HandleTxsResponse(request types.MessageTxsResponse)
}

type HeaderRequestHandler interface {
	HandleHeaderRequest(request types.MessageHeaderRequest, peerID string)
}

type HeaderResponseHandler interface {
	HandleHeaderResponse(headerMsg types.MessageHeaderResponse, peerID string)
}

func (m *MessageRouter) Start() {
	m.Hub.BroadcastMessage(MessageTypePing, []byte{})
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
	syncRequest := types.MessageSyncRequest{}
	_, err := syncRequest.UnmarshalMsg(msg.Message)
	if err != nil {
		msgLog.Debug("invalid MessageSyncRequest format")
		return
	}
	m.debugLog(msg,&syncRequest)
	m.FetchByHashRequestHandler.HandleFetchByHashRequest(syncRequest, msg.SourceID)
}

func (m *MessageRouter) RouteFetchByHashResponse(msg *P2PMessage) {
	syncResponse := types.MessageSyncResponse{}
	_, err := syncResponse.UnmarshalMsg(msg.Message)
	if err != nil {
		msgLog.Debug("invalid MessageSyncResponse format")
		return
	}
	m.debugLog(msg,&syncResponse)
	m.FetchByHashResponseHandler.HandleFetchByHashResponse(syncResponse, msg.SourceID)
}

func (m *MessageRouter) RouteNewTx(msg *P2PMessage) {
	newTx := types.MessageNewTx{}

	_, err := newTx.UnmarshalMsg(msg.Message)
	if err != nil {
		msgLog.WithError(err).Debug("invalid MessageNewTx format")
		return
	}
	m.debugLog(msg,&newTx)
	m.NewTxHandler.HandleNewTx(newTx)
}

func (m *MessageRouter) RouteNewTxs(msg *P2PMessage) {
	//maybe received more transactions
	var err error
	newTxs := types.MessageNewTxs{}
	_, err = newTxs.UnmarshalMsg(msg.Message)
	if err != nil {
		msgLog.WithError(err).Debug("invalid MessageNewTxs format")
		return
	}
	m.debugLog(msg,&newTxs)
	m.NewTxsHandler.HandleNewTxs(newTxs)
}
func (m *MessageRouter) RouteNewSequencer(msg *P2PMessage) {
	newSq := types.MessageNewSequencer{}
	_, err := newSq.UnmarshalMsg(msg.Message)
	if err != nil {
		msgLog.WithError(err).Debug("invalid NewSequence format")
		return
	}
	m.debugLog(msg,&newSq)
	m.NewSequencerHandler.HandleNewSequencer(newSq)
}

func (m *MessageRouter) RouteSequencerHeader(msg *P2PMessage) {
	msgHeader := types.MessageSequencerHeader{}
	_, err := msgHeader.UnmarshalMsg(msg.Message)
	if err != nil {
		msgLog.WithError(err).Debug("invalid MessageSequencerHeader format")
		return
	}
	m.debugLog(msg,&msgHeader)
	m.SequencerHeaderHandler.HandleSequencerHeader(msgHeader, msg.SourceID)
}
func (m *MessageRouter) RouteBodiesRequest(msg *P2PMessage) {

	msgReq := types.MessageBodiesRequest{}
	_, err := msgReq.UnmarshalMsg(msg.Message)
	if err != nil {
		msgLog.WithError(err).Debug("invalid MessageBodiesRequest format")
		return
	}
	m.debugLog(msg,&msgReq)
	m.BodiesRequestHandler.HandleBodiesRequest(msgReq, msg.SourceID)
}
func (m *MessageRouter) RouteBodiesResponse(msg *P2PMessage) {
	// A batch of block bodies arrived to one of our previous requests
	var request types.MessageBodiesResponse
	if _, err := request.UnmarshalMsg(msg.Message); err != nil {
		msgLog.WithError(err).Debug("invalid MessageBodiesResponse format")
		return
	}
	m.debugLog(msg,&request)
	m.BodiesResponseHandler.HandleBodiesResponse(request, msg.SourceID)
}
func (m *MessageRouter) RouteTxsRequest(msg *P2PMessage) {
	// Decode the retrieval message
	var msgReq types.MessageTxsRequest
	if _, err := msgReq.UnmarshalMsg(msg.Message); err != nil {
		msgLog.WithError(err).WithField("msg", fmt.Sprintf("%v", msg)).Debug("unmarshal message")
		return
	}
	m.debugLog(msg,&msgReq)
	m.TxsRequestHandler.HandleTxsRequest(msgReq, msg.SourceID)

}
func (m *MessageRouter) RouteTxsResponse(msg *P2PMessage) {
	// A batch of block bodies arrived to one of our previous requests
	var request types.MessageTxsResponse
	if _, err := request.UnmarshalMsg(msg.Message); err != nil {
		msgLog.WithError(err).WithField("msg", fmt.Sprintf("%v", msg)).Debug("unmarshal message")
		return
	}
	m.debugLog(msg,&request)
	m.TxsResponseHandler.HandleTxsResponse(request)

}
func (m *MessageRouter) RouteHeaderRequest(msg *P2PMessage) {
	// Decode the complex header query
	var query types.MessageHeaderRequest
	if _, err := query.UnmarshalMsg(msg.Message); err != nil {
		//return errResp(ErrDecode, "%v: %v", msg, err)
		msgLog.WithError(err).WithField("msg", fmt.Sprintf("%v", msg)).Debug("unmarshal message")
		return
	}
	m.debugLog(msg,&query)
	m.HeaderRequestHandler.HandleHeaderRequest(query, msg.SourceID)
}

func (m *MessageRouter) RouteHeaderResponse(msg *P2PMessage) {
	// A batch of headers arrived to one of our previous requests
	var headerMsg types.MessageHeaderResponse
	if _, err := headerMsg.UnmarshalMsg(msg.Message); err != nil {
		msgLog.WithError(err).WithField("msg", fmt.Sprintf("%v", msg)).Debug("unmarshal message")
		return
	}
	m.debugLog(msg,&headerMsg)
	m.HeaderResponseHandler.HandleHeaderResponse(headerMsg, msg.SourceID)
}

func (m*MessageRouter)debugLog(msg *P2PMessage , message types.Message) (){
	msgLog.WithField("type",msg.MessageType.String()).WithField("from", msg.SourceID).WithField(
		"q", message.String()).Debug("received a message")
}


// BroadcastMessage send message to all peers
func (m *MessageRouter) BroadcastMessage(messageType MessageType, message []byte) {
	m.Hub.BroadcastMessage(messageType, message)
}

// UnicastMessageRandomly send message to a randomly chosen peer
func (m *MessageRouter) UnicastMessageRandomly(messageType MessageType, message []byte) {
	m.Hub.BroadcastMessageToRandom(messageType, message)
}
