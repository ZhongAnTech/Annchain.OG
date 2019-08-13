package bft

import "github.com/annchain/OG/consensus/model"

type ProposalCondition struct {
	ValidHeight uint64
}

// ProposalGenerator is called when a proposal is needed
type ProposalGenerator interface {
	ProduceProposal() (proposal model.Proposal, validCondition ProposalCondition)
}

// ProposalGenerator is called when a proposal is needed
type ProposalValidator interface {
	ValidateProposal(proposal model.Proposal) error
}

type DecisionMaker interface {
	MakeDecision(proposal model.Proposal, state *HeightRoundState) (model.ConsensusDecision, error)
}

type BftPeerCommunicator interface {
	Broadcast(msg BftMessage, peers []PeerInfo)
	Unicast(msg BftMessage, peer PeerInfo)
	GetIncomingChannel() chan BftMessage
}
