package bft

import "github.com/annchain/OG/consensus/model"

type dummyBftPeerCommunicator struct {
	Myid     int
	Peers    []chan BftMessage
	Incoming chan BftMessage
}

func (d *dummyBftPeerCommunicator) Broadcast(msg BftMessage, peers []PeerInfo) {
	for _, peer := range peers {
		go func(peer PeerInfo) {
			d.Peers[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyBftPeerCommunicator) Unicast(msg BftMessage, peer PeerInfo) {
	go func() {
		d.Peers[peer.Id] <- msg
	}()
}

func (d *dummyBftPeerCommunicator) GetIncomingChannel() chan BftMessage {
	return d.Peers[d.Myid]
}

func NewDummyBftPeerCommunicator(myid int, incoming chan BftMessage, peers []chan BftMessage) *dummyBftPeerCommunicator {
	d := &dummyBftPeerCommunicator{
		Peers:    peers,
		Myid:     myid,
		Incoming: incoming,
	}
	return d
}

type dummyProposalGenerator struct {
	CurrentHeight uint64
}

func (d dummyProposalGenerator) ProduceProposal() (proposal model.Proposal, validCondition ProposalCondition) {
	p := model.StringProposal("xxx")
	return &p, ProposalCondition{ValidHeight: d.CurrentHeight}
}

type dummyProposalValidator struct {
}

func (d dummyProposalValidator) ValidateProposal(proposal model.Proposal) error {
	return nil
}

type dummyDecisionMaker struct {
}

func (d dummyDecisionMaker) MakeDecision(proposal model.Proposal, state *HeightRoundState) (model.ConsensusDecision, error) {
	return proposal, nil
}
