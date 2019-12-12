//// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
////
//// Licensed under the Apache License, Version 2.0 (the "License");
//// you may not use this file except in compliance with the License.
//// You may obtain a copy of the License at
////
////     http://www.apache.org/licenses/LICENSE-2.0
////
//// Unless required by applicable law or agreed to in writing, software
//// distributed under the License is distributed on an "AS IS" BASIS,
//// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//// See the License for the specific language governing permissions and
//// limitations under the License.
package og

//
//import (
//	"github.com/annchain/OG/common"
//	"github.com/annchain/OG/consensus/bft"
//	"github.com/annchain/OG/og/types"
//	"github.com/annchain/OG/og/types/archive"
//)
//
//// MessageRouter is a bridge between hub and components
//type MessageRouter struct {
//	Hub                                 *Hub
//	PingHandler                         PingHandler
//	PongHandler                         PongHandler
//	FetchByHashRequestHandler           FetchByHashHandlerRequest
//	FetchByHashResponseHandler          FetchByHashResponseHandler
//	NewTxHandler                        NewTxHandler
//	NewTxsHandler                       NewTxsHandler
//	NewSequencerHandler                 NewSequencerHandler
//	GetMsgHandler                       GetMsgHandler
//	ControlMsgHandler                   ControlMsgHandler
//	SequencerHeaderHandler              SequencerHeaderHandler
//	BodiesRequestHandler                BodiesRequestHandler
//	BodiesResponseHandler               BodiesResponseHandler
//	TxsRequestHandler                   TxsRequestHandler
//	TxsResponseHandler                  TxsResponseHandler
//	HeaderRequestHandler                HeaderRequestHandler
//	HeaderResponseHandler               HeaderResponseHandler
//	CampaignHandler                     CampaignHandler
//	TermChangeHandler                   TermChangeHandler
//	ArchiveHandler                      ArchiveHandler
//	ActionTxHandler                     ActionTxHandler
//	ConsensusDkgDealHandler             ConsensusDkgDealHandler
//	ConsensusDkgDealResponseHandler     ConsensusDkgDealResponseHandler
//	ConsensusDkgSigSetsHandler          ConsensusDkgSigSetsHandler
//	ConsensusDkgGenesisPublicKeyHandler ConsensusDkgGenesisPublicKeyHandler
//	ConsensusHandler                    ConsensusHandler
//	TermChangeRequestHandler            TermChangeRequestHandler
//	TermChangeResponseHandler           TermChangeResponseHandler
//}
//
//type ManagerConfig struct {
//	AcquireTxQueueSize uint // length of the channel for tx acquiring
//	BatchAcquireSize   uint // length of the buffer for batch tx acquire for a single node
//}
//
//type PingHandler interface {
//	HandlePing(peerId string)
//}
//type PongHandler interface {
//	HandlePong()
//}
//
//type FetchByHashHandlerRequest interface {
//	HandleFetchByHashRequest(req *p2p_message.MessageSyncRequest, sourceID string)
//}
//
//type FetchByHashResponseHandler interface {
//	HandleFetchByHashResponse(resp *p2p_message.MessageSyncResponse, sourceID string)
//}
//
//type NewTxHandler interface {
//	HandleNewTx(msg *p2p_message.MessageNewTx, peerId string)
//}
//
//type GetMsgHandler interface {
//	HandleGetMsg(msg *p2p_message.MessageGetMsg, peerId string)
//}
//
//type ControlMsgHandler interface {
//	HandleControlMsg(msg *archive.MessageControl, peerId string)
//}
//
//type NewTxsHandler interface {
//	HandleNewTxs(newTxs *archive.MessageNewTxs, peerId string)
//}
//
//type NewSequencerHandler interface {
//	HandleNewSequencer(msg *archive.MessageNewSequencer, peerId string)
//}
//
//type SequencerHeaderHandler interface {
//	HandleSequencerHeader(msgHeader *archive.MessageSequencerHeader, peerId string)
//}
//
//type BodiesRequestHandler interface {
//	HandleBodiesRequest(msgReq *archive.MessageBodiesRequest, peerID string)
//}
//
//type BodiesResponseHandler interface {
//	HandleBodiesResponse(request *archive.MessageBodiesResponse, peerId string)
//}
//
//type TxsRequestHandler interface {
//	HandleTxsRequest(msgReq *archive.MessageTxsRequest, peerID string)
//}
//
//type TxsResponseHandler interface {
//	HandleTxsResponse(request *archive.MessageTxsResponse)
//}
//
//type HeaderRequestHandler interface {
//	HandleHeaderRequest(request *archive.MessageHeaderRequest, peerID string)
//}
//
//type HeaderResponseHandler interface {
//	HandleHeaderResponse(headerMsg *archive.MessageHeaderResponse, peerID string)
//}
//
//type CampaignHandler interface {
//	HandleCampaign(request *types.MessageCampaign, peerId string)
//}
//
//type TermChangeHandler interface {
//	HandleTermChange(request *types.MessageTermChange, peerId string)
//}
//
//type ArchiveHandler interface {
//	HandleArchive(request *types.MessageNewArchive, peerId string)
//}
//
//type ActionTxHandler interface {
//	HandleActionTx(request *archive.MessageNewActionTx, perid string)
//}
//
//type ConsensusDkgDealHandler interface {
//	HandleConsensusDkgDeal(request *types.MessageConsensusDkgDeal, peerId string)
//}
//
//type ConsensusDkgDealResponseHandler interface {
//	HandleConsensusDkgDealResponse(request *types.MessageConsensusDkgDealResponse, peerId string)
//}
//
//type ConsensusDkgSigSetsHandler interface {
//	HandleConsensusDkgSigSets(request *types.MessageConsensusDkgSigSets, peerId string)
//}
//
//type ConsensusDkgGenesisPublicKeyHandler interface {
//	HandleConsensusDkgGenesisPublicKey(request *types.MessageConsensusDkgGenesisPublicKey, peerId string)
//}
//type ConsensusHandler interface {
//	HandleConsensus(request bft.BftMessage, peerId string)
//}
//type TermChangeRequestHandler interface {
//	HandleTermChangeRequest(request *types.MessageTermChangeRequest, peerId string)
//}
//type TermChangeResponseHandler interface {
//	HandleTermChangeResponse(request *types.MessageTermChangeResponse, peerId string)
//}
//
//func (m *MessageRouter) Start() {
//	m.Hub.BroadcastMessage(message.MessageTypePing, &archive.MessagePing{Data: []byte{}})
//}
//
//func (m *MessageRouter) Stop() {
//
//}
//
//func (m *MessageRouter) Name() string {
//	return "MessageRouter"
//}
//
//func (m *MessageRouter) RoutePing(msg *types) {
//	m.PingHandler.HandlePing(msg.SourceID)
//}
//
//func (m *MessageRouter) RoutePong(*types) {
//	m.PongHandler.HandlePong()
//}
//func (m *MessageRouter) RouteFetchByHashRequest(msg *types) {
//	m.FetchByHashRequestHandler.HandleFetchByHashRequest(msg.Message.(*archive.MessageSyncRequest), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteFetchByHashResponse(msg *types) {
//	m.FetchByHashResponseHandler.HandleFetchByHashResponse(msg.Message.(*archive.MessageSyncResponse), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteNewTx(msg *types) {
//	m.NewTxHandler.HandleNewTx(msg.Message.(*archive.MessageNewTx), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteNewTxs(msg *types) {
//	//maybe received more transactions
//	m.NewTxsHandler.HandleNewTxs(msg.Message.(*archive.MessageNewTxs), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteNewSequencer(msg *types) {
//	m.NewSequencerHandler.HandleNewSequencer(msg.Message.(*archive.MessageNewSequencer), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteGetMsg(msg *types) {
//	m.GetMsgHandler.HandleGetMsg(msg.Message.(*archive.MessageGetMsg), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteControlMsg(msg *types) {
//	m.ControlMsgHandler.HandleControlMsg(msg.Message.(*archive.MessageControl), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteSequencerHeader(msg *types) {
//	m.SequencerHeaderHandler.HandleSequencerHeader(msg.Message.(*archive.MessageSequencerHeader), msg.SourceID)
//}
//func (m *MessageRouter) RouteBodiesRequest(msg *types) {
//
//	m.BodiesRequestHandler.HandleBodiesRequest(msg.Message.(*archive.MessageBodiesRequest), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteBodiesResponse(msg *types) {
//	// A batch of block bodies arrived to one of our previous requests
//	m.BodiesResponseHandler.HandleBodiesResponse(msg.Message.(*archive.MessageBodiesResponse), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteTxsRequest(msg *types) {
//	// Decode the retrieval Message
//	m.TxsRequestHandler.HandleTxsRequest(msg.Message.(*archive.MessageTxsRequest), msg.SourceID)
//
//}
//func (m *MessageRouter) RouteTxsResponse(msg *types) {
//	// A batch of block bodies arrived to one of our previous requests
//	m.TxsResponseHandler.HandleTxsResponse(msg.Message.(*archive.MessageTxsResponse))
//
//}
//func (m *MessageRouter) RouteHeaderRequest(msg *types) {
//	// Decode the complex header query
//	m.HeaderRequestHandler.HandleHeaderRequest(msg.Message.(*archive.MessageHeaderRequest), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteHeaderResponse(msg *types) {
//	// A batch of headers arrived to one of our previous requests
//	m.HeaderResponseHandler.HandleHeaderResponse(msg.Message.(*archive.MessageHeaderResponse), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteCampaign(msg *types) {
//	// A batch of headers arrived to one of our previous requests
//	m.CampaignHandler.HandleCampaign(msg.Message.(*types.MessageCampaign), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteTermChange(msg *types) {
//	m.TermChangeHandler.HandleTermChange(msg.Message.(*types.MessageTermChange), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteArchive(msg *types) {
//	m.ArchiveHandler.HandleArchive(msg.Message.(*types.MessageNewArchive), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteActionTx(msg *types) {
//	m.ActionTxHandler.HandleActionTx(msg.Message.(*archive.MessageNewActionTx), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteConsensusDkgDeal(msg *types) {
//	m.ConsensusDkgDealHandler.HandleConsensusDkgDeal(msg.Message.(*types.MessageConsensusDkgDeal), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteConsensusDkgDealResponse(msg *types) {
//	m.ConsensusDkgDealResponseHandler.HandleConsensusDkgDealResponse(msg.Message.(*types.MessageConsensusDkgDealResponse), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteConsensusDkgSigSets(msg *types) {
//	m.ConsensusDkgSigSetsHandler.HandleConsensusDkgSigSets(msg.Message.(*types.MessageConsensusDkgSigSets), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteConsensusDkgGenesisPublicKey(msg *types) {
//	m.ConsensusDkgGenesisPublicKeyHandler.HandleConsensusDkgGenesisPublicKey(msg.Message.(*types.MessageConsensusDkgGenesisPublicKey), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteTermChangeRequest(msg *types) {
//	m.TermChangeRequestHandler.HandleTermChangeRequest(msg.Message.(*types.MessageTermChangeRequest), msg.SourceID)
//}
//
//func (m *MessageRouter) RouteTermChangeResponse(msg *types) {
//	m.TermChangeResponseHandler.HandleTermChangeResponse(msg.Message.(*types.MessageTermChangeResponse), msg.SourceID)
//}
//
//// BroadcastMessage send Message to all peers
//func (m *MessageRouter) BroadcastMessage(messageType message.BinaryMessageType, message types.Message) {
//	m.Hub.BroadcastMessage(messageType, message)
//}
//
//// MulticastMessage send Message to a randomly chosen peer
//func (m *MessageRouter) MulticastMessage(messageType message.BinaryMessageType, message types.Message) {
//	m.Hub.MulticastMessage(messageType, message)
//}
//
//func (m *MessageRouter) MulticastToSource(messageType message.BinaryMessageType, message types.Message, sourceMsgHash *common.Hash) {
//	m.Hub.MulticastToSource(messageType, message, sourceMsgHash)
//}
//
//func (m *MessageRouter) BroadcastMessageWithLink(messageType message.BinaryMessageType, message types.Message) {
//	m.Hub.BroadcastMessageWithLink(messageType, message)
//}
//
//func (m *MessageRouter) SendToPeer(peerId string, messageType message.BinaryMessageType, msg types.Message) {
//	err := m.Hub.SendToPeer(peerId, messageType, msg)
//	if err != nil {
//		message.msgLog.WithError(err).Warn("send failed")
//	}
//}
