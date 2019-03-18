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
	"bytes"
	"crypto/sha256"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

//BFT is og sequencer consensus system based on BFT consensus
type BFT struct {
	BFTPartner         *OGBFTPartner
	startBftChan       chan bool
	resetChan          chan bool
	mu                 sync.RWMutex
	quit               chan bool
	ann                *AnnSensus
	creator            *og.TxCreator
	JudgeNonceFunction func(account *account.SampleAccount) uint64
	decisionChan       chan *commitDecision
	//Verifiers     []og.Verifier
	proposalCache map[types.Hash]*types.MessageProposal
	DKGTermId     int
	sequencerTime time.Duration
}

//OGBFTPartner implements BFTPartner
type OGBFTPartner struct {
	BFTPartner
	PublicKey crypto.PublicKey
	Address   types.Address
}

func (p *OGBFTPartner) EventLoop() {
	go p.BFTPartner.(*DefaultPartner).receive()
}

func NewOgBftPeer(pk crypto.PublicKey, nbParticipants, Id int, sequencerTime time.Duration) *OGBFTPartner {
	p := NewBFTPartner(nbParticipants, Id, sequencerTime)
	bft := &OGBFTPartner{
		BFTPartner: p,
		PublicKey:  pk,
		Address:    pk.Address(),
	}
	return bft
}

func NewBFT(ann *AnnSensus, nbParticipants int, Id int, sequencerTime time.Duration, judgeNonceFunction func(me *account.SampleAccount) uint64,
	txCreator *og.TxCreator) *BFT {
	p := NewBFTPartner(nbParticipants, Id, sequencerTime)
	bft := &OGBFTPartner{
		BFTPartner: p,
	}
	om := &BFT{
		BFTPartner:    bft,
		quit:          make(chan bool),
		startBftChan:  make(chan bool),
		resetChan:     make(chan bool),
		decisionChan:  make(chan *commitDecision),
		creator:       txCreator,
		proposalCache: make(map[types.Hash]*types.MessageProposal),
	}
	om.BFTPartner.SetProposalFunc(om.ProduceProposal)
	om.JudgeNonceFunction = judgeNonceFunction
	om.sequencerTime = sequencerTime
	bft.RegisterDecisionReceiveFunc(om.commitDecision)
	om.ann = ann
	return om
}
func (b *BFT) AddPeers(peersPublicKey []crypto.PublicKey) {

}

func (b *BFT) Reset(TermId int, peersPublicKey []crypto.PublicKey, myId int) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	b.DKGTermId = TermId
	log.WithField("term Id ",TermId).Debug("bft will reset")
	var peers []BFTPartner
	for i, pk := range peersPublicKey {
		//the third param is not used in peer
		peers = append(peers, NewOgBftPeer(pk, b.ann.NbParticipants, i, time.Second))
	}
	b.BFTPartner.Reset(len(peersPublicKey), myId)
	b.BFTPartner.SetPeers(peers)
	//TODO immediately change round ?
}

func (t *BFT) Start() {
	go t.BFTPartner.WaiterLoop()
	go t.BFTPartner.EventLoop()
	go t.loop()
	logrus.Info("BFT started")
}

func (t *BFT) SetPeers() {
	t.BFTPartner.SetPeers(nil)
}

func (t *BFT) Stop() {
	t.BFTPartner.Stop()
	t.quit <- true
	logrus.Info("BFT stopped")
}

func (t *BFT) sendToPartners(msgType og.MessageType, request types.Message) {
	inChan := t.BFTPartner.GetIncomingMessageChannel()
	peers := t.BFTPartner.GetPeers()
	for _, peer := range peers {
		logrus.WithFields(logrus.Fields{
			"IM":   t.BFTPartner.GetId(),
			"to":   peer.GetId(),
			"type": msgType.String(),
			"msg":  request.String(),
		}).Debug("Out")
		bftPeer := peer.(*OGBFTPartner)
		if peer.GetId() == t.BFTPartner.GetId() {
			//it is for me
			go func() {
				time.Sleep(10 * time.Millisecond)
				msg := Message{
					Type:    msgType,
					Payload: request,
				}
				inChan <- msg
			}()
			continue
		}
		//send to others
		logrus.WithField("msg ", request).Debug("send msg ")
		t.ann.Hub.SendToAnynomous(msgType, request, &bftPeer.PublicKey)
	}
}

