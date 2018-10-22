package og

import (
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

// MessageRouter is a bridge between hub and components
type MessageRouter struct {
	Hub    *Hub
	PingHandler                PingHandler
	PongHandler                PongHandler
	FetchByHashResponseHandler FetchByHashResponseHandler
	NewTxHandler               NewTxHandler
	NewTxsHandler              NewTxsHandler
	NewSequencerHandler        NewSequencerHandler
}

type ManagerConfig struct {
	AcquireTxQueueSize uint // length of the channel for tx acquiring
	BatchAcquireSize   uint // length of the buffer for batch tx acquire for a single node
}

type PingHandler interface {
	HandlePing()
}
type PongHandler interface {
	HandlePong()
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


func (m *MessageRouter) Start() {
	m.Hub.BroadcastMessage(MessageTypePing, []byte{})
}

func (m *MessageRouter) Stop() {

}

func (m *MessageRouter) Name() string {
	return "MessageRouter"
}

func (m *MessageRouter) RoutePing(*P2PMessage) {
	m.PingHandler.HandlePing()
}

func (m *MessageRouter) RoutePong(*P2PMessage) {
	m.PongHandler.HandlePong()
}

func (m *MessageRouter) RouteFetchByHashResponse(msg *P2PMessage) {
	syncResponse := types.MessageSyncResponse{}
	//bytebufferd := bytes.NewBuffer(nil)
	//bytebuffers := bytes.NewBuffer(msg.Message)
	//msgp.CopyToJSON(bytebufferd, bytebuffers)
	//fmt.Println(bytebufferd.String())

	_, err := syncResponse.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.Debug("invalid MessageSyncResponse format")
		return
	}
	if (syncResponse.Txs == nil || len(syncResponse.Txs) == 0) &&
		(syncResponse.Sequencers == nil || len(syncResponse.Sequencers) == 0) {
		logrus.Debug("empty MessageSyncResponse")
		return
	}

	m.FetchByHashResponseHandler.HandleFetchByHashResponse(syncResponse, msg.SourceID)
}

func (m *MessageRouter) RouteNewTx(msg *P2PMessage) {
	newTx := types.MessageNewTx{}

	_, err := newTx.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.WithError(err).Debug("invalid MessageNewTx format")
		return
	}
	if newTx.Tx == nil {
		logrus.Debug("empty MessageNewTx")
		return
	}

	m.NewTxHandler.HandleNewTx(newTx)
}

func (m *MessageRouter) RouteNewTxs(msg *P2PMessage) {
	//maybe received more transactions
	var err error
	newTxs := types.MessageNewTxs{}
	_, err = newTxs.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.WithError(err).Debug("invalid MessageNewTxs format")
		return
	}
	if newTxs.Txs == nil {
		logrus.Debug("Empty MessageNewTx")
		return
	}

	m.NewTxsHandler.HandleNewTxs(newTxs)
}
func (m *MessageRouter) RouteNewSequencer(msg *P2PMessage) {
	newSq := types.MessageNewSequencer{}
	_, err := newSq.UnmarshalMsg(msg.Message)
	if err != nil {
		logrus.WithError(err).Debug("invalid NewSequence format")
		return
	}
	if newSq.Sequencer == nil {
		logrus.Debug("empty NewSequence")
		return
	}

	m.NewSequencerHandler.HandleNewSequencer(newSq)
}

// BroadcastMessage send message to all peers
func (m *MessageRouter) BroadcastMessage(messageType MessageType, message []byte) {
	m.Hub.BroadcastMessage(messageType, message)
}

// UnicastMessageRandomly send message to a randomly chosen peer
func (m *MessageRouter) UnicastMessageRandomly(messageType MessageType, message []byte) {
	m.Hub.BroadcastMessageToRandom(messageType, message)
}
