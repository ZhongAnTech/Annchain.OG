package bft_test

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/ffchan"
)

type dummyBftPeerCommunicator struct {
	Myid                   int
	Peers                  []chan bft.BftMessage
	ReceiverChannel        chan bft.BftMessage
	messageProviderChannel chan bft.BftMessage
}

func (d *dummyBftPeerCommunicator) GetReceivingChannel() chan bft.BftMessage {
	return d.ReceiverChannel
}

func NewDummyBftPeerCommunicator(myid int, incoming chan bft.BftMessage, peers []chan bft.BftMessage) *dummyBftPeerCommunicator {
	d := &dummyBftPeerCommunicator{
		Peers:                  peers,
		Myid:                   myid,
		ReceiverChannel:        incoming,
		messageProviderChannel: make(chan bft.BftMessage),
	}
	return d
}

func (d *dummyBftPeerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	for _, peer := range peers {
		go func(peer bft.PeerInfo) {
			ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "bft")
			//d.Peers[peer.MyIndex] <- msg
		}(peer)
	}
}

func (d *dummyBftPeerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	go func() {
		ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "bft")
		//d.Peers[peer.MyIndex] <- msg
	}()
}

func (d *dummyBftPeerCommunicator) GetIncomingChannel() chan bft.BftMessage {
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

func (d dummyProposalGenerator) ProduceProposal() (proposal bft.Proposal, validCondition bft.ProposalCondition) {
	p := bft.StringProposal("xxx")
	return &p, bft.ProposalCondition{ValidHeight: d.CurrentHeight}
}

type dummyProposalValidator struct {
}

func (d dummyProposalValidator) ValidateProposal(proposal bft.Proposal) error {
	return nil
}

type dummyDecisionMaker struct {
}

func (d dummyDecisionMaker) MakeDecision(proposal bft.Proposal, state *bft.HeightRoundState) (bft.ConsensusDecision, error) {
	return proposal, nil
}
