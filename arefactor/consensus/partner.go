package consensus

import (
	"fmt"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/goffchan"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

/**
Implemented according to
HotStuff: BFT Consensus in the Lens of Blockchain
Maofan Yin, Dahlia Malkhi, Michael K. Reiter, Guy Golan Gueta and Ittai Abraham
*/

type Partner struct {
	//BlockTree        *BlockTree
	Logger   *logrus.Logger
	Reporter *soccerdash.Reporter

	ProposalGenerator        consensus_interface.ProposalGenerator
	ProposalVerifier         consensus_interface.ProposalVerifier
	CommitteeProvider        consensus_interface.CommitteeProvider
	ConsensusSigner          consensus_interface.ConsensusSigner
	ConsensusAccountProvider consensus_interface.ConsensusAccountProvider
	//AccountProvider         og_interface.LedgerAccountProvider
	Hasher consensus_interface.Hasher
	Ledger consensus_interface.Ledger

	safety                  *Safety
	pendingBlockTree        *PendingBlockTree
	paceMaker               *PaceMaker
	proposalContextProvider consensus_interface.ProposalContextProvider
	proposalExecutor        consensus_interface.ProposalExecutor
	pendingQCs              map[string]consensus_interface.SignatureCollector // collected votes per block indexed by their LedgerInfo hash

	// event handlers
	myNewIncomingMessageEventChan chan *transport_interface.IncomingLetter
	newOutgoingMessageSubscribers []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent

	quit           chan bool
	BlockTime      time.Duration
	TimeoutTime    time.Duration
	timeoutChecker *time.Ticker
}

func (n *Partner) InitDefault() {
	n.quit = make(chan bool)
	n.safety = &Safety{
		Ledger:   n.Ledger,
		Reporter: n.Reporter,
		Logger:   n.Logger,
		Hasher:   n.Hasher,
	}
	n.safety.InitDefault()

	// for each hash, init a SignatureCollector
	n.pendingBlockTree = &PendingBlockTree{
		Logger: n.Logger,
		cache:  nil,
		Ledger: n.Ledger,
		Safety: n.safety,
	}
	n.pendingBlockTree.InitDefault()

	n.paceMaker = &PaceMaker{
		Logger:            n.Logger,
		CurrentRound:      n.Ledger.GetConsensusState().LastVoteRound,
		Safety:            n.safety,
		Partner:           n,
		ConsensusSigner:   n.ConsensusSigner,
		AccountProvider:   n.ConsensusAccountProvider,
		CommitteeProvider: n.CommitteeProvider,
		Reporter:          n.Reporter,
		BlockTime:         n.BlockTime,
	}
	n.paceMaker.InitDefault()

	n.proposalExecutor = n.pendingBlockTree
	n.proposalContextProvider = &DefaultProposalContextProvider{
		PaceMaker:        n.paceMaker,
		PendingBlockTree: n.pendingBlockTree,
		Safety:           n.safety,
	}
	n.pendingQCs = make(map[string]consensus_interface.SignatureCollector)
	n.myNewIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter)
	n.newOutgoingMessageSubscribers = []transport_interface.NewOutgoingMessageEventSubscriber{}
}
func (n *Partner) Start() {
	n.timeoutChecker = time.NewTicker(n.TimeoutTime)
	for {
		logrus.Trace("partner loop round start")
		select {
		case <-n.quit:
			return
		case msg := <-n.myNewIncomingMessageEventChan:
			n.handleIncomingMessage(msg)

			//n.Logger.WithField("msgType", msg.Typev.HotStuffMessageTypeString()).WithField("msgc", msg).Info("received message")
			//if ok := n.signatureOk(msg); !ok {
			//	fmt.Println(msg)
			//	panic("signature invalid")
			//	//continue
			//}

		case <-n.timeoutChecker.C:
			// check if the last proposal was received some times ago.
			if n.paceMaker.lastValidTime.Before(time.Now().Add(-n.TimeoutTime)) {
				// timeout
				logrus.WithField("round", n.paceMaker.CurrentRound).Warn("paceMaker timeout")
				n.paceMaker.LocalTimeoutRound()
				// update timeout
			}
		}
		consensusState := n.safety.ConsensusState()
		n.Reporter.Report("lastTC", consensusState.LastTC, false)
		n.Reporter.Report("CurrentRound", n.paceMaker.CurrentRound, false)
		n.Reporter.Report("HighQC", consensusState.HighQC.VoteData, false)

		logrus.Trace("partner loop round end")
	}
}

func (n *Partner) Stop() {
	close(n.quit)
}

func (n *Partner) Name() string {
	return fmt.Sprintf("Node %d", n.CommitteeProvider.GetMyPeerIndex())
}

