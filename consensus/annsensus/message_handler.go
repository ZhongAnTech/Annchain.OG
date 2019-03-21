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
package annsensus

import (
	"encoding/hex"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"sync/atomic"

	"github.com/annchain/OG/types"
)

func (a *AnnSensus) HandleConsensusDkgGenesisPublicKey(request *types.MessageConsensusDkgGenesisPublicKey, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgGenesisPublicKey")
		return
	}
	if atomic.LoadUint32(&a.genesisBftIsRunning) == 0 {
		log.WithField("from ", peerId).WithField("reqest ", request).Warn("i am not participant in genesis bft")
		return
	}
	log.WithField("dkg data", request).WithField("from peer ", peerId).Debug("got genesis pub key")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	a.genesisPkChan <- request
}

//HandleConsensusDkgDeal
func (a *AnnSensus) HandleConsensusDkgDeal(request *types.MessageConsensusDkgDeal, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgDeal")
		return
	}
	log.WithField("dkg data", request).WithField("from peer ", peerId).Debug("got dkg")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	a.dkg.gossipReqCh <- request

}

//HandleConsensusDkgDealResponse
func (a *AnnSensus) HandleConsensusDkgDealResponse(request *types.MessageConsensusDkgDealResponse, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgDealResponse")
		return
	}
	log.WithField("dkg response data", request).WithField("from peer ", peerId).Debug("got dkg response")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
	if !ok {
		log.Warn("verify signature failed")
		return
	}
	log.Debug("response ok")

	a.dkg.gossipRespCh <- request
}

//HandleConsensusDkgSigSets
func (a *AnnSensus) HandleConsensusDkgSigSets(request *types.MessageConsensusDkgSigSets, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgSigSets")
		return
	}
	log.WithField("data", request).WithField("from peer ", peerId).Debug("got dkg bls sigsets")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
	if !ok {
		log.WithField("pkbls ", hex.EncodeToString(request.PkBls)).WithField("pk ", hex.EncodeToString(request.PublicKey)).WithField(
			"sig ", hex.EncodeToString(request.Signature)).Warn("verify signature failed")
		return
	}
	log.Debug("response ok")

	a.dkg.gossipSigSetspCh <- request
}

//HandleConsensusProposal
func (a *AnnSensus) HandleConsensusProposal(request *types.MessageProposal, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil || request.Value == nil {
		log.Warn("got nil MessageConsensusDkgSigSets")
		return
	}
	if !a.bft.started {
		log.Debug("bft not started yet")
		return
	}
	switch msg := request.Value.(type) {
	case *types.SequencerProposal:
	default:
		log.WithField("request ", msg).Warn("unsupported proposal type")
		return
	}
	seq := request.Value.(*types.SequencerProposal).Sequencer
	log.WithField("data", request).WithField("from peer ", peerId).Debug("got bft proposal data")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, seq.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
	if !ok {
		log.WithField("pub ", seq.PublicKey[0:5]).WithField("sig ", hex.EncodeToString(request.Signature)).WithField("request ", request).Warn("verify MessageProposal  signature failed")
		return
	}
	seq.Proposing = true

	if !a.bft.verifyProposal(request, pk) {
		log.WithField("seq ", seq).Warn("verify raw seq fail")
		return
	}
	//cache them and sent to buffer to verify
	a.bft.proposalCache[seq.GetTxHash()] = request
	a.HandleNewTxi(&seq)
	log.Debug("response ok")
	//m := Message{
	//	Type:    og.MessageTypeProposal,
	//	Payload: request,
	//}
	//a.pbft.BFTPartner.GetIncomingMessageChannel() <- m

}

//HandleConsensusPreVote
func (a *AnnSensus) HandleConsensusPreVote(request *types.MessagePreVote, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgSigSets")
		return
	}
	if !a.bft.started {
		log.Debug("bft not started yet")
		return
	}
	log.WithField("data", request).WithField("from peer ", peerId).Debug("got bft PreVote data")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
	if !ok {
		log.WithField("request ", request).Warn("verify signature failed")
		return
	}
	if !a.bft.verifyIsPartNer(pk, int(request.SourceId)) {
		log.WithField("request ", request).Warn("verify partner failed")
		return
	}

	m := Message{
		Type:    og.MessageTypePreVote,
		Payload: request,
	}
	a.bft.BFTPartner.GetIncomingMessageChannel() <- m

}

//HandleConsensusPreCommit
func (a *AnnSensus) HandleConsensusPreCommit(request *types.MessagePreCommit, peerId string) {
	log := log.WithField("me", a.id)
	if request == nil {
		log.Warn("got nil MessageConsensusDkgSigSets")
		return
	}

	if !a.bft.started {
		log.Debug("bft not started yet")
		return
	}
	log.WithField("data", request).WithField("from peer ", peerId).Debug("got bft PreCommit data")
	pk := crypto.PublicKeyFromBytes(a.cryptoType, request.PublicKey)
	s := crypto.NewSigner(pk.Type)
	ok := s.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, request.Signature), request.SignatureTargets())
	if !ok {
		log.WithField("request ", request).Warn("verify signature failed")
		return
	}

	if !a.bft.verifyIsPartNer(pk, int(request.SourceId)) {
		log.WithField("request ", request).Warn("verify signature failed")
		return
	}
	m := Message{
		Type:    og.MessageTypePreCommit,
		Payload: request,
	}
	a.bft.BFTPartner.GetIncomingMessageChannel() <- m

}
