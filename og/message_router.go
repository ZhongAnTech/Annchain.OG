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
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
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
	HandleControlMsg(msg *archive.MessageControl, peerId string)
}

type NewTxsHandler interface {
	HandleNewTxs(newTxs *archive.MessageNewTxs, peerId string)
}

type NewSequencerHandler interface {
	HandleNewSequencer(msg *archive.MessageNewSequencer, peerId string)
}

type SequencerHeaderHandler interface {
	HandleSequencerHeader(msgHeader *archive.MessageSequencerHeader, peerId string)
}

type BodiesRequestHandler interface {
	HandleBodiesRequest(msgReq *archive.MessageBodiesRequest, peerID string)
}

type BodiesResponseHandler interface {
	HandleBodiesResponse(request *archive.MessageBodiesResponse, peerId string)
}

type TxsRequestHandler interface {
	HandleTxsRequest(msgReq *archive.MessageTxsRequest, peerID string)
}

type TxsResponseHandler interface {
	HandleTxsResponse(request *archive.MessageTxsResponse)
}

type HeaderRequestHandler interface {
	HandleHeaderRequest(request *archive.MessageHeaderRequest, peerID string)
}

type HeaderResponseHandler interface {
	HandleHeaderResponse(headerMsg *archive.MessageHeaderResponse, peerID string)
}

type CampaignHandler interface {
	HandleCampaign(request *ogmessage.MessageCampaign, peerId string)
}

type TermChangeHandler interface {
	HandleTermChange(request *ogmessage.MessageTermChange, peerId string)
}

type ArchiveHandler interface {
	HandleArchive(request *ogmessage.MessageNewArchive, peerId string)
}

type ActionTxHandler interface {
	HandleActionTx(request *archive.MessageNewActionTx, perid string)
}

type ConsensusDkgDealHandler interface {
	HandleConsensusDkgDeal(request *ogmessage.MessageConsensusDkgDeal, peerId string)
}

type ConsensusDkgDealResponseHandler interface {
	HandleConsensusDkgDealResponse(request *ogmessage.MessageConsensusDkgDealResponse, peerId string)
}

type ConsensusDkgSigSetsHandler interface {
	HandleConsensusDkgSigSets(request *ogmessage.MessageConsensusDkgSigSets, peerId string)
}

type ConsensusDkgGenesisPublicKeyHandler interface {
	HandleConsensusDkgGenesisPublicKey(request *ogmessage.MessageConsensusDkgGenesisPublicKey, peerId string)
}
type ConsensusHandler interface {
	HandleConsensus(request bft.BftMessage, peerId string)
}
type TermChangeRequestHandler interface {
	HandleTermChangeRequest(request *ogmessage.MessageTermChangeRequest, peerId string)
}
type TermChangeResponseHandler interface {
	HandleTermChangeResponse(request *ogmessage.MessageTermChangeResponse, peerId string)
}

func (m *MessageRouter) Start() {
	m.Hub.BroadcastMessage(message.MessageTypePing, &archive.MessagePing{Data: []byte{}})
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
	m.FetchByHashRequestHandler.HandleFetchByHashRequest(msg.Message.(*archive.MessageSyncRequest), msg.SourceID)
}

func (m *MessageRouter) RouteFetchByHashResponse(msg *OGMessage) {
	m.FetchByHashResponseHandler.HandleFetchByHashResponse(msg.Message.(*archive.MessageSyncResponse), msg.SourceID)
}

func (m *MessageRouter) RouteNewTx(msg *OGMessage) {
	m.NewTxHandler.HandleNewTx(msg.Message.(*archive.MessageNewTx), msg.SourceID)
}

func (m *MessageRouter) RouteNewTxs(msg *OGMessage) {
	//maybe received more transactions
	m.NewTxsHandler.HandleNewTxs(msg.Message.(*archive.MessageNewTxs), msg.SourceID)
}

func (m *MessageRouter) RouteNewSequencer(msg *OGMessage) {
	m.NewSequencerHandler.HandleNewSequencer(msg.Message.(*archive.MessageNewSequencer), msg.SourceID)
}

func (m *MessageRouter) RouteGetMsg(msg *OGMessage) {
	m.GetMsgHandler.HandleGetMsg(msg.Message.(*archive.MessageGetMsg), msg.SourceID)
}

func (m *MessageRouter) RouteControlMsg(msg *OGMessage) {
	m.ControlMsgHandler.HandleControlMsg(msg.Message.(*archive.MessageControl), msg.SourceID)
}

func (m *MessageRouter) RouteSequencerHeader(msg *OGMessage) {
	m.SequencerHeaderHandler.HandleSequencerHeader(msg.Message.(*archive.MessageSequencerHeader), msg.SourceID)
}
func (m *MessageRouter) RouteBodiesRequest(msg *OGMessage) {

	m.BodiesRequestHandler.HandleBodiesRequest(msg.Message.(*archive.MessageBodiesRequest), msg.SourceID)
}

