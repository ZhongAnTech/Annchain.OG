package annsensus_test

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/sirupsen/logrus"
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
	Myid        int
	PeerPipeIns []chan bft.BftMessage
	pipeIn      chan bft.BftMessage
	pipeOut     chan bft.BftMessage
}

func (d *dummyBftPeerCommunicator) AdaptOgMessage(incomingMsg *message.OGMessage) (bft.BftMessage, error) {
	panic("implement me")
}

func (d *dummyBftPeerCommunicator) HandleIncomingMessage(msg bft.BftMessage) {
	d.pipeIn <- msg
}

func NewDummyBftPeerCommunicator(myid int, incoming chan bft.BftMessage,
	peers []chan bft.BftMessage) *dummyBftPeerCommunicator {
	d := &dummyBftPeerCommunicator{
		PeerPipeIns: peers,
		Myid:        myid,
		pipeIn:      incoming,
		pipeOut:     make(chan bft.BftMessage),
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
			//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
			d.PeerPipeIns[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyBftPeerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	go func() {
		//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
		d.PeerPipeIns[peer.Id] <- msg
	}()
}

func (d *dummyBftPeerCommunicator) GetPipeIn() chan bft.BftMessage {
	return d.pipeIn
}

func (d *dummyBftPeerCommunicator) GetPipeOut() chan bft.BftMessage {
	return d.pipeOut
}

func (d *dummyBftPeerCommunicator) Run() {
	logrus.Info("dummyBftPeerCommunicator running")
	go func() {
		for {
			v := <-d.pipeIn
			//vv := v.Message.(*bft.BftMessage)
			d.pipeOut <- v
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

func (d dummyTermProvider) Peers(termId uint32) ([]bft.PeerInfo, error) {
	panic("implement me")
}

func (d dummyTermProvider) GetTermChangeEventChannel() chan *term.Term {
	panic("implement me")
}

type dummyDkgPeerCommunicator struct {
	Myid    int
	Peers   []chan dkg.DkgMessage
	pipeIn  chan dkg.DkgMessage
	pipeOut chan dkg.DkgMessage
}

func NewDummyDkgPeerCommunicator(myid int, incoming chan dkg.DkgMessage, peers []chan dkg.DkgMessage) *dummyDkgPeerCommunicator {
	d := &dummyDkgPeerCommunicator{
		Peers:   peers,
		Myid:    myid,
		pipeIn:  incoming,
		pipeOut: make(chan dkg.DkgMessage, 10000), // must be big enough to avoid blocking issue
	}
	return d
}

func (d *dummyDkgPeerCommunicator) Broadcast(msg dkg.DkgMessage, peers []dkg.PeerInfo) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("me", d.Myid).Debug("broadcasting message")
		go func(peer dkg.PeerInfo) {
			ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "dkg")
			//d.Peers[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyDkgPeerCommunicator) Unicast(msg dkg.DkgMessage, peer dkg.PeerInfo) {
	go func(peerId int) {
		ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "dkg")
		//d.Peers[peerId] <- msg
	}(peer.Id)
}

func (d *dummyDkgPeerCommunicator) GetPipeOut() chan dkg.DkgMessage {
	return d.pipeOut
}

func (d *dummyDkgPeerCommunicator) Run() {
	logrus.Info("dummyDkgPeerCommunicator running")
	go func() {
		for {
			v := <-d.pipeIn
			ffchan.NewTimeoutSenderShort(d.pipeOut, v, "pc")
			//d.pipeOut <- v
		}
	}()
}
