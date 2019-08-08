package bft

type ProposalCondition struct {
	ValidHeight uint64
}

// ProposalGenerator is called when a proposal is needed
type ProposalGenerator interface {
	ProduceProposal() (proposal Proposal, validCondition ProposalCondition)
}

// HeightProvider is called when a height is needed
type HeightProvider interface {
	CurrentHeight() uint64
}

type PeerCommunicator interface {
	Broadcast(msg Message, peers []PeerInfo)
	Unicast(msg Message, peer PeerInfo)
	GetIncomingChannel() chan Message
}
