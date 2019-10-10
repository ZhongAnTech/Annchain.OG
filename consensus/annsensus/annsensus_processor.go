package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/og/communicator"
	"github.com/sirupsen/logrus"
	"sync"
)

type TermCollection struct {
	Term       *term.Term
	BftPartner bft.BftPartner
	DkgPartner dkg.DkgPartner
	quit       chan bool
	quitWg     sync.WaitGroup
}

func NewTermCollection(term *term.Term, bftPartner bft.BftPartner, dkgPartner dkg.DkgPartner) *TermCollection {
	return &TermCollection{
		Term:       term,
		BftPartner: bftPartner,
		DkgPartner: dkgPartner,
		quit:       make(chan bool),
		quitWg:     sync.WaitGroup{},
	}
}

func (tc *TermCollection) Start() {
	// start all operators for this term.
	tc.quitWg.Add(1)
loop:
	for {
		select {
		case <-tc.quit:
			tc.BftPartner.Stop()
			tc.DkgPartner.Stop()
			tc.quitWg.Done()
			break loop
		}
	}
}

func (tc *TermCollection) Stop() {
	close(tc.quit)
	tc.quitWg.Wait()
}

// AnnsensusProcessor integrates dkg, bft and term change with vrf.
type AnnsensusProcessor struct {
	config                AnnsensusProcessorConfig
	bftAdapter            BftMessageAdapter      // message handlers in common. Injected into commuinicator
	dkgAdapter            DkgMessageAdapter      // message handlers in common. Injected into commuinicator
	annsensusCommunicator *AnnsensusCommunicator // interface to the p2p
	termHolder            TermHolder             // hold information for each term
}

type AnnsensusProcessorConfig struct {
	DisableTermChange  bool
	DisabledConsensus  bool
	TermChangeInterval int
	GenesisAccounts    crypto.PublicKeys
	PartnerNum         int
}

func checkConfig(config AnnsensusProcessorConfig) {
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
}

func NewAnnsensusProcessor(config AnnsensusProcessorConfig,
	signatureProvider account.SignatureProvider,
	termProvider TermProvider,
	p2pSender communicator.P2PSender,
) *AnnsensusProcessor {
	// Prepare common facilities that will be reused during each term
	// Prepare adapters
	bftAdapter := NewTrustfulBftAdapter(signatureProvider, termProvider)
	dkgAdapter := NewDkgAdapter()
	termHolder := NewBftTermHolder(termProvider)
	// Prepare annsensus communicator
	annsensusCommunicator := NewAnnsensusCommunicator(p2pSender, bftAdapter, dkgAdapter, termHolder)

	// Prepare process itself.
	ap := &AnnsensusProcessor{
		config:                config,
		bftAdapter:            bftAdapter,
		dkgAdapter:            dkgAdapter,
		annsensusCommunicator: annsensusCommunicator,
		termHolder:            termHolder,
	}
	return ap
}

func (ap *AnnsensusProcessor) Start() {
	if ap.config.DisabledConsensus {
		log.Warn("annsensus disabled")
		return
	}
	go ap.annsensusCommunicator.Run()
	log.Info("AnnSensus Started")
}

// buildTerm collects information from the info provider, to start a new term
func (ap *AnnsensusProcessor) buildTerm(termId uint32) *term.Term {
	term := term.NewTerm(termId)
}

func (ap *AnnsensusProcessor) StartNewTerm(termId uint32, context ConsensusContextProvider) error {
	// build a new Term
	// may need lots of information to build this term
	newTerm := ap.buildTerm(termId)

	//build a reliable bft, dkg and term
	//bftComm := communicator.NewTrustfulPeerCommunicator(ap.signatureProvider, ap.termProvider, ap.p2pSender)
	tc := NewTermCollection(newTerm, bftPartner, dkgPartner)
	ap.termHolder.SetTerm(termId, tc)
	return nil
}

func (ap *AnnsensusProcessor) Stop() {
	ap.annsensusCommunicator.Stop()
	logrus.Debug("AnnsensusProcessor stopped")
}