func (n *Partner) ProcessProposalMessage(msg *consensus_interface.HotStuffSignedMessage) {

	p := &consensus_interface.ContentProposal{}
	err := p.FromBytes(msg.ContentBytes)
	if err != nil {
		logrus.WithError(err).Warn("failed to decode ContentProposal")
		return
	}
	logrus.WithFields(logrus.Fields{
		"proposal": msg.String(),
		"from":     msg.SenderMemberId,
	}).Debug("received proposal")

	n.ProcessCertificates(p.Proposal.ParentQC, p.TC, "ProposalM")

	currentRound := n.paceMaker.CurrentRound

	if p.Proposal.Round != currentRound {
		n.Logger.WithField("proposalRound", p.Proposal.Round).WithField("currentRound", currentRound).Warn("current round not match.")
		return
	}

	if msg.SenderMemberId != n.CommitteeProvider.GetLeader(currentRound).MemberId {
		n.Logger.WithField("msg.SenderMemberId", msg.SenderMemberId).
			WithField("current leader", n.CommitteeProvider.GetLeader(currentRound).MemberId).
			Warn("current leader not match.")
		return
	}
	// verify proposal
	// TODO: now sync. change to async in the future
	verifyResult := n.ProposalVerifier.VerifyProposal(p)
	if !verifyResult.Ok {
		logrus.Warn("proposal verification failed")
		return
	}

	// execute the block
	// TODO: execute the block async
	//n.BlockTree.ExecuteAndInsert(&p.HotStuffMessageTypeProposal)
	// TODO: who is proposalExecutor?
	executionResult := n.proposalExecutor.ExecuteProposal(&p.Proposal)
	if executionResult.Err != nil {
		// TODO: send clearly reject message instead of not sending messages
		return
	}
	//if executionResult.ExecuteStateId != p.Proposal.Payload

	// vote after execution

	voteMsg := n.safety.MakeVote(p.Proposal.Id, p.Proposal.Round, p.Proposal.ParentQC)
	if voteMsg != nil {
		bytes := voteMsg.ToBytes()
		voteAggregator := n.CommitteeProvider.GetLeader(currentRound + 1)

		signature, err := n.sign(voteMsg)
		if err != nil {
			return
		}

		outMsg := &consensus_interface.HotStuffSignedMessage{
			HotStuffMessageType: int(consensus_interface.HotStuffMessageTypeVote),
			ContentBytes:        bytes,
			SenderMemberId:      n.CommitteeProvider.GetMyPeerId(),
			Signature:           signature,
		}
		letter := &transport_interface.OutgoingLetter{
			ExceptMyself:   false, // in case the next proposal generator is me.
			Msg:            outMsg,
			SendType:       transport_interface.SendTypeUnicast,
			CloseAfterSent: false,
			EndReceivers:   []string{voteAggregator.TransportPeerId},
		}

		n.notifyNewOutgoingMessage(letter)
		n.paceMaker.RefreshTimeout()
	} else {
		logrus.WithField("proposal", p.Proposal).Warn("I don't vote this proposal.")
	}

}

func (n *Partner) ProcessVoteMessage(msg *consensus_interface.HotStuffSignedMessage) {
	p := &consensus_interface.ContentVote{}
	err := p.FromBytes(msg.ContentBytes)
	if err != nil {
		logrus.WithError(err).Debug("failed to decode ContentVote")
		return
	}
	n.ProcessCertificates(p.QC, p.TC, "Vote")
	n.ProcessVote(p, msg.Signature, msg.SenderMemberId)
}

func (n *Partner) ProcessVote(vote *consensus_interface.ContentVote, signature consensus_interface.Signature, fromId string) {
	id, err := n.CommitteeProvider.GetPeerIndex(fromId)
	if err != nil {
		logrus.WithError(err).WithField("peerId", fromId).
			Fatal("error in finding peer in committee")
	}

	voteIndex := n.Hasher.Hash(vote.LedgerCommitInfo.GetHashContent())
	logrus.WithFields(logrus.Fields{
		"from":      fromId,
		"voteIndex": voteIndex,
	}).Info("handling vote")

	collector := n.ensureQCCollector(voteIndex)
	collector.Collect(signature, id)

	logrus.WithField("sigs", collector.GetCurrentCount()).
		WithField("sig", signature).Debug("signature got one")
	n.Reporter.Report("qcsig", fmt.Sprintf("B %s %d J %s", voteIndex, collector.GetCurrentCount(), collector.GetJointSignature()), false)

	if collector.Collected() {
		n.Logger.WithField("vote", vote).Info("votes collected")
		qc := &consensus_interface.QC{
			VoteData:       vote.VoteInfo, // TODO: check if the voteinfo is aligned
			JointSignature: collector.GetJointSignature(),
		}

		n.pendingBlockTree.EnsureHighQC(qc)
		n.paceMaker.AdvanceRound(qc, nil, "vote qc got")

	} else {
		n.Logger.WithField("vote", vote).
			WithField("now", collector.GetCurrentCount()).Trace("votes yet collected")
	}
}

