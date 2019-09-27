package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/message"
	"github.com/sirupsen/logrus"
	"sync"
)

type TermComposer struct {
	Term                   *term.Term
	BftOperator            bft.BftOperator
	DkgOperator            dkg.DkgOperator
	bftCommunicatorAdapter BftMessageAdapter
	dkgCommunicatorAdapter DkgMessageAdapter
	quit                   chan bool
	quitWg                 sync.WaitGroup
}

func NewTermComposer(term *term.Term, bftOperator bft.BftOperator, dkgOperator dkg.DkgOperator) *TermComposer {
	return &TermComposer{
		Term:        term,
		BftOperator: bftOperator,
		DkgOperator: dkgOperator,
		quit:        make(chan bool),
		quitWg:      sync.WaitGroup{},
	}
}

func (tc *TermComposer) Start() {
	// start all operators for this term.
	tc.quitWg.Add(1)
loop:
	for {
		select {
		case <-tc.quit:
			tc.BftOperator.Stop()
			tc.DkgOperator.Stop()
			tc.quitWg.Done()
			break loop
		}
	}
}

func (tc *TermComposer) Stop() {
	close(tc.quit)
	tc.quitWg.Wait()
}

// AnnsensusProcessor integrates dkg, bft and term change with vrf.
// It receives messages from
type AnnsensusProcessor struct {
	pipeIn chan *message.OGMessage
	config AnnsensusProcessorConfig

	myAccountProvider account.AccountProvider
	contextProvider   ConsensusContextProvider

	proposalGenerator bft.ProposalGenerator
	proposalValidator bft.ProposalValidator
	decisionMaker     bft.DecisionMaker
	termProvider      TermProvider

	// message handlers
	bftAdapter BftMessageAdapter
	dkgAdapter DkgMessageAdapter

	// in case of disordered message, cache the terms and the correspondent processors.
	// TODO: wipe it constantly
	termMap map[uint32]*TermComposer

	quit   chan bool
	quitWg sync.WaitGroup
}

type AnnsensusProcessorConfig struct {
	DisableTermChange  bool
	DisabledConsensus  bool
	TermChangeInterval int
	GenesisAccounts    crypto.PublicKeys
	PartnerNum         int
}

func NewAnnsensusProcessor(config AnnsensusProcessorConfig,
	myAccountProvider account.AccountProvider,
	signatureProvider account.SignatureProvider,
	contextProvider ConsensusContextProvider,
	proposalGenerator bft.ProposalGenerator,
	proposalValidator bft.ProposalValidator,
	decisionMaker bft.DecisionMaker,
	termProvider TermProvider) *AnnsensusProcessor {
	if config.DisabledConsensus {
		config.DisableTermChange = true
	}
	if !config.DisabledConsensus {
		if config.TermChangeInterval <= 0 && !config.DisableTermChange {
			panic("require termChangeInterval")
		}
		if len(config.GenesisAccounts) < config.PartnerNum && !config.DisableTermChange {
			panic("need more account")
		}
		if config.PartnerNum < 2 {
			panic(fmt.Sprintf("BFT needs at least 2 nodes, currently %d", config.PartnerNum))
		}
	}
	ap := &AnnsensusProcessor{
		pipeIn:            make(chan *message.OGMessage),
		config:            config,
		myAccountProvider: myAccountProvider,
		contextProvider:   contextProvider,
		proposalGenerator: proposalGenerator,
		proposalValidator: proposalValidator,
		decisionMaker:     decisionMaker,
		termProvider:      termProvider,
		bftAdapter:        NewTrustfulBftAdapter(signatureProvider, termProvider),
		dkgAdapter:        nil,
		termMap:           make(map[uint32]*TermComposer),
		quit:              make(chan bool),
		quitWg:            sync.WaitGroup{},
	}
	return ap
}

func (ap *AnnsensusProcessor) Start() {
	if ap.config.DisabledConsensus {
		log.Warn("annsensus disabled")
		return
	}
	ap.quitWg.Add(1)

	log.Info("AnnSensus Started")

loop:
	for {
		select {
		case <-ap.quit:
			ap.quitWg.Done()
			break loop
		case msg := <-ap.pipeIn:
			ap.HandleConsensusMessage(msg)
		}
	}
}

// buildTerm collects information from the info provider, to start a new term
func (ap *AnnsensusProcessor) buildTerm(u uint32) *term.Term {
	return nil
}

func (ap *AnnsensusProcessor) StartNewTerm(termId uint32) error {
	// build a new Term
	// may need lots of information to build this term
	newTerm := ap.buildTerm(termId)

	//build a reliable bft, dkg and term
	//bftComm := communicator.NewTrustfulPeerCommunicator(ap.signatureProvider, ap.termProvider, ap.p2pSender)
	dkgPartner, err := dkg.NewDkgPartner(
		ap.contextProvider.GetSuite(),
		termId,
		ap.contextProvider.GetNbParts(),
		ap.contextProvider.GetThreshold(),
		ap.contextProvider.GetAllPartPubs(),
		ap.contextProvider.GetMyPartSec(),
		ap.dkgPeerCommunicator,
		ap.dkgPeerCommunicator)
	if err != nil {
		return err
	}

	bftPartner := bft.NewDefaultBFTPartner(
		ap.contextProvider.GetNbParticipants(),
		ap.contextProvider.GetMyBftId(),
		ap.contextProvider.GetBlockTime(),
		ap.bftPeerCommunicator,
		ap.proposalGenerator,
		ap.proposalValidator,
		ap.decisionMaker,
	)
	tc := NewTermComposer(newTerm, bftPartner, dkgPartner)
	ap.termMap[termId] = tc
	return nil
}

func (ap *AnnsensusProcessor) Stop() {
	ap.quitWg.Wait()
	logrus.Debug("AnnsensusProcessor stopped")
}

// according to the height, get term, and send to the bft operator in that term.
func (ap *AnnsensusProcessor) handleBftMessage(ogMessage *message.OGMessage) {
	height := ogMessage.Message.(*bft.BftBasicInfo).HeightRound.Height
	// judge term
	termId := ap.termProvider.HeightTerm(height)
	msgTerm, ok := ap.termMap[termId]
	if !ok {
		logrus.Warn("term not found while handling bft message")
		return
	}
	// message adapter
	bftMessage, err := ap.bftAdapter.AdaptOgMessage(ogMessage.Message)
	if err != nil {
		logrus.WithError(err).Warn("cannot adapt ogMessage to bftMessage")
		return
	}

	msgTerm.BftOperator.GetBftPeerCommunicatorIncoming().GetPipeOut() <- bftMessage
}
