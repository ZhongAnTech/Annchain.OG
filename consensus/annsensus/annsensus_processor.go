package annsensus

import (
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/sirupsen/logrus"
	"sync"
)

// AnnsensusPartner integrates dkg, bft and term change with vrf.
type AnnsensusPartner struct {
	Config                   AnnsensusProcessorConfig
	BftAdapter               BftMessageAdapter // message handlers in common. Injected into commuinicator
	DkgAdapter               DkgMessageAdapter // message handlers in common. Injected into commuinicator
	TermProvider             TermIdProvider
	TermHolder               HistoricalTermsHolder // hold information for each term
	BftPartnerProvider       BftPartnerProvider    // factory method to generate a bft partner for each term
	DkgPartnerProvider       DkgPartnerProvider    // factory method to generate a dkg partner for each term
	ConsensusContextProvider ConsensusContextProvider
	PeerOutgoing             AnnsensusPeerCommunicatorOutgoing
	PeerIncoming             AnnsensusPeerCommunicatorIncoming

	quit   chan bool
	quitWg sync.WaitGroup
}

func (a *AnnsensusPartner) InitDefault() {
	a.quitWg = sync.WaitGroup{}
	a.quit = make(chan bool)
}

type AnnsensusProcessorConfig struct {
	DisableTermChange  bool
	DisabledConsensus  bool
	TermChangeInterval int
	GenesisAccounts    crypto.PublicKeys
	//PartnerNum         int
}

func checkConfig(config AnnsensusProcessorConfig) {
	if config.DisabledConsensus {
		config.DisableTermChange = true
	}
	if !config.DisabledConsensus {
		if config.TermChangeInterval <= 0 && !config.DisableTermChange {
			panic("require termChangeInterval")
		}
		//if len(config.GenesisAccounts) < config.PartnerNum && !config.DisableTermChange {
		//	panic("need more account")
		//}
		//if config.PartnerNum < 2 {
		//	panic(fmt.Sprintf("BFT needs at least 2 nodes, currently %d", config.PartnerNum))
		//}
	}
}

//func NewAnnsensusPartner(Config AnnsensusProcessorConfig,
//	signatureProvider account.SignatureProvider,
//	TermProvider TermIdProvider,
//	annsensusCommunicator *AnnsensusPeerCommunicator,
//) *AnnsensusPartner {
//	// Prepare common facilities that will be reused during each term
//	// Prepare adapters
//	BftAdapter := NewTrustfulBftAdapter(signatureProvider, TermProvider)
//	DkgAdapter := NewTrustfulDkgAdapter()
//	TermHolder := NewAnnsensusTermHolder(TermProvider)
//
//	// Prepare process itself.
//	ap := &AnnsensusPartner{
//		Config:                Config,
//		BftAdapter:            BftAdapter,
//		DkgAdapter:            DkgAdapter,
//		annsensusCommunicator: annsensusCommunicator,
//		TermHolder:            TermHolder,
//	}
//	return ap
//}

// Start makes AnnsensusPartner receive and handle consensus messages.
func (ap *AnnsensusPartner) Start() {
	if ap.Config.DisabledConsensus {
		log.Warn("annsensus disabled")
		return
	}
	// start the receiver
	go ap.loop()
	log.Info("AnnSensus Started")
}

// buildTerm collects information from the info provider, to start a new term
//func (ap *AnnsensusPartner) buildTerm(termId uint32) *term.Term {
//	//TODO
//	t := term.Term{
//		Id:                0,
//		PartsNum:          ap.Config.PartnerNum,
//		Threshold:         ap.Config.PartnerNum*2/3 + 1,
//		Senators:          nil,
//		AllPartPublicKeys: nil,
//		PublicKey:         nil,
//		ActivateHeight:    0,
//		Suite:             nil,
//	}
//	return t
//}

