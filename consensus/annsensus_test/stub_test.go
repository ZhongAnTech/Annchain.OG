package annsensus_test

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/types/msg"
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

type dummyBftPeerCommunicator struct {
	Myid        int
	PeerPipeIns []chan bft.BftMessage
	pipeIn      chan bft.BftMessage
	pipeOut     chan bft.BftMessage
}

func (d *dummyBftPeerCommunicator) Adapttypes(incomingMsg msg.TransportableMessage) (bft.BftMessage, error) {
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

func (d *dummyBftPeerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	logrus.Debug("broadcasting by dummyBftPeerCommunicator")
	for _, peer := range peers {
		go func(peer bft.PeerInfo) {
			//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
			d.PeerPipeIns[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyBftPeerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	logrus.Debug("unicasting by dummyBftPeerCommunicator")
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
			//vv := v.Message.(bft.BftMessage)
			logrus.WithField("type", v.GetType()).Debug("dummyBftPeerCommunicator received a message")
			d.pipeOut <- v
		}
	}()
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

func (d *dummyDkgPeerCommunicator) GetPipeIn() chan dkg.DkgMessage {
	return d.pipeIn
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

type dummyAnnsensusPartnerProvider struct {
	peerChansBft []chan bft.BftMessage
	peerChansDkg []chan dkg.DkgMessage
}

func NewDummyAnnsensusPartnerProivder(peerChansBft []chan bft.BftMessage, peerChansDkg []chan dkg.DkgMessage) *dummyAnnsensusPartnerProvider {
	dapp := &dummyAnnsensusPartnerProvider{
		peerChansBft: peerChansBft,
		peerChansDkg: peerChansDkg,
	}
	return dapp
}

func (d *dummyAnnsensusPartnerProvider) GetDkgPartnerInstance(context annsensus.ConsensusContextProvider) (dkgPartner dkg.DkgPartner, err error) {
	myId := context.GetMyBftId()
	communicatorDkg := NewDummyDkgPeerCommunicator(myId, d.peerChansDkg[myId], d.peerChansDkg)
	term := context.GetTerm()
	dkgPartner, err = dkg.NewDefaultDkgPartner(
		term.Suite,
		term.Id,
		term.PartsNum,
		term.Threshold,
		term.AllPartPublicKeys,
		context.GetMyPartSec(),
		communicatorDkg,
		communicatorDkg,
	)
	return

}

func (d *dummyAnnsensusPartnerProvider) GetBftPartnerInstance(context annsensus.ConsensusContextProvider) bft.BftPartner {
	myId := context.GetMyBftId()
	commuicatorBft := NewDummyBftPeerCommunicator(myId, d.peerChansBft[myId], d.peerChansBft)

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

type dummyAnnsensusPeerCommunicator struct {
	Myid  int
	Peers []chan annsensus.AnnsensusMessage
	pipe  chan annsensus.AnnsensusMessage
}

func (d *dummyAnnsensusPeerCommunicator) Broadcast(msg annsensus.AnnsensusMessage, peers []annsensus.AnnsensusPeer) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("me", d.Myid).Debug("broadcasting annsensus message")
		go func(peer annsensus.AnnsensusPeer) {
			ffchan.NewTimeoutSenderShort(d.Peers[peer.Id], msg, "annsensus")
			//d.Peers[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyAnnsensusPeerCommunicator) Unicast(msg annsensus.AnnsensusMessage, peer annsensus.AnnsensusPeer) {
	logrus.Debug("unicasting by dummyBftPeerCommunicator")
	go func() {
		//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
		d.Peers[peer.Id] <- msg
	}()
}

func (d *dummyAnnsensusPeerCommunicator) GetPipeIn() chan annsensus.AnnsensusMessage {
	return d.pipe
}

func (d *dummyAnnsensusPeerCommunicator) GetPipeOut() chan annsensus.AnnsensusMessage {
	return d.pipe
}

func NewDummyAnnsensusPeerCommunicator(myid int, incoming chan annsensus.AnnsensusMessage, peers []chan annsensus.AnnsensusMessage) *dummyAnnsensusPeerCommunicator {
	d := &dummyAnnsensusPeerCommunicator{
		Myid:  myid,
		Peers: peers,
		pipe:  incoming,
	}
	return d
}
