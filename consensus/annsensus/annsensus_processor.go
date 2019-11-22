package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/sirupsen/logrus"
	"sync"
)

type TermCollection struct {
	contextProvider ConsensusContextProvider
	BftPartner      bft.BftPartner
	DkgPartner      dkg.DkgPartner
	quit            chan bool
	quitWg          sync.WaitGroup
}

func NewTermCollection(
	contextProvider ConsensusContextProvider,
	bftPartner bft.BftPartner, dkgPartner dkg.DkgPartner) *TermCollection {
	return &TermCollection{
		contextProvider: contextProvider,
		BftPartner:      bftPartner,
		DkgPartner:      dkgPartner,
		quit:            make(chan bool),
		quitWg:          sync.WaitGroup{},
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
	bftAdapter            BftMessageAdapter          // message handlers in common. Injected into commuinicator
	dkgAdapter            DkgMessageAdapter          // message handlers in common. Injected into commuinicator
	annsensusCommunicator *AnnsensusPeerCommunicator // interface to the p2p
	termProvider          TermIdProvider
	termHolder            HistoricalTermsHolder // hold information for each term
	bftPartnerProvider    BftPartnerProvider    // factory method to generate a bft partner for each term
	dkgPartnerProvider    DkgPartnerProvider    // factory method to generate a dkg partner for each term

}

func NewAnnsensusProcessor(
	config AnnsensusProcessorConfig,
	bftAdapter BftMessageAdapter,
	dkgAdapter DkgMessageAdapter,
	annsensusCommunicator *AnnsensusPeerCommunicator,
	termProvider TermIdProvider,
	termHolder HistoricalTermsHolder,
	bftPartnerProvider BftPartnerProvider,
	dkgPartnerProvider DkgPartnerProvider) *AnnsensusProcessor {
	return &AnnsensusProcessor{config: config, bftAdapter: bftAdapter,
		dkgAdapter: dkgAdapter, annsensusCommunicator: annsensusCommunicator,
		termProvider: termProvider,
		termHolder:   termHolder, bftPartnerProvider: bftPartnerProvider,
		dkgPartnerProvider: dkgPartnerProvider}
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

//func NewAnnsensusProcessor(config AnnsensusProcessorConfig,
//	signatureProvider account.SignatureProvider,
//	termProvider TermIdProvider,
//	annsensusCommunicator *AnnsensusPeerCommunicator,
//) *AnnsensusProcessor {
//	// Prepare common facilities that will be reused during each term
//	// Prepare adapters
//	bftAdapter := NewTrustfulBftAdapter(signatureProvider, termProvider)
//	dkgAdapter := NewTrustfulDkgAdapter()
//	termHolder := NewAnnsensusTermHolder(termProvider)
//
//	// Prepare process itself.
//	ap := &AnnsensusProcessor{
//		config:                config,
//		bftAdapter:            bftAdapter,
//		dkgAdapter:            dkgAdapter,
//		annsensusCommunicator: annsensusCommunicator,
//		termHolder:            termHolder,
//	}
//	return ap
//}

// Start makes AnnsensusProcessor receive and handle consensus messages.
func (ap *AnnsensusProcessor) Start() {
	if ap.config.DisabledConsensus {
		log.Warn("annsensus disabled")
		return
	}
	// start the receiver
	go ap.annsensusCommunicator.Run()
	go ap.loop()

	log.Info("AnnSensus Started")
}

// buildTerm collects information from the info provider, to start a new term
//func (ap *AnnsensusProcessor) buildTerm(termId uint32) *term.Term {
//	//TODO
//	t := term.Term{
//		Id:                0,
//		PartsNum:          ap.config.PartnerNum,
//		Threshold:         ap.config.PartnerNum*2/3 + 1,
//		Senators:          nil,
//		AllPartPublicKeys: nil,
//		PublicKey:         nil,
//		ActivateHeight:    0,
//		Suite:             nil,
//	}
//	return t
//}

func (ap *AnnsensusProcessor) StartNewTerm(context ConsensusContextProvider) error { // build a new Term
	// may need lots of information to build this term
	//newTerm := ap.buildTerm(termId)

	//build a reliable bft, dkg and term
	//bftComm := communicator.NewTrustfulPeerCommunicator(ap.signatureProvider, ap.termProvider, ap.p2pSender)

	logrus.WithField("context", context).Debug("starting new term")
	// the bft instance for this term
	bftPartner := ap.bftPartnerProvider.GetBftPartnerInstance(context)
	// the dkg instance for this term to generate next term
	dkgPartner, err := ap.dkgPartnerProvider.GetDkgPartnerInstance(context)
	if err != nil {
		return err
	}

	tc := NewTermCollection(context, bftPartner, dkgPartner)
	ap.termHolder.SetTerm(context.GetTerm().Id, tc)
	// start to generate proposals, vote and generate sequencers
	go bftPartner.Start()
	bftPartner.StartNewEra(0, 0)
	// start to discuss next committee
	go dkgPartner.Start()
	return nil
}

func (ap *AnnsensusProcessor) Stop() {
	ap.annsensusCommunicator.Stop()
	logrus.Debug("AnnsensusProcessor stopped")
}

func (ap *AnnsensusProcessor) loop() {
	for {
		select {
		case context := <-ap.termProvider.GetTermChangeEventChannel():
			// new term is coming.
			// it is coming because of either:
			// 1, someone told you that you are keeping up-to-date with latest seq
			// 2, (just for debugging) bootstrap
			// Note that in real case, bootstrap will not trigger a direct term change event
			// Since there must be a communication first to get the latest seq.
			err := ap.StartNewTerm(context)
			if err != nil {
				logrus.WithError(err).WithField("context", context).Error("failed to start a new term")
			}
			//
		}
	}
}
