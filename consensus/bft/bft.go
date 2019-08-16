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
package bft

//import (
//	"github.com/annchain/OG/account"
//	"github.com/annchain/OG/common"
//	"github.com/annchain/OG/common/crypto"
//	"github.com/annchain/OG/common/goroutine"
//	"github.com/annchain/OG/common/hexutil"
//	"github.com/annchain/OG/consensus/dkg"
//	"github.com/annchain/OG/og"
//	"github.com/annchain/OG/og/txmaker"
//	"github.com/annchain/OG/types"
//	"github.com/annchain/OG/types/p2p_message"
//	"github.com/sirupsen/logrus"
//	log "github.com/sirupsen/logrus"
//	"sync"
//	"time"
//)
//
////BFT the consensus module for partner to
//type BFT struct {
//	BFTPartner   *OGBFTPartner `json:"bft_partner"`
//	startBftChan chan bool
//	resetChan    chan bool
//	mu           sync.RWMutex
//	quit         chan bool
//	creator      *txmaker.OGTxCreator
//
//	decisionChan chan *commitDecision
//	//Verifiers     []protocol.Verifier
//	proposalCache map[common.Hash]*MessageProposal
//
//	//DKGTermId     int           `json:"dkg_term_id"`
//	SequencerTime time.Duration `json:"sequencer_time"`
//	//dkg           *dkg.Dkg
//
//	//NbParticipants int
//
//	Hub announcer.MessageSender
//
//	//dag og.IDag
//
//	myAccount *account.Account
//
//	started bool
//
//	OnSelfGenTxi chan types.Txi
//}
//
//type commitDecision struct {
//	callbackChan chan error
//	state        *HeightRoundState
//}
//
//func NewOgBftPeer(pk crypto.PublicKey, nbParticipants, Id int, sequencerTime time.Duration) *OGBFTPartner {
//	p := NewDefaultBFTPartner(nbParticipants, Id, sequencerTime)
//	bft := &OGBFTPartner{
//		BFTPartner: p,
//		PeerInfo: PeerInfo{
//			PublicKey:      pk,
//			Address:        pk.Address(),
//			PublicKeyBytes: pk.Bytes[:],
//		},
//	}
//	return bft
//}
//
//
//func NewBFT(nbParticipants int, Id int, sequencerTime time.Duration, judgeNonceFunction func(me *account.Account) uint64,
//	txCreator *txmaker.OGTxCreator, dag og.IDag, myAccount *account.Account, OnSelfGenTxi chan types.Txi, dkg *dkg.Dkg) *BFT {
//	partner := NewDefaultBFTPartner(nbParticipants, Id, sequencerTime)
//	ogBftPartner := &OGBFTPartner{
//		BFTPartner: partner,
//		PeerInfo: PeerInfo{
//			PublicKey:      myAccount.PublicKey,
//			PublicKeyBytes: myAccount.PublicKey.Bytes[:],
//			Address:        myAccount.Address,
//		},
//	}
//	bft := &BFT{
//		BFTPartner:    ogBftPartner,
//		quit:          make(chan bool),
//		startBftChan:  make(chan bool),
//		resetChan:     make(chan bool),
//		decisionChan:  make(chan *commitDecision),
//		creator:       txCreator,
//		proposalCache: make(map[common.Hash]*MessageProposal),
//	}
//	bft.BFTPartner.SetProposalFunc(bft.ProduceProposal)
//	bft.BFTPartner.SetGetHeightFunc(dag.GetHeight)
//	bft.JudgeNonceFunction = judgeNonceFunction
//	bft.SequencerTime = sequencerTime
//	bft.OnSelfGenTxi = OnSelfGenTxi
//	//bft.dag = dag
//	bft.myAccount = myAccount
//	//bft.dkg = dkg
//	//bft.NbParticipants
//	ogBftPartner.RegisterDecisionReceiveFunc(bft.commitDecision)
//	return bft
//}

//func (b *BFT) Started() bool {
//	return b.started
//}

//func (b *BFT) Reset(TermId int, peersPublicKey []crypto.PublicKey, myId int) {
//	b.mu.RLock()
//	defer b.mu.RUnlock()
//	b.DKGTermId = TermId
//	var peers []BFTPartner
//	b.BFTPartner.PeersInfo = nil
//	for i, pk := range peersPublicKey {
//		//the third param is not used in peer
//		b.BFTPartner.PeersInfo = append(b.BFTPartner.PeersInfo, PeerInfo{Address: pk.Address(), PublicKey: pk, PublicKeyBytes: pk.Bytes[:]})
//		peers = append(peers, NewOgBftPeer(pk, len(peersPublicKey), i, time.Second))
//	}
//	b.BFTPartner.Reset(len(peersPublicKey), myId)
//	b.BFTPartner.SetPeers(peers)
//	log.WithField("len pks ", len(peersPublicKey)).WithField("len peers ", len(peers)).WithField("my id ", myId).WithField("with peers ", peers).WithField("term Id ", TermId).Debug("bft will reset")
//	//TODO immediately change round ?
//}

