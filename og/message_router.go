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
	"github.com/annchain/OG/types"
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

type ControlMsgHandler interface {
	HandleControlMsg(msg *types.MessageControl, peerId string)
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

type CampaignHandler interface {
	HandleCampaign(request *types.MessageCampaign, peerId string)
}

type TermChangeHandler interface {
	HandleTermChange(request *types.MessageTermChange, peerId string)
}

type ArchiveHandler interface {
	HandleArchive(request *types.MessageNewArchive, peerId string)
}

type ConsensusDkgDealHandler interface {
	HandleConsensusDkgDeal(request *types.MessageConsensusDkgDeal, peerId string)
}

type ConsensusDkgDealResponseHandler interface {
	HandleConsensusDkgDealResponse(request *types.MessageConsensusDkgDealResponse, peerId string)
}

type ConsensusDkgSigSetsHandler interface {
	HandleConsensusDkgSigSets(request *types.MessageConsensusDkgSigSets, peerId string)
}

type ConsensusDkgGenesisPublicKeyHandler interface {
	HandleConsensusDkgGenesisPublicKey(request *types.MessageConsensusDkgGenesisPublicKey, peerId string)
}
type ConsensusPreVoteHandler interface {
	HandleConsensusPreVote(request *types.MessagePreVote, peerId string)
}

type ConsensusPreCommitHandler interface {
	HandleConsensusPreCommit(request *types.MessagePreCommit, peerId string)
}

type ConsensusProposalHandler interface {
	HandleConsensusProposal(request *types.MessageProposal, peerId string)
}

type TermChangeRequestHandler interface {
	HandleTermChangeRequest(request *types.MessageTermChangeRequest, peerId string)
}
type TermChangeResponseHandler interface {
	HandleTermChangeResponse(request *types.MessageTermChangeResponse, peerId string)
}

func (m *MessageRouter) Start() {
	m.Hub.BroadcastMessage(MessageTypePing, &types.MessagePing{Data: []byte{}})
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
	m.FetchByHashRequestHandler.HandleFetchByHashRequest(msg.message.(*types.MessageSyncRequest), msg.sourceID)
}

func (m *MessageRouter) RouteFetchByHashResponse(msg *p2PMessage) {
	m.FetchByHashResponseHandler.HandleFetchByHashResponse(msg.message.(*types.MessageSyncResponse), msg.sourceID)
}

func (m *MessageRouter) RouteNewTx(msg *p2PMessage) {
	m.NewTxHandler.HandleNewTx(msg.message.(*types.MessageNewTx), msg.sourceID)
}

func (m *MessageRouter) RouteNewTxs(msg *p2PMessage) {
	//maybe received more transactions
	m.NewTxsHandler.HandleNewTxs(msg.message.(*types.MessageNewTxs), msg.sourceID)
}

func (m *MessageRouter) RouteNewSequencer(msg *p2PMessage) {
	m.NewSequencerHandler.HandleNewSequencer(msg.message.(*types.MessageNewSequencer), msg.sourceID)
}

func (m *MessageRouter) RouteGetMsg(msg *p2PMessage) {
	m.GetMsgHandler.HandleGetMsg(msg.message.(*types.MessageGetMsg), msg.sourceID)
}

func (m *MessageRouter) RouteControlMsg(msg *p2PMessage) {
	m.ControlMsgHandler.HandleControlMsg(msg.message.(*types.MessageControl), msg.sourceID)
}

func (m *MessageRouter) RouteSequencerHeader(msg *p2PMessage) {
	m.SequencerHeaderHandler.HandleSequencerHeader(msg.message.(*types.MessageSequencerHeader), msg.sourceID)
}
func (m *MessageRouter) RouteBodiesRequest(msg *p2PMessage) {

	m.BodiesRequestHandler.HandleBodiesRequest(msg.message.(*types.MessageBodiesRequest), msg.sourceID)
}

func (m *MessageRouter) RouteBodiesResponse(msg *p2PMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.BodiesResponseHandler.HandleBodiesResponse(msg.message.(*types.MessageBodiesResponse), msg.sourceID)
}

func (m *MessageRouter) RouteTxsRequest(msg *p2PMessage) {
	// Decode the retrieval message
	m.TxsRequestHandler.HandleTxsRequest(msg.message.(*types.MessageTxsRequest), msg.sourceID)

}
func (m *MessageRouter) RouteTxsResponse(msg *p2PMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.TxsResponseHandler.HandleTxsResponse(msg.message.(*types.MessageTxsResponse))

}
func (m *MessageRouter) RouteHeaderRequest(msg *p2PMessage) {
	// Decode the complex header query
	m.HeaderRequestHandler.HandleHeaderRequest(msg.message.(*types.MessageHeaderRequest), msg.sourceID)
}

func (m *MessageRouter) RouteHeaderResponse(msg *p2PMessage) {
	// A batch of headers arrived to one of our previous requests
	m.HeaderResponseHandler.HandleHeaderResponse(msg.message.(*types.MessageHeaderResponse), msg.sourceID)
}

func (m *MessageRouter) RouteCampaign(msg *p2PMessage) {
	// A batch of headers arrived to one of our previous requests
	m.CampaignHandler.HandleCampaign(msg.message.(*types.MessageCampaign), msg.sourceID)
}

func (m *MessageRouter) RouteTermChange(msg *p2PMessage) {
	m.TermChangeHandler.HandleTermChange(msg.message.(*types.MessageTermChange), msg.sourceID)
}

func (m*MessageRouter)RouteArchive(msg *p2PMessage) {
	m.ArchiveHandler.HandleArchive(msg.message.(*types.MessageNewArchive), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusDkgDeal(msg *p2PMessage) {
	m.ConsensusDkgDealHandler.HandleConsensusDkgDeal(msg.message.(*types.MessageConsensusDkgDeal), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusDkgDealResponse(msg *p2PMessage) {
	m.ConsensusDkgDealResponseHandler.HandleConsensusDkgDealResponse(msg.message.(*types.MessageConsensusDkgDealResponse), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusDkgSigSets(msg *p2PMessage) {
	m.ConsensusDkgSigSetsHandler.HandleConsensusDkgSigSets(msg.message.(*types.MessageConsensusDkgSigSets), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusDkgGenesisPublicKey(msg *p2PMessage) {
	m.ConsensusDkgGenesisPublicKeyHandler.HandleConsensusDkgGenesisPublicKey(msg.message.(*types.MessageConsensusDkgGenesisPublicKey), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusProposal(msg *p2PMessage) {
	m.ConsensusProposalHandler.HandleConsensusProposal(msg.message.(*types.MessageProposal), msg.sourceID)
}

func (m *MessageRouter) RouteConsensusPreVote(msg *p2PMessage) {
	m.ConsensusPreVoteHandler.HandleConsensusPreVote(msg.message.(*types.MessagePreVote), msg.sourceID)
}
func (m *MessageRouter) RouteConsensusPreCommit(msg *p2PMessage) {
	m.ConsensusPreCommitHandler.HandleConsensusPreCommit(msg.message.(*types.MessagePreCommit), msg.sourceID)
}

func (m *MessageRouter) RouteTermChangeRequest(msg *p2PMessage) {
	m.TermChangeRequestHandler.HandleTermChangeRequest(msg.message.(*types.MessageTermChangeRequest), msg.sourceID)
}

func (m *MessageRouter) RouteTermChangeResponse(msg *p2PMessage) {
	m.TermChangeResponseHandler.HandleTermChangeResponse(msg.message.(*types.MessageTermChangeResponse), msg.sourceID)
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