func (m *MessageRouter) RouteBodiesResponse(msg *OGMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.BodiesResponseHandler.HandleBodiesResponse(msg.Message.(*archive.MessageBodiesResponse), msg.SourceID)
}

func (m *MessageRouter) RouteTxsRequest(msg *OGMessage) {
	// Decode the retrieval Message
	m.TxsRequestHandler.HandleTxsRequest(msg.Message.(*archive.MessageTxsRequest), msg.SourceID)

}
func (m *MessageRouter) RouteTxsResponse(msg *OGMessage) {
	// A batch of block bodies arrived to one of our previous requests
	m.TxsResponseHandler.HandleTxsResponse(msg.Message.(*archive.MessageTxsResponse))

}
func (m *MessageRouter) RouteHeaderRequest(msg *OGMessage) {
	// Decode the complex header query
	m.HeaderRequestHandler.HandleHeaderRequest(msg.Message.(*archive.MessageHeaderRequest), msg.SourceID)
}

func (m *MessageRouter) RouteHeaderResponse(msg *OGMessage) {
	// A batch of headers arrived to one of our previous requests
	m.HeaderResponseHandler.HandleHeaderResponse(msg.Message.(*archive.MessageHeaderResponse), msg.SourceID)
}

func (m *MessageRouter) RouteCampaign(msg *OGMessage) {
	// A batch of headers arrived to one of our previous requests
	m.CampaignHandler.HandleCampaign(msg.Message.(*ogmessage.MessageCampaign), msg.SourceID)
}

func (m *MessageRouter) RouteTermChange(msg *OGMessage) {
	m.TermChangeHandler.HandleTermChange(msg.Message.(*ogmessage.MessageTermChange), msg.SourceID)
}

func (m *MessageRouter) RouteArchive(msg *OGMessage) {
	m.ArchiveHandler.HandleArchive(msg.Message.(*ogmessage.MessageNewArchive), msg.SourceID)
}

func (m *MessageRouter) RouteActionTx(msg *OGMessage) {
	m.ActionTxHandler.HandleActionTx(msg.Message.(*archive.MessageNewActionTx), msg.SourceID)
}

func (m *MessageRouter) RouteConsensusDkgDeal(msg *OGMessage) {
	m.ConsensusDkgDealHandler.HandleConsensusDkgDeal(msg.Message.(*ogmessage.MessageConsensusDkgDeal), msg.SourceID)
}

func (m *MessageRouter) RouteConsensusDkgDealResponse(msg *OGMessage) {
	m.ConsensusDkgDealResponseHandler.HandleConsensusDkgDealResponse(msg.Message.(*ogmessage.MessageConsensusDkgDealResponse), msg.SourceID)
}

func (m *MessageRouter) RouteConsensusDkgSigSets(msg *OGMessage) {
	m.ConsensusDkgSigSetsHandler.HandleConsensusDkgSigSets(msg.Message.(*ogmessage.MessageConsensusDkgSigSets), msg.SourceID)
}

func (m *MessageRouter) RouteConsensusDkgGenesisPublicKey(msg *OGMessage) {
	m.ConsensusDkgGenesisPublicKeyHandler.HandleConsensusDkgGenesisPublicKey(msg.Message.(*ogmessage.MessageConsensusDkgGenesisPublicKey), msg.SourceID)
}

func (m *MessageRouter) RouteTermChangeRequest(msg *OGMessage) {
	m.TermChangeRequestHandler.HandleTermChangeRequest(msg.Message.(*ogmessage.MessageTermChangeRequest), msg.SourceID)
}

func (m *MessageRouter) RouteTermChangeResponse(msg *OGMessage) {
	m.TermChangeResponseHandler.HandleTermChangeResponse(msg.Message.(*ogmessage.MessageTermChangeResponse), msg.SourceID)
}

// BroadcastMessage send Message to all peers
func (m *MessageRouter) BroadcastMessage(messageType message.BinaryMessageType, message ogmessage.Message) {
	m.Hub.BroadcastMessage(messageType, message)
}

// MulticastMessage send Message to a randomly chosen peer
func (m *MessageRouter) MulticastMessage(messageType message.BinaryMessageType, message ogmessage.Message) {
	m.Hub.MulticastMessage(messageType, message)
}

func (m *MessageRouter) MulticastToSource(messageType message.BinaryMessageType, message ogmessage.Message, sourceMsgHash *common.Hash) {
	m.Hub.MulticastToSource(messageType, message, sourceMsgHash)
}

func (m *MessageRouter) BroadcastMessageWithLink(messageType message.BinaryMessageType, message ogmessage.Message) {
	m.Hub.BroadcastMessageWithLink(messageType, message)
}

func (m *MessageRouter) SendToPeer(peerId string, messageType message.BinaryMessageType, msg ogmessage.Message) {
	err := m.Hub.SendToPeer(peerId, messageType, msg)
	if err != nil {
		message.msgLog.WithError(err).Warn("send failed")
	}
}
