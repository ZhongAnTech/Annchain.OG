package partner

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/annsensus/bft"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/sirupsen/logrus"
)

// partner is a proposal generator
type OGPartner struct {
	JudgeNonceFunction func(account *account.Account) uint64
}

func (o *OGPartner) ProduceProposal() (proposal p2p_message.Proposal, validCondition bft.ProposalCondition) {
	me := o.myAccount
	nonce := o.JudgeNonceFunction(me)
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
	proposal := p2p_message.SequencerProposal{
		Sequencer: *seq,
	}
	return &proposal, seq.Height
}
