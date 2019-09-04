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

type dummyBftPeerCommunicator struct {
	Myid                   int
	Peers                  []chan bft.BftMessage
	ReceiverChannel        chan bft.BftMessage
	messageProviderChannel chan bft.BftMessage
}

func (d *dummyBftPeerCommunicator) HandleIncomingMessage(msg bft.BftMessage) {
	d.ReceiverChannel <- msg
}

func (d *dummyBftPeerCommunicator) GetReceivingChannel() chan bft.BftMessage {
	return d.ReceiverChannel
}


func NewDummyBftPeerCommunicator(myid int, incoming chan bft.BftMessage,
	peers []chan bft.BftMessage) *dummyBftPeerCommunicator {
	d := &dummyBftPeerCommunicator{
		Peers:                  peers,
		Myid:                   myid,
		ReceiverChannel:        incoming,
		messageProviderChannel: make(chan bft.BftMessage),
	}
	return d
}

func  (d *dummyBftPeerCommunicator) wrapOGMessage(msg bft.BftMessage) *message.OGMessage{
	return &message.OGMessage{
		MessageType:    message.OGMessageType(msg.Type),
		Data:           nil,
		Hash:           nil,
		SourceID:       "",
		SendingType:    0,
		Version:        0,
		Message:        msg.Payload,
		SourceHash:     nil,
		MarshalState:   false,
		DisableEncrypt: false,
	}
}


func (d *dummyBftPeerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	for _, peer := range peers {
		go func(peer bft.PeerInfo) {
			//ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "bft")
			d.Peers[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyBftPeerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	go func() {
		//ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "bft")
		d.Peers[peer.Id] <- msg
	}()
}

func (d *dummyBftPeerCommunicator) GetIncomingChannel() chan bft.BftMessage {
	return d.messageProviderChannel
}

func (d *dummyBftPeerCommunicator) Run() {
	go func() {
		for {
			v := <-d.ReceiverChannel
			//vv := v.Message.(*bft.BftMessage)
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
