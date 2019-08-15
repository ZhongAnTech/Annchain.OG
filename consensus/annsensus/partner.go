package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/communicator"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/model"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
)

var (
	SequencerGenerationRetryTimes = 7
)

// partner is a participant in the consensus group.
// partner does not care which consensus method is being used in the bottom layer.
// it only provides necessary functions and infomation to support consensus module.
// e.g., produce proposal, broadcast messages, receive message and update consensus state
// When there comes a term change, reset bftPartnerMyself, dkg
type AnnsensusPartner struct {
	accountNonceProvider    AccountNonceProvider
	accountProvider         ConsensusAccountProvider
	peerCommunicator        bft.BftPeerCommunicator // AnnsensusPartner is a BftPeerCommunicator, peerCommunicator is a PeerCommunicator
	bftPartnerMyself        *bft.BftOperator
	dkg                     *dkg.Dkg
	termProvider            TermProvider
	heightProvider          HeightProvider
	sequencerProducer       SequencerProducer
	consensusReachedChannel chan model.ConsensusDecision
	quit                    chan bool
}

func NewAnnsensusPartner(accountNonceProvider AccountNonceProvider, peerCommunicator bft.BftPeerCommunicator,
	termProvider TermProvider, accountProvider ConsensusAccountProvider, sequencerProducer SequencerProducer) *AnnsensusPartner {

	ap := &AnnsensusPartner{
		accountNonceProvider:    accountNonceProvider,
		accountProvider:         accountProvider,
		peerCommunicator:        peerCommunicator,
		termProvider:            termProvider,
		sequencerProducer:       sequencerProducer,
		consensusReachedChannel: make(chan model.ConsensusDecision),
		quit:                    make(chan bool),
	}
	trustfulPeerCommunicator := communicator.NewTrustfulPeerCommunicator(signer, termProvider, accountProvider)

	ap := &AnnsensusPartner{
		peerCommunicator:     trustfulPeerCommunicator,
		bftPartnerMyself:     bft.NewDefaultBFTPartner(nParticipants, id, blockTime),
		accountNonceProvider: accountNonceProvider,
	}
	ap.bftPartnerMyself.RegisterConsensusReachedListener(ap)

	return ap
}

func (o *AnnsensusPartner) Start() {
	// start loop
	goroutine.New(o.loop)
}

func (o *AnnsensusPartner) Stop() {
	panic("implement me")
}

func (o *AnnsensusPartner) Name() string {
	panic("implement me")
}

// MakeDecision here is the final validator for recovering BLS threshold signature for this Proposal.
// It is not the same as the one in verifiers. Those are for normal tx validation for all nodes.
func (o *AnnsensusPartner) MakeDecision(proposal model.Proposal, state *bft.HeightRoundState) (model.ConsensusDecision, error) {
	var sigShares [][]byte
	sequencerProposal := proposal.(*model.SequencerProposal)
	// reform bls signature
	for i, commit := range state.PreCommits {
		if commit == nil {
			logrus.WithField("partner", i).WithField("hr", state.MessageProposal.HeightRound).
				Trace("parnter commit is nil")
			continue
		}
		//logrus.WithField("len", len(commit.BlsSignature)).WithField("sigs", hexutil.Encode(commit.BlsSignature)).
		//	Trace("commit", commit)
		sigShares = append(sigShares, commit.BlsSignature)
	}
	// TODO: concurrency check for currentTerm
	currentTerm := o.termProvider.CurrentTerm()

	jointSig, err := o.dkg.RecoverAndVerifySignature(sigShares, sequencerProposal.GetId().ToBytes(), currentTerm)
	if err != nil {
		logrus.WithField("termId", currentTerm).WithError(err).Warn("joint sig verification failed")
		return nil, err
	}
	sequencerProposal.BlsJointSig = jointSig
	// TODO: may set the pubkey
	sequencerProposal.Proposing = false
	return sequencerProposal, nil
}

func (o *AnnsensusPartner) GetConsensusDecisionMadeEventChannel() chan model.ConsensusDecision {
	return o.consensusReachedChannel
}

// ValidateProposal is called once a proposal is received from consensus peers
//
func (o *AnnsensusPartner) ValidateProposal(proposal model.Proposal) error {
	if err := o.validateSignature(proposal); err != nil {
		return err
	}
	if err := o.validateTerm(proposal); err != nil {
		return err
	}

	h := proposal.BasicMessage.HeightRound
	id := b.BFTPartner.Proposer(h)
	if uint16(id) != proposal.SourceId {
		if proposal.BasicMessage.TermId == uint32(b.DKGTermId)-1 {
			//former term message
			//TODO optimize in the future
		}
		logrus.Warn("not your turn")
		return false
	}

	if !b.VerifyIsPartNer(pubkey, int(id)) {
		logrus.Warn("verify pubkey error")
		return false
	}
	return true
}

func (o *AnnsensusPartner) ProduceProposal() (proposal model.Proposal, validCondition bft.ProposalCondition) {
	me := o.accountProvider.Account()
	nonce := o.accountNonceProvider.GetNonce(me)
	logrus.WithField("nonce", nonce).Debug("gen seq")
	targetHeight := o.heightProvider.CurrentHeight() + 1
	targetTermId := o.termProvider.HeightTerm(targetHeight)
	blsPub, err := o.dkg.GetJoinPublicKey(targetTermId).MarshalBinary()

	if err != nil {
		logrus.WithField("term", targetTermId).WithField("height", targetHeight).
			WithError(err).Error("error on getting joint public key")
		panic(err)
	}
	var seq *tx_types.Sequencer

	for i := 0; i < SequencerGenerationRetryTimes; i++ {
		innerSequencer, err, genAgain := o.sequencerProducer.GenerateSequencer(me.Address, targetHeight, nonce, &me.PrivateKey, blsPub)
		if err != nil {
			logrus.WithError(err).WithField("times", i).Warn("gen sequencer failed")
			if !genAgain {
				break
			}
			logrus.WithField("retry", i).Warn("try to generate sequencer again")
		} else {
			seq = innerSequencer
		}
	}
	if seq == nil {
		logrus.WithField("height", targetHeight).WithField("term", targetTermId).Error("failed to generate sequencer")
		return
	}
	return &model.SequencerProposal{Sequencer: *seq}, bft.ProposalCondition{ValidHeight: targetHeight}
}

func (o *AnnsensusPartner) loop() {
	// wait for term ready
	// Am I in the term and so that I should participate in the consensus?
	// if yes, init bft
	// once a decision is made, consume, broadcast, and wait for another term ready.
	// term will be ready once seq has been written into the ledger and next term is decided.
	for {
		select {
		case <-o.quit:
			break
		case newTerm := <-o.termProvider.GetTermChangeEventChannel():
			// term changed, init term
			o.handleTermChanged(newTerm)
		case decision := <-o.consensusReachedChannel:
			o.handleConsensusReached(decision)
		}
	}
}

func (o *AnnsensusPartner) handleConsensusReached(decision model.ConsensusDecision) {
	// decision is made, broadcast to others
	fmt.Println(decision)
	logrus.Warn("Here you need to broadcast decision to other non-consensus nodes")
}

func (o *AnnsensusPartner) handleTermChanged(term *term.Term) {
	// init a bft
	bft := bft.NewDefaultBFTPartner()
	fmt.Println(term)
}
