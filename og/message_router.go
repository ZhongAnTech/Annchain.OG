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
	ConsensusProposalHandler            ConsensusProposalHandler
	ConsensusPreVoteHandler             ConsensusPreVoteHandler
	ConsensusPreCommitHandler           ConsensusPreCommitHandler
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
type ConsensusPreVoteHandler interface {
	HandleConsensusPreVote(request *p2p_message.MessagePreVote, peerId string)
}

type ConsensusPreCommitHandler interface {
	HandleConsensusPreCommit(request *p2p_message.MessagePreCommit, peerId string)
}

type ConsensusProposalHandler interface {
	HandleConsensusProposal(request *p2p_message.MessageProposal, peerId string)
}

type TermChangeRequestHandler interface {
	HandleTermChangeRequest(request *p2p_message.MessageTermChangeRequest, peerId string)
}
type TermChangeResponseHandler interface {
	HandleTermChangeResponse(request *p2p_message.MessageTermChangeResponse, peerId string)
}

func (m *MessageRouter) Start() {
	m.Hub.BroadcastMessage(p2p_message.MessageTypePing, &p2p_message.MessagePing{Data: []byte{}})
}

func (m *MessageRouter) Stop() {

}

func (m *MessageRouter) Name() string {
	return "MessageRouter"
}

func (m *MessageRouter) RoutePing(msg *p2PMessage) {
	m.PingHandler.HandlePing(msg.sourceID)
}

func (m *MessageRouter) RoutePong(*p2PMessage) {
	m.PongHandler.HandlePong()
}
func (m *MessageRouter) RouteFetchByHashRequest(msg *p2PMessage) {
	m.FetchByHashRequestHandler.HandleFetchByHashRequest(msg.message.(*p2p_message.MessageSyncRequest), msg.sourceID)
}

func (m *MessageRouter) RouteFetchByHashResponse(msg *p2PMessage) {
	m.FetchByHashResponseHandler.HandleFetchByHashResponse(msg.message.(*p2p_message.MessageSyncResponse), msg.sourceID)
}

func (m *MessageRouter) RouteNewTx(msg *p2PMessage) {
	m.NewTxHandler.HandleNewTx(msg.message.(*p2p_message.MessageNewTx), msg.sourceID)
}

func (m *MessageRouter) RouteNewTxs(msg *p2PMessage) {
	//maybe received more transactions
	m.NewTxsHandler.HandleNewTxs(msg.message.(*p2p_message.MessageNewTxs), msg.sourceID)
}

func (m *MessageRouter) RouteNewSequencer(msg *p2PMessage) {
	m.NewSequencerHandler.HandleNewSequencer(msg.message.(*p2p_message.MessageNewSequencer), msg.sourceID)
}

func (m *MessageRouter) RouteGetMsg(msg *p2PMessage) {
	m.GetMsgHandler.HandleGetMsg(msg.message.(*p2p_message.MessageGetMsg), msg.sourceID)
}

func (m *MessageRouter) RouteControlMsg(msg *p2PMessage) {
	m.ControlMsgHandler.HandleControlMsg(msg.message.(*p2p_message.MessageControl), msg.sourceID)
}

func (m *MessageRouter) RouteSequencerHeader(msg *p2PMessage) {
	m.SequencerHeaderHandler.HandleSequencerHeader(msg.message.(*p2p_message.MessageSequencerHeader), msg.sourceID)
}
func (m *MessageRouter) RouteBodiesRequest(msg *p2PMessage) {

	m.BodiesRequestHandler.HandleBodiesRequest(msg.message.(*p2p_message.MessageBodiesRequest), msg.sourceID)
}

func (m *MessageRouter) RouteBodiesResponse(msg *p2PMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.BodiesResponseHandler.HandleBodiesResponse(msg.message.(*p2p_message.MessageBodiesResponse), msg.sourceID)
}

func (m *MessageRouter) RouteTxsRequest(msg *p2PMessage) {
	// Decode the retrieval message
	m.TxsRequestHandler.HandleTxsRequest(msg.message.(*p2p_message.MessageTxsRequest), msg.sourceID)

}
func (m *MessageRouter) RouteTxsResponse(msg *p2PMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.TxsResponseHandler.HandleTxsResponse(msg.message.(*p2p_message.MessageTxsResponse))

}
func (m *MessageRouter) RouteHeaderRequest(msg *p2PMessage) {
	// Decode the complex header query
	m.HeaderRequestHandler.HandleHeaderRequest(msg.message.(*p2p_message.MessageHeaderRequest), msg.sourceID)
}

