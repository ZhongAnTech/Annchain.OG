package bft

type ProposalCondition struct {
	ValidHeight uint64
}

// ProposalGenerator is called when a proposal is needed
type ProposalGenerator interface {
	ProduceProposal() (proposal Proposal, validCondition ProposalCondition)
}

// ProposalGenerator is called when a proposal is needed
type ProposalValidator interface {
	ValidateProposal(proposal Proposal) error
}

// HeightProvider is called when a height is needed
type HeightProvider interface {
	CurrentHeight() uint64
}

type BftPeerCommunicator interface {
	Broadcast(msg BftMessage, peers []PeerInfo)
	Unicast(msg BftMessage, peer PeerInfo)
	GetIncomingChannel() chan BftMessage
}
