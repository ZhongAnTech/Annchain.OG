package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/bft_test"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/og/message"
	"github.com/sirupsen/logrus"
	"sync"
)

type TermComposer struct {
	Term        *term.Term
	BftOperator bft.BftOperator
	DkgOperator dkg.DkgOperator
	quit        chan bool
	quitWg      sync.WaitGroup
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
	incomingChannel chan *message.OGMessage
	config          AnnsensusProcessorConfig

	myAccountProvider account.AccountProvider
	signatureProvider account.SignatureProvider
	contextProvider   ConsensusContextProvider
	peerCommunicator  bft.BftPeerCommunicator
	proposalGenerator bft.ProposalGenerator
	proposalValidator bft.ProposalValidator
	decisionMaker     bft.DecisionMaker
	termProvider      TermProvider

	// in case of disordered message, cache the terms and the correspondent processors.
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
	peerCommunicator bft.BftPeerCommunicator,
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
		incomingChannel:   make(chan *message.OGMessage),
		config:            config,
		myAccountProvider: myAccountProvider,
		signatureProvider: signatureProvider,
		contextProvider:   contextProvider,
		peerCommunicator:  peerCommunicator,
		proposalGenerator: proposalGenerator,
		proposalValidator: proposalValidator,
		decisionMaker:     decisionMaker,
		termProvider:      termProvider,
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
		case msg := <-ap.incomingChannel:
			ap.HandleConsensusMessage(msg)
		}
	}

}

func (ap *AnnsensusProcessor) StartNewTerm(termId uint32) error {
	// build a new Term
	// may need lots of information to build this term
	term := ap.buildTerm(termId)

	//build a reliable bft, dkg and term
	//bftComm := communicator.NewTrustfulPeerCommunicator(ap.signatureProvider, ap.termProvider, ap.p2pSender)

	dkgPartner, err := dkg.NewDkgPartner(
		ap.contextProvider.GetSuite(),
		termId,
		ap.contextProvider.GetNbParts(),
		ap.contextProvider.GetThreshold(),
		ap.contextProvider.GetAllPartPubs(),
		ap.contextProvider.GetMyPartSec())
	if err != nil {
		return err
	}

	tc := &TermComposer{
		Term: term,
		// TODO: check the parameters
		BftOperator: bft.NewDefaultBFTPartner(
			ap.contextProvider.GetNbParticipants(),
			ap.contextProvider.GetMyBftId(),
			ap.contextProvider.GetBlockTime(),
			ap.peerCommunicator,
			ap.proposalGenerator,
			ap.proposalValidator,
			ap.decisionMaker,
		),
		DkgOperator: dkgPartner,
	}
	ap.termMap[termId] = tc
	return nil
}

func (ap *AnnsensusProcessor) Stop() {
	ap.quitWg.Wait()
	logrus.Debug("AnnsensusProcessor stopped")
}

// HandleConsensusMessage is a sub-router for routing consensus message to either bft,dkg or term.
// As part of Annsensus, bft,dkg and term may not be regarded as a separate component of OG.
// Annsensus itself is also a plugin of OG supporting consensus messages.
// Do not block the pipe for any message processing. Router should not be blocked. Use channel.
func (ap *AnnsensusProcessor) HandleConsensusMessage(msg *message.OGMessage) {
	switch msg.MessageType {
	case message.OGMessageType(bft.BftMessageTypePreCommit):
		ap.BftOperator.GetPeerCommunicator().GetIncomingChannel()
	}
}

// buildTerm collects information from the info provider, to start a new term
func (ap *AnnsensusProcessor) buildTerm(u uint32) *term.Term {
	return nil
}
