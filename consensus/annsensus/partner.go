package annsensus

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/communicator"
	"github.com/sirupsen/logrus"
	"time"
)

// partner is a participant in the consensus group.
// partner does not care which consensus method is being used in the bottom layer.
// it only provides necessary functions and infomation to support consensus module.
// e.g., produce proposal, broadcast messages, receive message and update consensus state
// When there comes a term change,
type AnnsensusPartner struct {
	accountNonceProvider AccountNonceProvider
	peerCommunicator     bft.BftPeerCommunicator
	bftPartnerMyself     *bft.BftOperator
}

// ValidateProposal is called once a proposal is received from
func (o *AnnsensusPartner) ValidateProposal(proposal bft.Proposal) error {
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

func (o *AnnsensusPartner) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	panic("implement me")
}

func (o *AnnsensusPartner) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	panic("implement me")
}

func (o *AnnsensusPartner) GetIncomingChannel() chan bft.BftMessage {
	panic("implement me")
}

type dummyTermProvider struct {
}

func (dummyTermProvider) CurrentDkgTerm() uint32 {
	return 1
}

func NewAnnsensusPartner(signer crypto.ISigner, accountProvider communicator.ConsensusAccountProvider,
	accountNonceProvider AccountNonceProvider, nParticipants int, id int,
	blockTime time.Duration) *AnnsensusPartner {
	termProvider := dummyTermProvider{}
	trustfulPeerCommunicator := communicator.NewTrustfulPeerCommunicator(signer, termProvider, accountProvider)

	return &AnnsensusPartner{
		peerCommunicator:     trustfulPeerCommunicator,
		bftPartnerMyself:     bft.NewBFTPartner(nParticipants, id, blockTime),
		accountNonceProvider: accountNonceProvider,
	}
}

func (o *AnnsensusPartner) ProduceProposal() (proposal bft.Proposal, validCondition bft.ProposalCondition) {
	me := o.myAccount
	nonce := GetNonce(me)
	logrus.WithField(" nonce ", nonce).Debug("gen seq")
	blsPub, err := b.dkg.GetJoinPublicKey(b.DKGTermId).MarshalBinary()
	if err != nil {
		logrus.WithError(err).Error("unmarshal fail")
		panic(err)
	}
	seq, genAgain := b.creator.GenerateSequencer(me.Address, b.dag.GetHeight()+1, nonce, &me.PrivateKey, blsPub)
	for i := 0; i < 7 && seq == nil; i++ {
		logrus.WithField("times ", i).Warn("gen sequencer failed,try again ")
		seq, genAgain = b.creator.GenerateSequencer(me.Address, b.dag.GetHeight()+1, b.JudgeNonceFunction(me), &me.PrivateKey, blsPub)
		_ = genAgain
	}
	if seq == nil {
		panic("gen sequencer failed")
	}
	proposal := bft.SequencerProposal{
		Sequencer: *seq,
	}
	return &proposal, seq.Height
}
