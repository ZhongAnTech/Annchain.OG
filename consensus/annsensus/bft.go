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

//PBFT is og sequencer consensus system based on PBFT consensus
type PBFT struct {
	BFTPartner       *OGBFTPartner
	startBftChan chan bool
	stopBft    chan bool
	resetChan        chan bool
	mu               sync.RWMutex
	quit             chan bool
	ann              *AnnSensus
	creator                *og.TxCreator
	JudgeNonce       func(account *account.SampleAccount) uint64
	decisionCh       chan *HeightRoundState
	Verifiers        []og.Verifier
}

type OGBFTPartner struct {
	BFTPartner
	PublicKey crypto.PublicKey
	Address   types.Address
}


func NewOgBftPeer(pk crypto.PublicKey,nbParticipants, Id int ,  sequencerTime time.Duration)*OGBFTPartner{
	p := NewBFTPartner(nbParticipants, Id, sequencerTime)
	bft := &OGBFTPartner{
		BFTPartner: p,
		PublicKey:pk,
		Address: pk.Address(),
	}
	return bft
}

func NewPBFT(nbParticipants int, Id int, sequencerTime time.Duration, judgeNonce func(me *account.SampleAccount) uint64,
	txcreator *og.TxCreator, Verifiers []og.Verifier) *PBFT {
	p := NewBFTPartner(nbParticipants, Id, sequencerTime)
	bft := &OGBFTPartner{
		BFTPartner: p,
	}
	om := &PBFT{
		BFTPartner:       bft,
		quit:             make(chan bool),
		startBftChan: make(chan bool),
		resetChan:        make(chan bool),
		decisionCh:       make(chan *HeightRoundState),
		creator:                txcreator,
	}
	om.BFTPartner.SetProposalFunc(om.ProduceProposal)
	om.JudgeNonce = judgeNonce
	bft.RegisterDecisionReceive(om.decisionCh)
	om.Verifiers = Verifiers
	return om
}

func (t *PBFT) Start() {
	go t.BFTPartner.WaiterLoop()
	go t.BFTPartner.EventLoop()
	go t.loop()
	logrus.Info("PBFT stated")
}

func (t *PBFT) SetPeers() {
	t.BFTPartner.SetPeers(nil)
}

func (t *PBFT) Stop() {
	t.BFTPartner.Stop()
	t.quit <- true
	logrus.Info("PBFT stopped")
}

func (t *PBFT) sendToPartners(msgType og.MessageType, request types.Message) {
	peers := t.BFTPartner.GetPeers
	for _, peer := range peers() {
		logrus.WithFields(logrus.Fields{
			"IM":  t.BFTPartner.GetId(),
			"to":  peer.GetId(),
			"msg": request.String(),
		}).Debug("Out")
		bftPeer := peer.(OGBFTPartner)
		if peer.GetId() == t.BFTPartner.GetId() {
			//it is for me
			go func() {
				msg := Message{
					Type:    msgType,
					Payload: request,
				}
				t.BFTPartner.GetIncomingMessageChannel() <- msg
			}()
			continue
		}
		//send to others
		t.ann.Hub.SendToAnynomous(msgType, request, &bftPeer.PublicKey)
	}
}

func (t *PBFT) loop() {
	signer := crypto.NewSigner(t.ann.cryptoType)
	log := logrus.WithField("me", t.BFTPartner.GetId())
	for {
		select {
		case <-t.quit:
			log.Info("got quit signal, PBFT loop")
		case <-t.startBftChan:
			go t.BFTPartner.StartNewEra(0, 0)

		case msg := <-t.BFTPartner.GetOutgoingMessageChannel():
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
					sig, err := t.ann.dkg.partner.Sig(preCommit.Idv.ToBytes())
					if err != nil {
						panic(err)
						preCommit.BlsSignature = sig
					}
				}
				preCommit.PublicKey = t.ann.MyAccount.PublicKey.Bytes
				preCommit.Signature = signer.Sign(t.ann.MyAccount.PrivateKey, preCommit.SignatureTargets()).Bytes
				t.sendToPartners(msg.Type, preCommit)
			default:
				panic("never come here unknown type")
			}

		case decision := <-t.decisionCh:
			seq := decision.MessageProposal.Value.(*types.SequencerProposal)
			for _, commit := range decision.PreCommits {
				//blsSig := &types.BlsSigSet{
				//	PublicKey:    commit.PublicKey,
				//	BlsSignature: commit.BlsSignature,
				//}
				//blsSigsets = append(blsSigsets, blsSig)
				t.ann.dkg.partner.SigShares = append(t.ann.dkg.partner.SigShares, commit.BlsSignature)
			}
			jointSig, err := t.ann.dkg.partner.RecoverSig(seq.GetId().ToBytes())
			if err != nil {
				log.Warnf("partner %d cannot recover jointSig with %d sigshares: %s\n",
					t.ann.dkg.partner.Id, len(t.ann.dkg.partner.SigShares), err)
				continue
			}
			log.Debugf("threshold signature from partner %d: %s\n", t.ann.dkg.partner.Id, hexutil.Encode(jointSig))
			// verify if JointSig meets the JointPubkey
			err = t.ann.dkg.partner.VerifyByDksPublic(seq.GetId().ToBytes(), jointSig)
			if err == nil {
				// verify if JointSig meets the JointPubkey
				err = t.ann.dkg.partner.VerifyByPubPoly(seq.GetId().ToBytes(), jointSig)
			}
			if err != nil {
				log.WithError(err).Warnf("joinsig verify failed ")
				continue
			}
			t.ann.Hub.BroadcastMessage(og.MessageTypeNewSequencer, seq.RawSequencer())

			case <-t.resetChan:

		}

	}
}

func (t *PBFT) ProduceProposal() types.Proposal {
	me := t.ann.MyAccount
	nonce := t.JudgeNonce(me)
	seq := t.creator.GenerateSequencer(me.Address, t.ann.Idag.LatestSequencer().Height, nonce, &me.PrivateKey)
	if seq == nil {
		logrus.Warn("gen sequencer failed")
		panic("gen sequencer failed")
	}
	proposal := types.SequencerProposal{
		Sequencer: *seq,
	}
	return &proposal
}

func (t *PBFT) verifyProposal(proposal *types.MessageProposal, pubkey crypto.PublicKey) bool {
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
	msg := proposal.Value.(*types.SequencerProposal)
	//
	for _, verifier := range t.Verifiers {
		if !verifier.Verify(&msg.Sequencer) {
			logrus.Warn("sequencer verify failed")
			return false
		}
	}
	return true
}

func (t *PBFT) verifyIsPartNer(publicKey crypto.PublicKey, sourcePartner int) bool {
	peers := t.BFTPartner.GetPeers()
	if sourcePartner < 0 || sourcePartner >= len(peers) {
		return false
	}
	partner := peers[sourcePartner].(OGBFTPartner)
	if bytes.Equal(partner.PublicKey.Bytes, publicKey.Bytes) {
		return true
	}
	return false

}

//func (t*PBFT)VerifyPrevote( msg *types.MessagePreVote) bool{
//  return true
//}
//
//func (t*PBFT)VerifyPreCommit(msg *types.MessagePreCommit ) bool{
// return true
//}


//calculate seed
func CalculateRandomSeed(jointSig []byte) []byte {
	//TODO
	h := sha256.New()
	h.Write(jointSig)
    seed := h.Sum(nil)
    return seed
}