package annsensus_test

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
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
	term      *term.Term
	MyBftId   int
	MyPartSec dkg.PartSec
	blockTime time.Duration
}

func (d dummyContextProvider) GetTerm() *term.Term {
	return d.term
}

func (d dummyContextProvider) GetMyBftId() int {
	return d.MyBftId
}

func (d dummyContextProvider) GetMyPartSec() dkg.PartSec {
	return d.MyPartSec
}

func (d dummyContextProvider) GetBlockTime() time.Duration {
	return d.blockTime
}

type dummyProposalGenerator struct {
	CurrentHeight uint64
}

func (d dummyProposalGenerator) ProduceProposal() (proposal bft.Proposal, validCondition bft.ProposalCondition) {
	currentTime := time.Now()
	p := bft.StringProposal{Content: currentTime.Format("2006-01-02 15:04:05")}
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
	termChangeEventChan chan annsensus.ConsensusContextProvider
}

func NewDummyTermProvider() *dummyTermProvider {
	return &dummyTermProvider{termChangeEventChan: make(chan annsensus.ConsensusContextProvider)}
}

func (d dummyTermProvider) HeightTerm(height uint64) (termId uint32) {
	// currently always return 0 as a genesis term.
	return 0
	// return uint32(height / 10)
}

func (d dummyTermProvider) CurrentTerm() (termId uint32) {
	panic("implement me")
}

func (d dummyTermProvider) Peers(termId uint32) ([]bft.PeerInfo, error) {
	panic("implement me")
}

func (d dummyTermProvider) GetTermChangeEventChannel() chan annsensus.ConsensusContextProvider {
	return d.termChangeEventChan
}

type dummyAnnsensusPartnerProvider struct {
	peerChans []chan *annsensus.AnnsensusMessageEvent
}

func NewDummyAnnsensusPartnerProivder(peerChans []chan *annsensus.AnnsensusMessageEvent) *dummyAnnsensusPartnerProvider {
	dapp := &dummyAnnsensusPartnerProvider{
		peerChans: peerChans,
	}
	return dapp
}

func (d *dummyAnnsensusPartnerProvider) GetDkgPartnerInstance(context annsensus.ConsensusContextProvider) (dkgPartner dkg.DkgPartner, err error) {
	myId := context.GetMyBftId()

	localAnnsensusPeerCommunicator := NewLocalAnnsensusPeerCommunicator(myId,
		d.peerChans[myId], d.peerChans)

	dkgMessageAdapter := &annsensus.PlainDkgAdapter{
		DkgMessageUnmarshaller: &annsensus.DkgMessageUnmarshaller{},
	}

	commuicatorDkg := annsensus.NewProxyDkgPeerCommunicator(dkgMessageAdapter, localAnnsensusPeerCommunicator)

	term := context.GetTerm()
	dkgPartner, err = dkg.NewDefaultDkgPartner(
		term.Suite,
		term.Id,
		term.PartsNum,
		term.Threshold,
		term.AllPartPublicKeys,
		context.GetMyPartSec(),
		commuicatorDkg,
		commuicatorDkg,
	)
	return

}

func (d *dummyAnnsensusPartnerProvider) GetBftPartnerInstance(context annsensus.ConsensusContextProvider) bft.BftPartner {
	myId := context.GetMyBftId()

	bftMessageAdapter := &annsensus.PlainBftAdapter{
		BftMessageUnmarshaller: &annsensus.BftMessageUnmarshaller{},
	}

	localAnnsensusPeerCommunicator := NewLocalAnnsensusPeerCommunicator(myId,
		d.peerChans[myId], d.peerChans)

	commuicatorBft := annsensus.NewProxyBftPeerCommunicator(bftMessageAdapter, localAnnsensusPeerCommunicator)

	currentTerm := context.GetTerm()

	peerInfos := annsensus.DkgToBft(currentTerm.AllPartPublicKeys)

	bftPartner := bft.NewDefaultBFTPartner(
		currentTerm.PartsNum,
		context.GetMyBftId(),
		context.GetBlockTime(),
		commuicatorBft,
		commuicatorBft,
		&dummyProposalGenerator{},
		&dummyProposalValidator{},
		&dummyDecisionMaker{},
		peerInfos,
	)
	return bftPartner
}

type LocalAnnsensusPeerCommunicator struct {
	Myid  int
	Peers []chan *annsensus.AnnsensusMessageEvent
	pipe  chan *annsensus.AnnsensusMessageEvent
}

func (d *LocalAnnsensusPeerCommunicator) Broadcast(msg annsensus.AnnsensusMessage, peers []annsensus.AnnsensusPeer) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("IM", d.Myid).
			WithField("msg", msg).Debug("local broadcasting annsensus message")
		go func(peer annsensus.AnnsensusPeer) {
			//ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "annsensus")
			d.Peers[peer.Id] <- &annsensus.AnnsensusMessageEvent{
				Message: msg,
				Peer:    peer,
			}
		}(peer)
	}
}

func (d *LocalAnnsensusPeerCommunicator) Unicast(msg annsensus.AnnsensusMessage, peer annsensus.AnnsensusPeer) {
	logrus.Debug("local unicasting by dummyBftPeerCommunicator")
	go func() {
		//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
		d.Peers[peer.Id] <- &annsensus.AnnsensusMessageEvent{
			Message: msg,
			Peer:    peer,
		}
	}()
}

func (d *LocalAnnsensusPeerCommunicator) GetPipeIn() chan *annsensus.AnnsensusMessageEvent {
	return d.pipe
}

func (d *LocalAnnsensusPeerCommunicator) GetPipeOut() chan *annsensus.AnnsensusMessageEvent {
	return d.pipe
}

func NewLocalAnnsensusPeerCommunicator(myid int, incoming chan *annsensus.AnnsensusMessageEvent,
	peers []chan *annsensus.AnnsensusMessageEvent) *LocalAnnsensusPeerCommunicator {
	d := &LocalAnnsensusPeerCommunicator{
		Myid:  myid,
		Peers: peers,
		pipe:  incoming,
	}
	return d
}
