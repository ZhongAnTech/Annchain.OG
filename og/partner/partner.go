package partner

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/annsensus/bft"
	"github.com/sirupsen/logrus"
)

// partner is a participant in the consensus group.
// partner does not care which consensus method is being used in the bottom layer.
// it only provides necessary functions and infomation to support consensus module.
// e.g., produce proposal, broadcast messages, receive message and update consensus state
type OGPartner struct {
	JudgeNonceFunction func(account *account.Account) uint64
	MessageSigner      crypto.ISigner
}

func NewOGPartner(myAccount *account.Account) *OGPartner {

}

func (o *OGPartner) Sign(msg bft.BftMessage) {

}

func (o *OGPartner) ProduceProposal() (proposal bft.Proposal, validCondition bft.ProposalCondition) {
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
	proposal := bft.SequencerProposal{
		Sequencer: *seq,
	}
	return &proposal, seq.Height
}