func (ap *AnnsensusPartner) StartNewTerm(context ConsensusContext) error { // build a new Term
	// may need lots of information to build this term
	//newTerm := ap.buildTerm(termId)

	//build a reliable bft, dkg and term
	//bftComm := communicator.NewTrustfulPeerCommunicator(ap.signatureProvider, ap.TermProvider, ap.p2pSender)

	// check if the term is already there
	if _, ok := ap.TermHolder.GetTermById(context.GetTerm().Id); ok {
		return errors.New("term already there")
	}

	logrus.WithField("context", context).Debug("starting new term")
	// the bft instance for this term
	bftPartner := ap.BftPartnerProvider.GetBftPartnerInstance(context)
	// the dkg instance for this term to generate next term
	dkgPartner, err := ap.DkgPartnerProvider.GetDkgPartnerInstance(context)
	if err != nil {
		return err
	}

	tc := NewTermCollection(context, bftPartner, dkgPartner)
	ap.TermHolder.SetTerm(context.GetTerm().Id, tc)
	// start to generate proposals, vote and generate sequencers
	go bftPartner.Start()
	bftPartner.StartNewEra(context.GetTerm().ActivateHeight, 0)
	// start to discuss next committee
	go dkgPartner.Start()
	return nil
}

func (ap *AnnsensusPartner) Stop() {
	ap.quit <- true
	ap.quitWg.Wait()
	logrus.Debug("AnnsensusPartner stopped")
}

func (ap *AnnsensusPartner) loop() {
	for {
		select {
		case <-ap.quit:
			ap.quitWg.Done()
			logrus.Debug("AnnsensusPartner quit")
			return
		case context := <-ap.TermProvider.GetTermChangeEventChannel():
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
		case msgEvent := <-ap.PeerIncoming.GetPipeOut():
			ap.HandleAnnsensusMessage(msgEvent)
		}
	}
}

// HandleConsensusMessage is a sub-router for routing consensus message to either bft,dkg or term.
// As part of Annsensus, bft,dkg and term may not be regarded as a separate component of OG.
// Annsensus itself is also a plugin of OG supporting consensus messages.
// Do not block the pipe for any message processing. Router should not be blocked. Use channel.
func (ap *AnnsensusPartner) HandleAnnsensusMessage(msgEvent *AnnsensusMessageEvent) {
	annsensusMessage := msgEvent.Message
	peer := msgEvent.Peer

	switch annsensusMessage.GetType() {
	case AnnsensusMessageTypeBftPlain:
		// let adapter handle plain and signed
		fallthrough
	case AnnsensusMessageTypeBftSigned:
		bftMessage, err := ap.BftAdapter.AdaptAnnsensusMessage(annsensusMessage)
		if err != nil {
			logrus.WithError(err).WithField("msg", annsensusMessage).
				Error("error adapting annsensus to bft msg")
			return
		}
		bftPeer, err := ap.BftAdapter.AdaptAnnsensusPeer(peer)
		if err != nil {
			logrus.WithError(err).WithField("peer", peer).
				Error("error adapting annsensus to bft peer")
			return
		}
		// judge height
		msgTerm, ok := ap.TermHolder.GetTermByHeight(bftMessage)
		if !ok {
			logrus.WithError(err).WithField("msg", bftMessage).
				Error("error finding or building msgTerm")
			return
		}
		// route to correspondant BFTPartner
		msgTerm.BftPartner.GetBftPeerCommunicatorIncoming().GetPipeIn() <- &bft.BftMessageEvent{
			Message: bftMessage,
			Peer:    bftPeer,
		}

	case AnnsensusMessageTypeDkgPlain:
		fallthrough
	case AnnsensusMessageTypeDkgSigned:
		dkgMessage, err := ap.DkgAdapter.AdaptAnnsensusMessage(annsensusMessage)
		if err != nil {
			logrus.WithError(err).WithField("msg", annsensusMessage).
				Error("error adapting annsensus to dkg msg")
			return
		}
		dkgPeer, err := ap.DkgAdapter.AdaptAnnsensusPeer(peer)
		if err != nil {
			logrus.WithError(err).WithField("peer", peer).
				Error("error adapting annsensus to dkg peer")
			return
		}
		msgTerm, ok := ap.TermHolder.GetTermByHeight(dkgMessage)
		if !ok {
			logrus.WithError(err).WithField("msg", dkgMessage).
				Error("error finding msgTerm")
			return
		}
		msgTerm.DkgPartner.GetDkgPeerCommunicatorIncoming().GetPipeIn() <- &dkg.DkgMessageEvent{
			Message: dkgMessage,
			Peer:    dkgPeer,
		}

	case AnnsensusMessageTypeBftEncrypted:
		// decrypt first and then send to handler
	case AnnsensusMessageTypeDkgEncrypted:
		// decrypt first and then send to handler
	default:
		logrus.WithField("msg", annsensusMessage).WithField("IM", ap.TermHolder.DebugMyId()).Warn("unsupported annsensus message type")
	}
	return
}
