package partner

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/annsensus/bft"
	"github.com/sirupsen/logrus"
)

// ReliableBftSender signs and validate messages using pubkey/privkey given by DKG/BLS
type ReliableBftSender struct {
	buffer       chan bft.BftMessage
	Signer       crypto.ISigner
	TermProvider DkgTermProvider
}

func (r *ReliableBftSender) Sign(msg bft.BftMessage) {
	logrus.Tracef("signing msg %v", msg)

	switch msg.Type {
	case bft.BftMessageTypeProposal:
		proposal := msg.Payload.(*bft.MessageProposal)
		proposal.Signature = r.Signer.Sign(b.myAccount.PrivateKey, proposal.SignatureTargets()).Bytes
		proposal.TermId = uint32(b.DKGTermId)
		b.sendToPartners(msg.Type, proposal)
	case bft.BftMessageTypePreVote:
		prevote := msg.Payload.(*bft.MessagePreVote)
		prevote.PublicKey = b.myAccount.PublicKey.Bytes
		prevote.Signature = r.Signer.Sign(b.myAccount.PrivateKey, prevote.SignatureTargets()).Bytes
		prevote.TermId = uint32(b.DKGTermId)
		b.sendToPartners(msg.Type, prevote)
	case bft.BftMessageTypePreCommit:
		preCommit := msg.Payload.(*bft.MessagePreCommit)
		if preCommit.Idv != nil {
			logrus.WithField("dkg id ", b.dkg.GetId()).WithField("term id ", b.DKGTermId).Debug("signed ")
			sig, err := b.dkg.Sign(preCommit.Idv.ToBytes(), b.DKGTermId)
			if err != nil {
				logrus.WithError(err).Error("sign error")
				panic(err)
			}
			preCommit.BlsSignature = sig
		}
		preCommit.PublicKey = b.myAccount.PublicKey.Bytes
		preCommit.Signature = crypto.Signer.Sign(b.myAccount.PrivateKey, preCommit.SignatureTargets()).Bytes
		preCommit.TermId = uint32(b.DKGTermId)
		b.sendToPartners(msg.Type, preCommit)
	default:
		panic("never come here unknown type")
	}
}
