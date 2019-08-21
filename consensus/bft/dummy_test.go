package bft

import "github.com/annchain/OG/ffchan"

type dummyBftPeerCommunicator struct {
	Myid                   int
	Peers                  []chan BftMessage
	ReceiverChannel        chan BftMessage
	messageProviderChannel chan BftMessage
}

func NewDummyBftPeerCommunicator(myid int, incoming chan BftMessage, peers []chan BftMessage) *dummyBftPeerCommunicator {
	d := &dummyBftPeerCommunicator{
		Peers:                  peers,
		Myid:                   myid,
		ReceiverChannel:        incoming,
		messageProviderChannel: make(chan BftMessage),
	}
	return d
}

func (d *dummyBftPeerCommunicator) Broadcast(msg BftMessage, peers []PeerInfo) {
	for _, peer := range peers {
		go func(peer PeerInfo) {
			ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "bft")
			//d.Peers[peer.MyIndex] <- msg
		}(peer)
	}
}

func (d *dummyBftPeerCommunicator) Unicast(msg BftMessage, peer PeerInfo) {
	go func() {
		ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "bft")
		//d.Peers[peer.MyIndex] <- msg
	}()
}

func (d *dummyBftPeerCommunicator) GetIncomingChannel() chan BftMessage {
	return d.messageProviderChannel
}

func (d *dummyBftPeerCommunicator) Run() {
	go func() {
		for {
			v := <-d.ReceiverChannel
			d.messageProviderChannel <- v
		}
	}()
}

type dummyProposalGenerator struct {
	CurrentHeight uint64
}

func (d dummyProposalGenerator) ProduceProposal() (proposal Proposal, validCondition ProposalCondition) {
	p := StringProposal("xxx")
	return &p, ProposalCondition{ValidHeight: d.CurrentHeight}
}

type dummyProposalValidator struct {
}

func (d dummyProposalValidator) ValidateProposal(proposal Proposal) error {
	return nil
}

type dummyDecisionMaker struct {
}

func (d dummyDecisionMaker) MakeDecision(proposal Proposal, state *HeightRoundState) (ConsensusDecision, error) {
	return proposal, nil
}