func (m *MessageRouter) RouteHeaderResponse(msg *p2PMessage) {
	// A batch of headers arrived to one of our previous requests
	m.HeaderResponseHandler.HandleHeaderResponse(msg.message.(*p2p_message.MessageHeaderResponse), msg.sourceID)
}

func (m *MessageRouter) RouteCampaign(msg *p2PMessage) {
	// A batch of headers arrived to one of our previous requests
	m.CampaignHandler.HandleCampaign(msg.message.(*p2p_message.MessageCampaign), msg.sourceID)
}

func (m *MessageRouter) RouteTermChange(msg *p2PMessage) {
	m.TermChangeHandler.HandleTermChange(msg.message.(*p2p_message.MessageTermChange), msg.sourceID)
}

func (m *MessageRouter) RouteArchive(msg *p2PMessage) {
	m.ArchiveHandler.HandleArchive(msg.message.(*p2p_message.MessageNewArchive), msg.sourceID)
}

func (m *MessageRouter) RouteActionTx(msg *p2PMessage) {
	m.ActionTxHandler.HandleActionTx(msg.message.(*p2p_message.MessageNewActionTx), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusDkgDeal(msg *p2PMessage) {
	m.ConsensusDkgDealHandler.HandleConsensusDkgDeal(msg.message.(*p2p_message.MessageConsensusDkgDeal), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusDkgDealResponse(msg *p2PMessage) {
	m.ConsensusDkgDealResponseHandler.HandleConsensusDkgDealResponse(msg.message.(*p2p_message.MessageConsensusDkgDealResponse), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusDkgSigSets(msg *p2PMessage) {
	m.ConsensusDkgSigSetsHandler.HandleConsensusDkgSigSets(msg.message.(*p2p_message.MessageConsensusDkgSigSets), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusDkgGenesisPublicKey(msg *p2PMessage) {
	m.ConsensusDkgGenesisPublicKeyHandler.HandleConsensusDkgGenesisPublicKey(msg.message.(*p2p_message.MessageConsensusDkgGenesisPublicKey), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusProposal(msg *p2PMessage) {
	m.ConsensusProposalHandler.HandleConsensusProposal(msg.message.(*p2p_message.MessageProposal), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusPreVote(msg *p2PMessage) {
	m.ConsensusPreVoteHandler.HandleConsensusPreVote(msg.message.(*p2p_message.MessagePreVote), msg.sourceID)
}
func (m *MessageRouter) RouteConsensusPreCommit(msg *p2PMessage) {
	m.ConsensusPreCommitHandler.HandleConsensusPreCommit(msg.message.(*p2p_message.MessagePreCommit), msg.sourceID)
}

func (m *MessageRouter) RouteTermChangeRequest(msg *p2PMessage) {
	m.TermChangeRequestHandler.HandleTermChangeRequest(msg.message.(*p2p_message.MessageTermChangeRequest), msg.sourceID)
}

func (m *MessageRouter) RouteTermChangeResponse(msg *p2PMessage) {
	m.TermChangeResponseHandler.HandleTermChangeResponse(msg.message.(*p2p_message.MessageTermChangeResponse), msg.sourceID)
}

// BroadcastMessage send message to all peers
func (m *MessageRouter) BroadcastMessage(messageType p2p_message.MessageType, message p2p_message.Message) {
	m.Hub.BroadcastMessage(messageType, message)
}

// MulticastMessage send message to a randomly chosen peer
func (m *MessageRouter) MulticastMessage(messageType p2p_message.MessageType, message p2p_message.Message) {
	m.Hub.MulticastMessage(messageType, message)
}

func (m *MessageRouter) MulticastToSource(messageType p2p_message.MessageType, message p2p_message.Message, sourceMsgHash *common.Hash) {
	m.Hub.MulticastToSource(messageType, message, sourceMsgHash)
}

func (m *MessageRouter) BroadcastMessageWithLink(messageType p2p_message.MessageType, message p2p_message.Message) {
	m.Hub.BroadcastMessageWithLink(messageType, message)
}

func (m *MessageRouter) SendToPeer(peerId string, messageType p2p_message.MessageType, msg p2p_message.Message) {
	err := m.Hub.SendToPeer(peerId, messageType, msg)
	if err != nil {
		msgLog.WithError(err).Warn("send failed")
	}
}
