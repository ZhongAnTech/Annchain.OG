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

type BftPeerCommunicatorOutgoing interface {
	Broadcast(msg *BftMessage, peers []PeerInfo)
	Unicast(msg *BftMessage, peer PeerInfo)
}
type BftPeerCommunicatorIncoming interface {
	GetPipeIn() chan BftMessage
	GetPipeOut() chan BftMessage
}

type BftPartner interface {
	StartNewEra(height uint64, round int)
	Stop()
	WaiterLoop()
	EventLoop()
	GetBftPeerCommunicatorIncoming() BftPeerCommunicatorIncoming
	RegisterConsensusReachedListener(listener ConsensusReachedListener)
}

type ConsensusReachedListener interface {
	GetConsensusDecisionMadeEventChannel() chan ConsensusDecision
}
