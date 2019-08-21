package bft

type ProposalCondition struct {
	ValidHeight uint64
}

// ProposalGenerator is called when a proposal is needed
type ProposalGenerator interface {
	ProduceProposal() (proposal Proposal, validCondition ProposalCondition)
}

// ProposalValidator is called when a proposal is needed to be verified
type ProposalValidator interface {
	ValidateProposal(proposal Proposal) error
}

type DecisionMaker interface {
	MakeDecision(proposal Proposal, state *HeightRoundState) (ConsensusDecision, error)
}

type BftPeerCommunicator interface {
	Broadcast(msg BftMessage, peers []PeerInfo)
	Unicast(msg BftMessage, peer PeerInfo)
	GetIncomingChannel() chan BftMessage
	Run()
}

type BftOperator interface {
	StartNewEra(height uint64, round int)
	WaiterLoop()
	EventLoop()
	GetPeerCommunicator() BftPeerCommunicator
	Stop()
}

type ConsensusReachedListener interface {
	GetConsensusDecisionMadeEventChannel() chan ConsensusDecision
}