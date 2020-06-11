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
package archive

//
//func (a *AnnSensus) HandleConsensusDkgGenesisPublicKey(request *p2p_message.MessageConsensusDkgGenesisPublicKey, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("annsensus disabled")
//		return
//	}
//	log := a.dkg.Log()
//	if request == nil {
//		log.Warn("got nil MessageConsensusDkgGenesisPublicKey")
//		return
//	}
//	if atomic.LoadUint32(&a.genesisBftIsRunning) == 0 {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("i am not participant in genesis bft")
//		return
//	}
//	log.WithField("dkg data", request).WithField("from peer ", peerId).Debug("got genesis pub key")
//	pk := ogcrypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
//	s := ogcrypto.NewSigner(pk.Type)
//	ok := s.Verify(pk, ogcrypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
//	if !ok {
//		log.Warn("verify signature failed")
//		return
//	}
//	a.genesisPkChan <- request
//}
//
////HandleConsensusDkgDeal
//func (a *AnnSensus) HandleConsensusDkgDeal(request *p2p_message.MessageConsensusDkgDeal, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("annsensus disabled")
//		return
//	}
//	log := a.dkg.Log()
//	if request == nil {
//		log.Warn("got nil MessageConsensusDkgDeal")
//		return
//	}
//	log.WithField("dkg data", request).WithField("from peer ", peerId).Debug("got dkg")
//	pk := ogcrypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
//	s := ogcrypto.NewSigner(pk.Type)
//	ok := s.Verify(pk, ogcrypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
//	if !ok {
//		log.Warn("verify signature failed")
//		return
//	}
//	a.dkg.HandleDkgDeal(request)
//
//}
//
////HandleConsensusDkgDealResponse
//func (a *AnnSensus) HandleConsensusDkgDealResponse(request *p2p_message.MessageConsensusDkgDealResponse, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("annsensus disabled")
//		return
//	}
//	log := a.dkg.Log()
//	if request == nil {
//		log.Warn("got nil MessageConsensusDkgDealResponse")
//		return
//	}
//	log.WithField("dkg response data", request).WithField("from peer ", peerId).Debug("got dkg response")
//	pk := ogcrypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
//	s := ogcrypto.NewSigner(pk.Type)
//	ok := s.Verify(pk, ogcrypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
//	if !ok {
//		log.Warn("verify signature failed")
//		return
//	}
//	log.Debug("response ok")
//
//	a.dkg.HandleDkgDealRespone(request)
//}
//
////HandleConsensusDkgSigSets
//func (a *AnnSensus) HandleConsensusDkgSigSets(request *p2p_message.MessageConsensusDkgSigSets, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("annsensus disabled")
//		return
//	}
//	log := a.dkg.Log()
//	if request == nil {
//		log.Warn("got nil MessageConsensusDkgSigSets")
//		return
//	}
//	log.WithField("data", request).WithField("from peer ", peerId).Debug("got dkg bls sigsets")
//	pk := ogcrypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
//	s := ogcrypto.NewSigner(pk.Type)
//	ok := s.Verify(pk, ogcrypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
//	if !ok {
//		log.WithField("pkbls ", hex.EncodeToString(request.PkBls)).WithField("pk ", hex.EncodeToString(request.PublicKey)).WithField(
//			"sig ", hex.EncodeToString(request.Signature)).Warn("verify signature failed")
//		return
//	}
//	log.Debug("response ok")
//
//	a.dkg.HandleSigSet(request)
//}
//
////HandleConsensusProposal
//func (a *AnnSensus) HandleConsensusProposal(request *bft.MessageProposal, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("annsensus disabled")
//		return
//	}
//	log := a.dkg.Log()
//	if request == nil || request.Value == nil {
//		log.Warn("got nil MessageConsensusDkgSigSets")
//		return
//	}
//	if !a.bft.Started() {
//		log.Debug("bft not started yet")
//		return
//	}
//	switch msg := request.Value.(type) {
//	case *bft.SequencerProposal:
//	default:
//		log.WithField("request ", msg).Warn("unsupported proposal type")
//		return
//	}
//	seq := request.Value.(*bft.SequencerProposal).Sequencer
//	log.WithField("data", request).WithField("from peer ", peerId).Debug("got bft proposal data")
//	pk := ogcrypto.PublicKeyFromBytes(a.cryptoType, seq.PublicKey)
//	s := ogcrypto.NewSigner(pk.Type)
//	ok := s.Verify(pk, ogcrypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
//	if !ok {
//		log.WithField("pub ", seq.PublicKey[0:5]).WithField("sig ", hex.EncodeToString(request.Signature)).WithField("request ", request).Warn("verify MessageProposal  signature failed")
//		return
//	}
//	seq.Proposing = true
//
//	if !a.bft.VerifyProposal(request, pk) {
//		log.WithField("seq ", request).Warn("verify raw seq fail")
//		return
//	}
//	//cache them and sent to buffer to verify
//	a.bft.CacheProposal(seq.GetHash(), request)
//	a.HandleNewTxi(&seq, peerId)
//	log.Debug("response ok")
//	//m := BftMessage{
//	//	Type:    p2p_message.MessageTypeProposal,
//	//	Payload: request,
//	//}
//	//a.pbft.BFTPartner.GetIncomingMessageChannel() <- m
//
//}
//
////HandleConsensusPreVote
//func (a *AnnSensus) HandleConsensusPreVote(request *bft.MessagePreVote, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("annsensus disabled")
//		return
//	}
//	log := a.dkg.Log()
//	if request == nil {
//		log.Warn("got nil MessageConsensusDkgSigSets")
//		return
//	}
//	if !a.bft.Started() {
//		log.Debug("bft not started yet")
//		return
//	}
//	log.WithField("data", request).WithField("from peer ", peerId).Debug("got bft PreVote data")
//	pk := ogcrypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
//	s := ogcrypto.NewSigner(pk.Type)
//	ok := s.Verify(pk, ogcrypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
//	if !ok {
//		log.WithField("request ", request).Warn("verify signature failed")
//		return
//	}
//	if !a.bft.VerifyIsPartNer(pk, int(request.SourceId)) {
//		log.WithField("pks ", a.bft.BFTPartner.PeersInfo).WithField("sourc id ", request.SourceId).WithField("pk ", pk).WithField("request ", request).Warn("verify partner failed")
//		return
//	}
//
//	m := bft.BftMessage{
//		Type:    p2p_message.MessageTypePreVote,
//		Payload: request,
//	}
//	a.bft.BFTPartner.GetIncomingMessageChannel() <- m
//
//}
//
////HandleConsensusPreCommit
//func (a *AnnSensus) HandleConsensusPreCommit(request *bft.MessagePreCommit, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("annsensus disabled")
//		return
//	}
//	log := a.dkg.Log()
//	if request == nil {
//		log.Warn("got nil MessageConsensusDkgSigSets")
//		return
//	}
//
//	if !a.bft.Started() {
//		log.Debug("bft not started yet")
//		return
//	}
//	log.WithField("data", request).WithField("from peer ", peerId).Debug("got bft PreCommit data")
//	pk := ogcrypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
//	s := ogcrypto.NewSigner(pk.Type)
//	ok := s.Verify(pk, ogcrypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
//	if !ok {
//		log.WithField("request ", request).Warn("verify signature failed")
//		return
//	}
//
//	if !a.bft.VerifyIsPartNer(pk, int(request.SourceId)) {
//		log.WithField("request ", request).Warn("verify signature failed")
//		return
//	}
//	a.bft.HandlePreCommit(request)
//
//}
//
//func (a *AnnSensus) HandleTermChangeRequest(request *p2p_message.MessageTermChangeRequest, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("reqest ", request).Warn("annsensus disabled")
//		return
//	}
//	log := log.WithField("me", a.dkg.GetId())
//	if request == nil {
//		log.Warn("got nil MessageConsensusDkgSigSets")
//		return
//	}
//	if !a.term.Started() {
//		log.Debug("term change  not started yet")
//		return
//	}
//	s := ogcrypto.NewSigner(a.cryptoType)
//
//	//send  genesis term change
//	tc := a.term.GetGenesisTermChange()
//	tc.GetBase().PublicKey = a.MyAccount.PublicKey.KeyBytes
//	tc.GetBase().Signature = s.Sign(a.MyAccount.PrivateKey, tc.SignatureTargets()).KeyBytes
//	tc.GetBase().Hash = tc.CalcTxHash()
//	msg := &p2p_message.MessageTermChangeResponse{
//		TermChange: tc,
//	}
//	a.Hub.SendToPeer(peerId, message.MessageTypeTermChangeResponse, msg)
//
//	log.WithField("data", msg).WithField("to  peer ", peerId).Debug("send term change")
//}
//
//func (a *AnnSensus) HandleTermChangeResponse(response *p2p_message.MessageTermChangeResponse, peerId string) {
//	if a.disable {
//		log.WithField("from ", peerId).WithField("response ", response).Warn("annsensus disabled")
//		return
//	}
//	if response == nil || response.TermChange == nil {
//		log.Warn("got nil MessageConsensusDkgSigSets")
//		return
//	}
//	if a.term.Started() {
//		log.Debug("term change  already stated")
//		return
//	}
//	tc := response.TermChange
//	log.WithField("data", response).WithField("from  peer ", peerId).Debug("got term chan")
//	s := ogcrypto.NewSigner(a.cryptoType)
//
//	pk := ogcrypto.PublicKeyFromBytes(a.cryptoType, tc.PublicKey)
//
//	ok := s.Verify(pk, ogcrypto.SignatureFromBytes(a.cryptoType, tc.Signature), tc.SignatureTargets())
//	if !ok {
//		log.WithField("request ", response).Warn("verify signature failed")
//		return
//	}
//	if !a.VerifyRequestedTermChange(tc) {
//		log.WithField("request ", response).Warn("verify term change  failed")
//		return
//	}
//	a.termChangeChan <- tc
//}