//func (b *BFT) Start() {
//	goroutine.New(b.BFTPartner.WaiterLoop)
//	goroutine.New(b.BFTPartner.EventLoop)
//	goroutine.New(b.loop)
//	logrus.Info("BFT started")
//}

//func (b *BFT) ReSetPeers() {
//	b.BFTPartner.SetPeers(nil)
//}
//
//func (b *BFT) StartGossip() {
//	b.startBftChan <- true
//}

//func (b *BFT) Stop() {
//	log.Info("BFT will stop")
//	b.BFTPartner.Stop()
//	close(b.quit)
//	logrus.Info("BFT stopped")
//}

//func (b *BFT) commitDecision(state *HeightRoundState) error {
//	commit := commitDecision{
//		state:        state,
//		callbackChan: make(chan error),
//	}
//	b.decisionChan <- &commit
//	// waiting for callback
//	select {
//	case err := <-commit.callbackChan:
//		if err != nil {
//			return err
//		}
//	}
//	log.Trace("commit success")
//	return nil
//}

//func (b *BFT) sendToPartners(msgType BftMessageType, request p2p_message.Message) {
//	inChan := b.BFTPartner.GetIncomingMessageChannel()
//	peers := b.BFTPartner.GetPeers()
//	for _, peer := range peers {
//		logrus.WithFields(logrus.Fields{
//			"IM":   b.BFTPartner.GetId(),
//			"to":   peer.GetId(),
//			"type": msgType.String(),
//			"msg":  request.String(),
//		}).Debug("Out")
//		bftPeer := peer.(*OGBFTPartner)
//		if peer.GetId() == b.BFTPartner.GetId() {
//			//it is for me
//			goroutine.New(
//				func() {
//					time.Sleep(10 * time.Millisecond)
//					msg := BftMessage{
//						Type:    msgType,
//						Payload: request,
//					}
//					inChan <- msg
//				})
//			continue
//		}
//		//send to others
//		logrus.WithField("msg", request).Debug("send msg")
//		b.Hub.AnonymousSendMessage(msgType, request, &bftPeer.PublicKey)
//	}
//}
//
//func (b *BFT) loop() {
//	outCh := b.BFTPartner.GetOutgoingMessageChannel()
//	logev := log.WithField("me", b.BFTPartner.GetId())
//	for {
//		select {
//		case <-b.quit:
//			logev.Info("got quit signal, BFT loop")
//			return
//		case <-b.startBftChan:
//			logev.Info("bft got start gossip signal")
//			if !b.started {
//				goroutine.New(func() {
//					b.BFTPartner.StartNewEra(b.dag.GetHeight(), 0)
//				})
//			}
//			b.started = true
//
//		case msg := <-outCh:
//			logev.Tracef("got msg %v", msg)
//			// sign and send msg
//
//		case decision := <-b.decisionChan:
			//state := decision.state
			////set nil first
			//var sigShares [][]byte
			//sequencerProposal := state.Decision.(*SequencerProposal)
			//for i, commit := range state.PreCommits {
			//	//blsSig := &types.BlsSigSet{
			//	//	PublicKey:    commit.PublicKey,
			//	//	BlsSignature: commit.BlsSignature,
			//	//}
			//	//blsSigsets = append(blsSigsets, blsSig)
			//	if commit == nil {
			//		logev.WithField("i ", i).Debug("commit is nil")
			//		continue
			//	}
			//	logev.WithField("len ", len(commit.BlsSignature)).WithField("sigs ", hexutil.Encode(commit.BlsSignature))
			//	logev.Debug("commit ", commit)
			//	sigShares = append(sigShares, commit.BlsSignature)
			//}
			//jointSig, err := b.dkg.RecoverAndVerifySignature(sigShares, sequencerProposal.GetId().ToBytes(), b.DKGTermId)
			//if err != nil {
			//	logev.WithField("termId ", b.DKGTermId).WithError(err).Warnf("joinsig verify failed ")
			//
			//	decision.callbackChan <- err
			//	continue
			//} else {
			//	decision.callbackChan <- nil
			//}
			//
			//sequencerProposal.BlsJointSig = jointSig
			//logev.Debug("will send buffer")
			////seq.BlsJointPubKey = blsPub
			//Sequencer.Proposing = false
			//b.OnSelfGenTxi <- &Sequencer
			////b.ann.Hub.BroadcastMessage(BftMessageTypeNewSequencer, seq.RawSequencer())

