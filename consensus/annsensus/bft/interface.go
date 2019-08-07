package bft

import "github.com/annchain/OG/types/p2p_message"

type ProposalCondition struct {
	ValidHeight uint64
}

type ProposalGenerator interface {
	ProduceProposal() (proposal p2p_message.Proposal, validCondition ProposalCondition)
}
