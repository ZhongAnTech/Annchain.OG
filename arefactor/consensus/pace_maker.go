package consensus

import (
	"fmt"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/goffchan"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

type PaceMaker struct {
	Logger *logrus.Logger

	CurrentRound    int64
	Safety          *Safety
	Partner         *Partner
	ConsensusSigner consensus_interface.ConsensusSigner
	AccountProvider consensus_interface.ConsensusAccountProvider
	//Ledger          consensus_interface.Ledger

	CommitteeProvider consensus_interface.CommitteeProvider
	Reporter          *soccerdash.Reporter
	BlockTime         time.Duration

	// last valid action transition time. Do not use timer.Reset unless you find a way to reset the timer without triggering an event.
	// valid scenario:
	// Once a QVote/TVote is sent
	lastValidTime time.Time

	newOutgoingMessageSubscribers []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent

	pendingTCs map[int64]consensus_interface.SignatureCollector // round : sender list:true

	quit chan bool
}

func (m *PaceMaker) InitDefault() {
	m.quit = make(chan bool)
	m.pendingTCs = make(map[int64]consensus_interface.SignatureCollector)
	m.newOutgoingMessageSubscribers = []transport_interface.NewOutgoingMessageEventSubscriber{}

}

// subscribe mine
func (m *PaceMaker) AddSubscriberNewOutgoingMessageEvent(sub transport_interface.NewOutgoingMessageEventSubscriber) {
	m.newOutgoingMessageSubscribers = append(m.newOutgoingMessageSubscribers, sub)
}

func (m *PaceMaker) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range m.newOutgoingMessageSubscribers {
		logrus.WithField("to", subscriber.Name()).WithField("type", event.String()).Info("notifyNewOutgoingmessage")
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing hotstuff pacemaker"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (m *PaceMaker) ProcessRemoteTimeoutMessage(msg *consensus_interface.HotStuffSignedMessage) {
	p := &consensus_interface.ContentTimeout{}
	err := p.FromBytes(msg.ContentBytes)
	if err != nil {
		logrus.WithError(err).Debug("failed to decode ContentTimeout")
		return
	}

	m.ProcessRemoteTimeout(p, msg.Signature, msg.SenderMemberId)
}

func (m *PaceMaker) ProcessRemoteTimeout(p *consensus_interface.ContentTimeout, signature consensus_interface.Signature, fromMemberId string) {
	id, err := m.CommitteeProvider.GetPeerIndex(fromMemberId)
	if err != nil {
		logrus.WithError(err).WithField("peerId", fromMemberId).
			Fatal("error in finding peer in committee")
		return
	}

	m.Partner.ProcessCertificates(p.HighQC, p.TC, "RemoteTimeout")

	collector := m.ensureTCCollector(p.Round)
	collector.Collect(signature, id)
	m.Reporter.Report("tcsig", fmt.Sprintf("R%d %d J %s", p.Round, collector.GetCurrentCount(), string(collector.GetJointSignature())), false)

	m.Logger.WithFields(logrus.Fields{
		"memberIndex": id,
		"round":       p.Round,
		"tcs":         collector.GetCurrentCount(),
		"rand":        rand.Int31(),
	}).Warn("T got")

	if collector.Collected() {
		m.Logger.WithField("round", p.Round).Info("TC got")
		m.AdvanceRound(nil, &consensus_interface.TC{
			Round:          p.Round,
			JointSignature: collector.GetJointSignature(),
		}, "remote tc got")
	}
}

func (m *PaceMaker) LocalTimeoutRound() {

	logrus.WithField("rand", rand.Int31()).WithField("round", m.CurrentRound).Warn("local timeout")
	_ = m.ensureTCCollector(m.CurrentRound)

	m.Safety.IncreaseLastVoteRound(m.CurrentRound)

	timeoutMsg := m.MakeTimeoutMessage()
	bytes := timeoutMsg.ToBytes()
	signature, err := m.sign(timeoutMsg)
	if err != nil {
		return
	}

	// announce a timeout msg
	outMsg := &consensus_interface.HotStuffSignedMessage{
		HotStuffMessageType: int(consensus_interface.HotStuffMessageTypeTimeout),
		ContentBytes:        bytes,
		SenderMemberId:      m.CommitteeProvider.GetMyPeerId(),
		Signature:           signature,
	}
	letter := &transport_interface.OutgoingLetter{
		ExceptMyself:   false, // send to me also to collect signature.
		Msg:            outMsg,
		SendType:       transport_interface.SendTypeMulticast,
		CloseAfterSent: false,
		EndReceivers:   m.CommitteeProvider.GetAllMemberTransportIds(),
	}
	m.notifyNewOutgoingMessage(letter)
	m.RefreshTimeout()
}

func (m *PaceMaker) AdvanceRound(qc *consensus_interface.QC, tc *consensus_interface.TC, reason string) {
	m.Logger.WithField("qc", qc).WithField("tc", tc).WithField("reason", reason).Trace("advancing round")

	latestRound := int64(0)
	if qc != nil && latestRound < qc.VoteData.Round {
		latestRound = qc.VoteData.Round

	}
	if tc != nil && latestRound < tc.Round {
		latestRound = tc.Round

	}
	if latestRound < m.CurrentRound {
		m.Logger.WithField("cround", latestRound).WithField("currentRound", m.CurrentRound).WithField("reason", reason).Debug("qc/tc round is less than current round so do not advance")
		return
	}
	if qc != nil {
		m.Safety.SetHighQC(qc)
	}
	if tc != nil {
		m.Safety.SetLastTC(tc)
	}

	m.CurrentRound = latestRound + 1

	m.Reporter.Report("CurrentRound", m.CurrentRound, false)
	m.Logger.WithField("latestRound", latestRound).WithField("currentRound", m.CurrentRound).WithField("reason", reason).Info("round advanced")

	// prepare vote message
	vote := &consensus_interface.ContentVote{
		QC: qc,
		TC: tc,
	}
	bytes := vote.ToBytes()
	signature, err := m.sign(vote)
	if err != nil {
		return
	}

	// announce vote to the leader
	outMsg := &consensus_interface.HotStuffSignedMessage{
		HotStuffMessageType: int(consensus_interface.HotStuffMessageTypeVote),
		ContentBytes:        bytes,
		SenderMemberId:      m.CommitteeProvider.GetMyPeerId(),
		Signature:           signature,
	}
	letter := &transport_interface.OutgoingLetter{
		ExceptMyself:   true,
		Msg:            outMsg,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{m.CommitteeProvider.GetLeader(m.CurrentRound).TransportPeerId},
	}
	m.notifyNewOutgoingMessage(letter)
	m.RefreshTimeout()
	m.Partner.ProcessNewRoundEvent()
}

func (m *PaceMaker) MakeTimeoutMessage() *consensus_interface.ContentTimeout {
	consensusState := m.Safety.ConsensusState()
	return &consensus_interface.ContentTimeout{
		Round:  m.CurrentRound,
		HighQC: consensusState.HighQC,
		TC:     consensusState.LastTC,
	}
}

func (m *PaceMaker) GetRoundTimer(round int64) time.Duration {
	return m.BlockTime + time.Second*5
}

func (m *PaceMaker) ensureTCCollector(round int64) consensus_interface.SignatureCollector {
	if _, ok := m.pendingTCs[round]; !ok {
		collector := &BlsSignatureCollector{
			CommitteeProvider: m.CommitteeProvider,
			Round:             round,
		}
		collector.InitDefault()
		m.pendingTCs[round] = collector
		logrus.WithField("round", round).Trace("tc collector established")
	}
	collector := m.pendingTCs[round]
	return collector
}

func (m *PaceMaker) sign(msg Signable) (signature []byte, err error) {
	account, err := m.AccountProvider.ProvideAccount()
	if err != nil {
		logrus.WithError(err).Warn("account provider cannot provide account")
		return
	}

	signature = m.ConsensusSigner.Sign(msg.SignatureTarget(), account)
	return
}

func (m *PaceMaker) RefreshTimeout() {
	m.lastValidTime = time.Now()
	m.Reporter.Report("TIMEOUT", m.lastValidTime.String(), false)
}