func (t *BFT) loop() {
	outCh := t.BFTPartner.GetOutgoingMessageChannel()
	signer := crypto.NewSigner(t.ann.cryptoType)
	log := logrus.WithField("me", t.BFTPartner.GetId())
	for {
		select {
		case <-t.quit:
			log.Info("got quit signal, BFT loop")
		case <-t.startBftChan:
			go t.BFTPartner.StartNewEra(0, 0)

		case msg := <-outCh:
			log.Tracef("got msg %v", msg)
			switch msg.Type {
			case og.MessageTypeProposal:
				proposal := msg.Payload.(*types.MessageProposal)
				proposal.Signature = signer.Sign(t.ann.MyAccount.PrivateKey, proposal.SignatureTargets()).Bytes
				t.sendToPartners(msg.Type, proposal)
			case og.MessageTypePreVote:
				prevote := msg.Payload.(*types.MessagePreVote)
				prevote.PublicKey = t.ann.MyAccount.PublicKey.Bytes
				prevote.Signature = signer.Sign(t.ann.MyAccount.PrivateKey, prevote.SignatureTargets()).Bytes
				t.sendToPartners(msg.Type, prevote)
			case og.MessageTypePreCommit:
				preCommit := msg.Payload.(*types.MessagePreCommit)
				if preCommit.Idv != nil {
					log.WithField("dkg id ",t.ann.dkg.partner.Id).WithField("term id ",t.DKGTermId).Debug("signed ")
					sig, err := t.ann.dkg.Sign(preCommit.Idv.ToBytes(), t.DKGTermId)
					if err != nil {
						log.WithError(err).Error("sign error")
						panic(err)
					}
					preCommit.BlsSignature = sig
				}
				preCommit.PublicKey = t.ann.MyAccount.PublicKey.Bytes
				preCommit.Signature = signer.Sign(t.ann.MyAccount.PrivateKey, preCommit.SignatureTargets()).Bytes
				t.sendToPartners(msg.Type, preCommit)
			default:
				panic("never come here unknown type")
			}

		case decision := <-t.decisionChan:
			state:= decision.state
			//set nil first
			var sigShares [][]byte
			sequencerProposal := state.Decision.(*types.SequencerProposal)
			for i, commit := range state.PreCommits {
				//blsSig := &types.BlsSigSet{
				//	PublicKey:    commit.PublicKey,
				//	BlsSignature: commit.BlsSignature,
				//}
				//blsSigsets = append(blsSigsets, blsSig)
				if commit == nil {
					log.WithField("i ", i).Debug("commit is nil")
					continue
				}
				log.WithField("len ", len(commit.BlsSignature)).WithField("sigs ", hexutil.Encode(commit.BlsSignature))
				log.Debug("commit ", commit)
				sigShares = append(sigShares, commit.BlsSignature)
			}
			jointSig, err := t.ann.dkg.RecoverAndVerifySignature(sigShares, sequencerProposal.GetId().ToBytes(), t.DKGTermId)
			if err != nil {
				log.WithField("termId ", t.DKGTermId).WithError(err).Warnf("joinsig verify failed ")
				decision.callbackChan <-err
				continue
			}else {
				decision.callbackChan <-nil
			}
			sequencerProposal.BlsJointSig = jointSig
			//seq.BlsJointPubKey = blsPub
			t.ann.OnSelfGenTxi <- &sequencerProposal.Sequencer
			//t.ann.Hub.BroadcastMessage(og.MessageTypeNewSequencer, seq.RawSequencer())

		case <-t.resetChan:

		}

	}
}

type commitDecision struct {
	callbackChan chan error
	state *HeightRoundState
}

func (t *BFT) commitDecision(state *HeightRoundState) error {
	commit:= commitDecision{
		state:state,
		callbackChan: make(chan error),
	}
	t.decisionChan <-&commit
	// waiting for callback
	select {
	case err := <-commit.callbackChan:
		if err != nil {
			return err
		}
	}
	log.Trace("commit success")
	return nil
}


func (t *BFT) ProduceProposal() (pro types.Proposal, validHeight uint64) {
	me := t.ann.MyAccount
	nonce := t.JudgeNonceFunction(me)
	logrus.WithField(" nonce ", nonce).Debug("gen seq")
	blsPub, err := t.ann.dkg.GetJoinPublicKey(t.DKGTermId).MarshalBinary()
	if err != nil {
		logrus.WithError(err).Error("unmarshal fail")
		panic(err)
	}
	seq := t.creator.GenerateSequencer(me.Address, t.ann.Idag.LatestSequencer().Height+1, nonce, &me.PrivateKey, blsPub)
	if seq == nil {
		logrus.Warn("gen sequencer failed")
		panic("gen sequencer failed")
	}
	proposal := types.SequencerProposal{
		Sequencer: *seq,
	}
	return &proposal,seq.Height
}

func (t *BFT) verifyProposal(proposal *types.MessageProposal, pubkey crypto.PublicKey) bool {
	h := proposal.BasicMessage.HeightRound
	id := t.BFTPartner.Proposer(h)
	if uint16(id) != proposal.SourceId {
		logrus.Warn("not your turn")
		return false
	}
	if !t.verifyIsPartNer(pubkey, int(id)) {
		logrus.Warn("verify pubkey error")
		return false
	}
	//will verified in buffer
	//msg := proposal.Value.(*types.SequencerProposal)
	//
	//for _, verifier := range t.Verifiers {
	//	if !verifier.Verify(&msg.Sequencer) {
	//		logrus.Warn("sequencer verify failed")
	//		return false
	//	}
	//}
	return true
}

func (t *BFT) verifyIsPartNer(publicKey crypto.PublicKey, sourcePartner int) bool {
	peers := t.BFTPartner.GetPeers()
	if sourcePartner < 0 || sourcePartner > len(peers)-1 {
		logrus.WithField("len partner ", len(peers)).WithField("sr ", sourcePartner).Debug("sourceId error")
		return false
	}
	partner := peers[sourcePartner].(*OGBFTPartner)
	if bytes.Equal(partner.PublicKey.Bytes, publicKey.Bytes) {
		return true
	}
	logrus.Trace(publicKey.String(), " ", partner.PublicKey.String())
	return false

}

//calculate seed
func CalculateRandomSeed(jointSig []byte) []byte {
	//TODO
	h := sha256.New()
	h.Write(jointSig)
	seed := h.Sum(nil)
	return seed
}
