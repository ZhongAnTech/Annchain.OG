// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package og

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types/p2p_message"
)

// MessageRouter is a bridge between hub and components
type MessageRouter struct {
	Hub                                 *Hub
	PingHandler                         PingHandler
	PongHandler                         PongHandler
	FetchByHashRequestHandler           FetchByHashHandlerRequest
	FetchByHashResponseHandler          FetchByHashResponseHandler
	NewTxHandler                        NewTxHandler
	NewTxsHandler                       NewTxsHandler
	NewSequencerHandler                 NewSequencerHandler
	GetMsgHandler                       GetMsgHandler
	ControlMsgHandler                   ControlMsgHandler
	SequencerHeaderHandler              SequencerHeaderHandler
	BodiesRequestHandler                BodiesRequestHandler
	BodiesResponseHandler               BodiesResponseHandler
	TxsRequestHandler                   TxsRequestHandler
	TxsResponseHandler                  TxsResponseHandler
	HeaderRequestHandler                HeaderRequestHandler
	HeaderResponseHandler               HeaderResponseHandler
	CampaignHandler                     CampaignHandler
	TermChangeHandler                   TermChangeHandler
	ArchiveHandler                      ArchiveHandler
	ActionTxHandler                     ActionTxHandler
	ConsensusDkgDealHandler             ConsensusDkgDealHandler
	ConsensusDkgDealResponseHandler     ConsensusDkgDealResponseHandler
	ConsensusDkgSigSetsHandler          ConsensusDkgSigSetsHandler
	ConsensusDkgGenesisPublicKeyHandler ConsensusDkgGenesisPublicKeyHandler
	ConsensusHandler                    ConsensusHandler
	TermChangeRequestHandler            TermChangeRequestHandler
	TermChangeResponseHandler           TermChangeResponseHandler
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
	HandleFetchByHashRequest(req *p2p_message.MessageSyncRequest, sourceID string)
}

type FetchByHashResponseHandler interface {
	HandleFetchByHashResponse(resp *p2p_message.MessageSyncResponse, sourceID string)
}

type NewTxHandler interface {
	HandleNewTx(msg *p2p_message.MessageNewTx, peerId string)
}

type GetMsgHandler interface {
	HandleGetMsg(msg *p2p_message.MessageGetMsg, peerId string)
}

type ControlMsgHandler interface {
	HandleControlMsg(msg *p2p_message.MessageControl, peerId string)
}

type NewTxsHandler interface {
	HandleNewTxs(newTxs *p2p_message.MessageNewTxs, peerId string)
}

type NewSequencerHandler interface {
	HandleNewSequencer(msg *p2p_message.MessageNewSequencer, peerId string)
}

type SequencerHeaderHandler interface {
	HandleSequencerHeader(msgHeader *p2p_message.MessageSequencerHeader, peerId string)
}

type BodiesRequestHandler interface {
	HandleBodiesRequest(msgReq *p2p_message.MessageBodiesRequest, peerID string)
}

type BodiesResponseHandler interface {
	HandleBodiesResponse(request *p2p_message.MessageBodiesResponse, peerId string)
}

type TxsRequestHandler interface {
	HandleTxsRequest(msgReq *p2p_message.MessageTxsRequest, peerID string)
}

type TxsResponseHandler interface {
	HandleTxsResponse(request *p2p_message.MessageTxsResponse)
}

type HeaderRequestHandler interface {
	HandleHeaderRequest(request *p2p_message.MessageHeaderRequest, peerID string)
}

type HeaderResponseHandler interface {
	HandleHeaderResponse(headerMsg *p2p_message.MessageHeaderResponse, peerID string)
}

type CampaignHandler interface {
	HandleCampaign(request *p2p_message.MessageCampaign, peerId string)
}

type TermChangeHandler interface {
	HandleTermChange(request *p2p_message.MessageTermChange, peerId string)
}

type ArchiveHandler interface {
	HandleArchive(request *p2p_message.MessageNewArchive, peerId string)
}

type ActionTxHandler interface {
	HandleActionTx(request *p2p_message.MessageNewActionTx, perid string)
}

type ConsensusDkgDealHandler interface {
	HandleConsensusDkgDeal(request *p2p_message.MessageConsensusDkgDeal, peerId string)
}

