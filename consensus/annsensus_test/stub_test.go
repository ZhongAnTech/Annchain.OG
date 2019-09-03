package annsensus_test

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"time"
)

type dummyAccountProvider struct {
	MyAccount *account.Account
}

func (d dummyAccountProvider) Account() *account.Account {
	return d.MyAccount
}

type dummySignatureProvider struct {
}

func (s dummySignatureProvider) Sign(data []byte) []byte {
	// no sign
	return data
}

type dummyContextProvider struct {
	NbParticipants int
	NbParts        int
	Threshold      int
	MyBftId        int
	BlockTime      time.Duration
	Suite          *bn256.Suite
	AllPartPubs    []dkg.PartPub
	MyPartSec      dkg.PartSec
}

func (d dummyContextProvider) GetNbParticipants() int {
	return d.NbParticipants
}

func (d dummyContextProvider) GetNbParts() int {
	return d.NbParts
}

func (d dummyContextProvider) GetThreshold() int {
	return d.Threshold
}

func (d dummyContextProvider) GetMyBftId() int {
	return d.MyBftId
}

func (d dummyContextProvider) GetBlockTime() time.Duration {
	return d.BlockTime
}

func (d dummyContextProvider) GetSuite() *bn256.Suite {
	return d.Suite
}

func (d dummyContextProvider) GetAllPartPubs() []dkg.PartPub {
	return d.AllPartPubs
}

func (d dummyContextProvider) GetMyPartSec() dkg.PartSec {
	return d.MyPartSec
}

type dummyPeerCommunicator struct {
}

func (d dummyPeerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	panic("implement me")
}

func (d dummyPeerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	panic("implement me")
}

func (d dummyPeerCommunicator) GetIncomingChannel() chan bft.BftMessage {
	panic("implement me")
}

func (d dummyPeerCommunicator) GetReceivingChannel() chan *message.OGMessage {
	panic("implement me")
}

func (d dummyPeerCommunicator) Run() {
	panic("implement me")
}

type dummyProposalGenerator struct {
}

func (d dummyProposalGenerator) ProduceProposal() (proposal bft.Proposal, validCondition bft.ProposalCondition) {
	panic("implement me")
}

type dummyProposalValidator struct {
}

func (d dummyProposalValidator) ValidateProposal(proposal bft.Proposal) error {
	panic("implement me")
}

type dummyDecisionMaker struct {
}

func (d dummyDecisionMaker) MakeDecision(proposal bft.Proposal, state *bft.HeightRoundState) (bft.ConsensusDecision, error) {
	panic("implement me")
}

type dummyTermProvider struct {
}

func (d dummyTermProvider) HeightTerm(height uint64) (termId uint32) {
	panic("implement me")
}

func (d dummyTermProvider) CurrentTerm() (termId uint32) {
	panic("implement me")
}

func (d dummyTermProvider) Peers(termId uint32) []bft.PeerInfo {
	panic("implement me")
}

func (d dummyTermProvider) GetTermChangeEventChannel() chan *term.Term {
	panic("implement me")
}