func (n *Partner) ProcessCertificates(qc *consensus_interface.QC, tc *consensus_interface.TC, reason string) {
	n.paceMaker.AdvanceRound(qc, tc, reason+"ProcessCertificates"+strconv.FormatInt(n.paceMaker.CurrentRound, 10))
	if qc != nil {
		n.safety.UpdatePreferredRound(qc)
		if qc.VoteData.ExecStateId != "" {
			n.pendingBlockTree.Commit(qc.VoteData.Id)
		}
	}
}

func (n *Partner) ProposalGeneratedEventHandler(proposal *consensus_interface.ContentProposal) {
	if proposal == nil {
		logrus.Warn("No proposal could be made. abandon")
		return
	}
	n.Logger.WithField("proposal", proposal).Warn("I'm the current leader")
	n.Reporter.Report("leader", proposal.Proposal.Round, false)

	bytes := proposal.ToBytes()
	signature, err := n.sign(proposal)
	if err != nil {
		return
	}

	// announce it
	outMsg := &consensus_interface.HotStuffSignedMessage{
		HotStuffMessageType: int(consensus_interface.HotStuffMessageTypeProposal),
		ContentBytes:        bytes,
		SenderMemberId:      n.CommitteeProvider.GetMyPeerId(),
		Signature:           signature,
	}
	letter := &transport_interface.OutgoingLetter{
		ExceptMyself:   false, // announce to me also so that I can process that proposal.
		Msg:            outMsg,
		SendType:       transport_interface.SendTypeMulticast,
		CloseAfterSent: false,
		EndReceivers:   n.CommitteeProvider.GetAllMemberTransportIds(),
	}
	n.notifyNewOutgoingMessage(letter)
}

func (n *Partner) ProcessNewRoundEvent() {
	if !n.CommitteeProvider.AmILeader(n.paceMaker.CurrentRound) {
		// not the leader
		n.Logger.Trace("I'm not the leader so just return")
		return
	}
	//proposal := n.BlockTree.GenerateProposal(n.paceMaker.CurrentRound, strconv.Itoa(RandInt()))
	proposalContext := n.proposalContextProvider.GetProposalContext()
	n.ProposalGenerator.GenerateProposalAsync(proposalContext, n.ProposalGeneratedEventHandler)
}

func (n *Partner) handleIncomingMessage(msg *transport_interface.IncomingLetter) {
	if msg.Msg.MsgType != consensus_interface.HotStuffMessageTypeRoot {
		return
	}
	// convert from wireMessage to SignedMessage since consensus need to verify signature
	signedMessage := &consensus_interface.HotStuffSignedMessage{}
	_, err := signedMessage.UnmarshalMsg(msg.Msg.ContentBytes)
	if err != nil {
		logrus.WithError(err).Debug("failed to parse HotStuffSignedMessage message")
		return
	}

	// TODO: verify if the sender is in the committee.
	// TODO: verify signature

	switch consensus_interface.HotStuffMessageType(signedMessage.HotStuffMessageType) {
	case consensus_interface.HotStuffMessageTypeProposal:
		logrus.Info("handling proposal")
		n.ProcessProposalMessage(signedMessage)
	case consensus_interface.HotStuffMessageTypeVote:
		logrus.Info("handling vote")
		n.ProcessVoteMessage(signedMessage)
	case consensus_interface.HotStuffMessageTypeTimeout:
		logrus.WithField("rand", rand.Int31()).Info("handling timeout")
		n.paceMaker.ProcessRemoteTimeoutMessage(signedMessage)
	default:
		panic("unsupported typev")
	}
}

// notifications

func (n *Partner) NewIncomingMessageEventChannel() chan *transport_interface.IncomingLetter {
	return n.myNewIncomingMessageEventChan
}

// subscribe mine
func (n *Partner) AddSubscriberNewOutgoingMessageEvent(sub transport_interface.NewOutgoingMessageEventSubscriber) {
	n.newOutgoingMessageSubscribers = append(n.newOutgoingMessageSubscribers, sub)
	n.paceMaker.AddSubscriberNewOutgoingMessageEvent(sub)
}

func (n *Partner) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range n.newOutgoingMessageSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing hotstuff partner"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (n *Partner) ensureQCCollector(commitInfoHash string) consensus_interface.SignatureCollector {
	if _, ok := n.pendingQCs[commitInfoHash]; !ok {
		collector := &BlsSignatureCollector{
			CommitteeProvider: n.CommitteeProvider,
		}
		collector.InitDefault()
		n.pendingQCs[commitInfoHash] = collector
	}
	collector := n.pendingQCs[commitInfoHash]
	return collector
}

func (n *Partner) sign(msg Signable) (signature []byte, err error) {
	account, err := n.ConsensusAccountProvider.ProvideAccount()
	if err != nil {
		logrus.WithError(err).Warn("account provider cannot provide private key")
		return
	}
	signature = n.ConsensusSigner.Sign(msg.SignatureTarget(), account)
	return
}