type ConsensusDkgDealResponseHandler interface {
	HandleConsensusDkgDealResponse(request *p2p_message.MessageConsensusDkgDealResponse, peerId string)
}

type ConsensusDkgSigSetsHandler interface {
	HandleConsensusDkgSigSets(request *p2p_message.MessageConsensusDkgSigSets, peerId string)
}

type ConsensusDkgGenesisPublicKeyHandler interface {
	HandleConsensusDkgGenesisPublicKey(request *p2p_message.MessageConsensusDkgGenesisPublicKey, peerId string)
}
type ConsensusHandler interface {
	HandleConsensus(request bft.BftMessage, peerId string)
}
type TermChangeRequestHandler interface {
	HandleTermChangeRequest(request *p2p_message.MessageTermChangeRequest, peerId string)
}
type TermChangeResponseHandler interface {
	HandleTermChangeResponse(request *p2p_message.MessageTermChangeResponse, peerId string)
}

func (m *MessageRouter) Start() {
	m.Hub.BroadcastMessage(message.MessageTypePing, &p2p_message.MessagePing{Data: []byte{}})
}

func (m *MessageRouter) Stop() {

}

func (m *MessageRouter) Name() string {
	return "MessageRouter"
}

func (m *MessageRouter) RoutePing(msg *OGMessage) {
	m.PingHandler.HandlePing(msg.SourceID)
}

func (m *MessageRouter) RoutePong(*OGMessage) {
	m.PongHandler.HandlePong()
}
func (m *MessageRouter) RouteFetchByHashRequest(msg *OGMessage) {
	m.FetchByHashRequestHandler.HandleFetchByHashRequest(msg.Message.(*p2p_message.MessageSyncRequest), msg.SourceID)
}

func (m *MessageRouter) RouteFetchByHashResponse(msg *OGMessage) {
	m.FetchByHashResponseHandler.HandleFetchByHashResponse(msg.Message.(*p2p_message.MessageSyncResponse), msg.SourceID)
}

func (m *MessageRouter) RouteNewTx(msg *OGMessage) {
	m.NewTxHandler.HandleNewTx(msg.Message.(*p2p_message.MessageNewTx), msg.SourceID)
}

func (m *MessageRouter) RouteNewTxs(msg *OGMessage) {
	//maybe received more transactions
	m.NewTxsHandler.HandleNewTxs(msg.Message.(*p2p_message.MessageNewTxs), msg.SourceID)
}

func (m *MessageRouter) RouteNewSequencer(msg *OGMessage) {
	m.NewSequencerHandler.HandleNewSequencer(msg.Message.(*p2p_message.MessageNewSequencer), msg.SourceID)
}

func (m *MessageRouter) RouteGetMsg(msg *OGMessage) {
	m.GetMsgHandler.HandleGetMsg(msg.Message.(*p2p_message.MessageGetMsg), msg.SourceID)
}

func (m *MessageRouter) RouteControlMsg(msg *OGMessage) {
	m.ControlMsgHandler.HandleControlMsg(msg.Message.(*p2p_message.MessageControl), msg.SourceID)
}

func (m *MessageRouter) RouteSequencerHeader(msg *OGMessage) {
	m.SequencerHeaderHandler.HandleSequencerHeader(msg.Message.(*p2p_message.MessageSequencerHeader), msg.SourceID)
}
func (m *MessageRouter) RouteBodiesRequest(msg *OGMessage) {

	m.BodiesRequestHandler.HandleBodiesRequest(msg.Message.(*p2p_message.MessageBodiesRequest), msg.SourceID)
}

func (m *MessageRouter) RouteBodiesResponse(msg *OGMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.BodiesResponseHandler.HandleBodiesResponse(msg.Message.(*p2p_message.MessageBodiesResponse), msg.SourceID)
}

func (m *MessageRouter) RouteTxsRequest(msg *OGMessage) {
	// Decode the retrieval Message
	m.TxsRequestHandler.HandleTxsRequest(msg.Message.(*p2p_message.MessageTxsRequest), msg.SourceID)

}
func (m *MessageRouter) RouteTxsResponse(msg *OGMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.TxsResponseHandler.HandleTxsResponse(msg.Message.(*p2p_message.MessageTxsResponse))

}
func (m *MessageRouter) RouteHeaderRequest(msg *OGMessage) {
	// Decode the complex header query
	m.HeaderRequestHandler.HandleHeaderRequest(msg.Message.(*p2p_message.MessageHeaderRequest), msg.SourceID)
}