//		case <-b.resetChan:
//			//todo
//		}
//
//	}
//}

//func (b *BFT) VerifyProposal(proposal *MessageProposal, pubkey crypto.PublicKey) bool {
//	h := proposal.BftBasicInfo.HeightRound
//	id := b.BFTPartner.Proposer(h)
//	if uint16(id) != proposal.SourceId {
//		if proposal.BftBasicInfo.TermId == uint32(b.DKGTermId)-1 {
//			//former term message
//			//TODO optimize in the future
//		}
//		logrus.Warn("not your turn")
//		return false
//	}
//
//	if !b.VerifyIsPartNer(pubkey, int(id)) {
//		logrus.Warn("verify pubkey error")
//		return false
//	}
//	//will verified in buffer
//	//msg := proposal.Value.(*tx_types.SequencerProposal)
//	//
//	//for _, verifier := range b.Verifiers {
//	//	if !verifier.Verify(&msg.Sequencer) {
//	//		logrus.Warn("sequencer verify failed")
//	//		return false
//	//	}
//	//}
//	return true
//}

//func (b *BFT) VerifyIsPartNer(publicKey crypto.PublicKey, sourcePartner int) bool {
//	peers := b.BFTPartner.GetPeers()
//	if sourcePartner < 0 || sourcePartner > len(peers)-1 {
//		logrus.WithField("len partner ", len(peers)).WithField("sr ", sourcePartner).Warn("sourceId error")
//		return false
//	}
//	partner := peers[sourcePartner].(*OGBFTPartner)
//	if bytes.Equal(partner.PublicKey.Bytes, publicKey.Bytes) {
//		return true
//	}
//	logrus.Trace(publicKey.String(), " ", partner.PublicKey.String())
//	return false
//
//}

//func (b *BFT) GetProposalCache(hash common.Hash) *MessageProposal {
//	b.mu.RLock()
//	defer b.mu.RUnlock()
//	return b.proposalCache[hash]
//}
//
//func (b *BFT) DeleteProposalCache(hash common.Hash) {
//	b.mu.RLock()
//	defer b.mu.RUnlock()
//	delete(b.proposalCache, hash)
//}
//func (b *BFT) CacheProposal(hash common.Hash, proposal *MessageProposal) {
//	b.mu.RLock()
//	defer b.mu.RUnlock()
//	b.proposalCache[hash] = proposal
//}

//func (b *BFT) GetInfo() *BFTInfo {
//	bftInfo := BFTInfo{
//		BFTPartner:    b.BFTPartner.PeerInfo,
//		Partners:      b.BFTPartner.PeersInfo,
//		DKGTermId:     b.DKGTermId,
//		SequencerTime: b.SequencerTime,
//	}
//	return &bftInfo
//}
//
//type BFTInfo struct {
//	BFTPartner    PeerInfo      `json:"bft_partner"`
//	DKGTermId     int           `json:"dkg_term_id"`
//	SequencerTime time.Duration `json:"sequencer_time"`
//	Partners      []PeerInfo    `json:"partners"`
//}

//
//func (b *BFT) HandlePreCommit(request *MessagePreCommit) {
//	m := BftMessage{
//		Type:    BftMessageTypePreCommit,
//		Payload: request,
//	}
//	b.BFTPartner.GetIncomingMessageChannel() <- m
//}
//
//func (b *BFT) HandlePreVote(request *MessagePreVote) {
//	m := BftMessage{
//		Type:    BftMessageTypePreVote,
//		Payload: request,
//	}
//	b.BFTPartner.GetIncomingMessageChannel() <- m
//}
//
//func (b *BFT) HandleProposal(hash common.Hash) {
//	request := b.GetProposalCache(hash)
//	if request != nil {
//		b.DeleteProposalCache(hash)
//		m := BftMessage{
//			Type:    BftMessageTypeProposal,
//			Payload: request,
//		}
//		b.BFTPartner.GetIncomingMessageChannel() <- m
//
//	}
//}
//
//func (b *BFT) GetStatus() interface{} {
//	return b.BFTPartner.Status()
//}