func (m *MessageRouter) RouteHeaderResponse(msg *OGMessage) {
	// A batch of headers arrived to one of our previous requests
	m.HeaderResponseHandler.HandleHeaderResponse(msg.Message.(*p2p_message.MessageHeaderResponse), msg.SourceID)
}

func (m *MessageRouter) RouteCampaign(msg *OGMessage) {
	// A batch of headers arrived to one of our previous requests
	m.CampaignHandler.HandleCampaign(msg.Message.(*p2p_message.MessageCampaign), msg.SourceID)
}

func (m *MessageRouter) RouteTermChange(msg *OGMessage) {
	m.TermChangeHandler.HandleTermChange(msg.Message.(*p2p_message.MessageTermChange), msg.SourceID)
}

func (m *MessageRouter) RouteArchive(msg *OGMessage) {
	m.ArchiveHandler.HandleArchive(msg.Message.(*p2p_message.MessageNewArchive), msg.SourceID)
}

func (m *MessageRouter) RouteActionTx(msg *OGMessage) {
	m.ActionTxHandler.HandleActionTx(msg.Message.(*p2p_message.MessageNewActionTx), msg.SourceID)
}

func (m *MessageRouter) RouteConsensusDkgDeal(msg *OGMessage) {
	m.ConsensusDkgDealHandler.HandleConsensusDkgDeal(msg.Message.(*p2p_message.MessageConsensusDkgDeal), msg.SourceID)
}

func (m *MessageRouter) RouteConsensusDkgDealResponse(msg *OGMessage) {
	m.ConsensusDkgDealResponseHandler.HandleConsensusDkgDealResponse(msg.Message.(*p2p_message.MessageConsensusDkgDealResponse), msg.SourceID)
}

func (m *MessageRouter) RouteConsensusDkgSigSets(msg *OGMessage) {
	m.ConsensusDkgSigSetsHandler.HandleConsensusDkgSigSets(msg.Message.(*p2p_message.MessageConsensusDkgSigSets), msg.SourceID)
}

func (m *MessageRouter) RouteConsensusDkgGenesisPublicKey(msg *OGMessage) {
	m.ConsensusDkgGenesisPublicKeyHandler.HandleConsensusDkgGenesisPublicKey(msg.Message.(*p2p_message.MessageConsensusDkgGenesisPublicKey), msg.SourceID)
}

func (m *MessageRouter) RouteTermChangeRequest(msg *OGMessage) {
	m.TermChangeRequestHandler.HandleTermChangeRequest(msg.Message.(*p2p_message.MessageTermChangeRequest), msg.SourceID)
}

func (m *MessageRouter) RouteTermChangeResponse(msg *OGMessage) {
	m.TermChangeResponseHandler.HandleTermChangeResponse(msg.Message.(*p2p_message.MessageTermChangeResponse), msg.SourceID)
}

// BroadcastMessage send Message to all peers
func (m *MessageRouter) BroadcastMessage(messageType message.BinaryMessageType, message p2p_message.Message) {
	m.Hub.BroadcastMessage(messageType, message)
}

// MulticastMessage send Message to a randomly chosen peer
func (m *MessageRouter) MulticastMessage(messageType message.BinaryMessageType, message p2p_message.Message) {
	m.Hub.MulticastMessage(messageType, message)
}

func (m *MessageRouter) MulticastToSource(messageType message.BinaryMessageType, message p2p_message.Message, sourceMsgHash *common.Hash) {
	m.Hub.MulticastToSource(messageType, message, sourceMsgHash)
}

func (m *MessageRouter) BroadcastMessageWithLink(messageType message.BinaryMessageType, message p2p_message.Message) {
	m.Hub.BroadcastMessageWithLink(messageType, message)
}

func (m *MessageRouter) SendToPeer(peerId string, messageType message.BinaryMessageType, msg p2p_message.Message) {
	err := m.Hub.SendToPeer(peerId, messageType, msg)
	if err != nil {
		message.msgLog.WithError(err).Warn("send failed")
	}
}
