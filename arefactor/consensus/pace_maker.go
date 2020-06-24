package consensus

import (
	"io"
	"time"

	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/goffchan"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
)

type LedgerAccountHolder interface {
	ProvideAccount() (*types.OgLedgerAccount, error)
	Generate(src io.Reader) (account *types.OgLedgerAccount, err error)
	Load() (account *types.OgLedgerAccount, err error)
	Save() (err error)
}

type PaceMaker struct {
	Logger *logrus.Logger

	CurrentRound    int
	Safety          *Safety
	Partner         *Partner
	Signer          consensus_interface.Signer
	AccountProvider og_interface.LedgerAccountHolder
	Ledger          consensus_interface.Ledger

	CommitteeProvider consensus_interface.CommitteeProvider
	Reporter          *soccerdash.Reporter

	newOutgoingMessageSubscribers []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent

	lastTC     *consensus_interface.TC
	pendingTCs map[int]consensus_interface.SignatureCollector // round : sender list:true

	timer *time.Timer
	quit  chan bool
}

func (m *PaceMaker) InitDefault() {
	m.quit = make(chan bool)
	m.pendingTCs = make(map[int]consensus_interface.SignatureCollector)
	m.timer = time.NewTimer(time.Second * 2)
	m.newOutgoingMessageSubscribers = []transport_interface.NewOutgoingMessageEventSubscriber{}

}

// subscribe mine
func (m *PaceMaker) AddSubscriberNewOutgoingMessageEvent(sub transport_interface.NewOutgoingMessageEventSubscriber) {
	m.newOutgoingMessageSubscribers = append(m.newOutgoingMessageSubscribers, sub)
}

func (m *PaceMaker) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range m.newOutgoingMessageSubscribers {
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

	m.ProcessRemoteTimeout(p, msg.Signature, msg.SenderId)
}

func (m *PaceMaker) ProcessRemoteTimeout(p *consensus_interface.ContentTimeout, signature consensus_interface.Signature, fromId string) {
	id, err := m.CommitteeProvider.GetPeerIndex(fromId)
	if err != nil {
		logrus.WithError(err).WithField("peerId", fromId).
			Debug("error in finding peer in committee")
		return
	}

	m.Partner.ProcessCertificates(p.HighQC, p.TC, "RemoteTimeout")

	collector := m.ensureTCCollector(p.Round)
	collector.Collect(signature, id)

	m.Logger.WithFields(logrus.Fields{
		"from":  fromId,
		"round": p.Round,
		"tcs":   collector.GetCurrentCount(),
	}).Debug("T got")

	if collector.Collected() {
		m.Logger.WithField("round", p.Round).Info("TC got")
		m.AdvanceRound(nil, &consensus_interface.TC{
			Round:          p.Round,
			JointSignature: collector.GetJointSignature(),
		}, "remote tc got")
	}
}

func (m *PaceMaker) LocalTimeoutRound() {
	collector := m.ensureTCCollector(m.CurrentRound)

	m.Safety.IncreaseLastVoteRound(m.CurrentRound)
	m.Ledger.SaveConsensusState(m.Safety.ConsensusState())

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
		SenderId:            m.CommitteeProvider.GetMyPeerId(),
		Signature:           signature,
	}
	letter := &transport_interface.OutgoingLetter{
		Msg:            outMsg,
		SendType:       transport_interface.SendTypeMulticast,
		CloseAfterSent: false,
		EndReceivers:   m.CommitteeProvider.GetAllMemberPeedIds(),
	}
	m.notifyNewOutgoingMessage(letter)

	collector.Collect(signature, m.CommitteeProvider.GetMyPeerIndex())
	logrus.Info("reset timer")
	m.timer.Reset(m.GetRoundTimer(m.CurrentRound))
}

func (m *PaceMaker) AdvanceRound(qc *consensus_interface.QC, tc *consensus_interface.TC, reason string) {
	m.Logger.WithField("qc", qc).WithField("tc", tc).WithField("reason", reason).Trace("advancing round")

	latestRound := 0
	if qc != nil && latestRound < qc.VoteData.Round {
		latestRound = qc.VoteData.Round
	}
	if tc != nil && latestRound < tc.Round {
		latestRound = tc.Round
	}
	if latestRound < m.CurrentRound {
		m.Logger.WithField("cround", latestRound).WithField("currentRound", m.CurrentRound).WithField("reason", reason).Trace("qc round is less than current round so do not advance")
		return
	}
	m.StopLocalTimer(latestRound)
	m.CurrentRound = latestRound + 1

	m.Reporter.Report("CurrentRound", m.CurrentRound, false)

	m.lastTC = tc
	m.Logger.WithField("latestRound", latestRound).WithField("currentRound", m.CurrentRound).WithField("reason", reason).Warn("round advanced")
	if !m.CommitteeProvider.AmILeader(m.CurrentRound) {

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

		// announce it
		outMsg := &consensus_interface.HotStuffSignedMessage{
			HotStuffMessageType: int(consensus_interface.HotStuffMessageTypeVote),
			ContentBytes:        bytes,
			SenderId:            m.CommitteeProvider.GetMyPeerId(),
			Signature:           signature,
		}
		letter := &transport_interface.OutgoingLetter{
			Msg:            outMsg,
			SendType:       transport_interface.SendTypeUnicast,
			CloseAfterSent: false,
			EndReceivers:   []string{m.CommitteeProvider.GetLeaderPeerId(m.CurrentRound)},
		}
		m.notifyNewOutgoingMessage(letter)
	}
	m.StartLocalTimer(m.CurrentRound, m.GetRoundTimer(m.CurrentRound))
	m.Partner.ProcessNewRoundEvent()
}

func (m *PaceMaker) MakeTimeoutMessage() *consensus_interface.ContentTimeout {
	return &consensus_interface.ContentTimeout{
		Round:  m.CurrentRound,
		HighQC: m.Ledger.GetHighQC(),
		TC:     m.lastTC,
	}
}

func (m *PaceMaker) StopLocalTimer(r int) {
	if !m.timer.Stop() {
		logrus.Warn("timer stopped ahead")
		select {
		case <-m.timer.C: // TRY to drain the channel
		default:
		}
	}
	logrus.Warn("timer stopped and cleared")
}

func (m *PaceMaker) GetRoundTimer(round int) time.Duration {
	return time.Second * 10
}

func (m *PaceMaker) StartLocalTimer(round int, duration time.Duration) {
	logrus.Warn("Starting local timer")
	m.timer.Reset(duration)
}

func (m *PaceMaker) ensureTCCollector(round int) consensus_interface.SignatureCollector {
	if _, ok := m.pendingTCs[round]; !ok {
		collector := &BlsSignatureCollector{
			CommitteeProvider: m.CommitteeProvider,
		}
		collector.InitDefault()
		m.pendingTCs[round] = collector
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

	signature = m.Signer.Sign(msg.SignatureTarget(), account.PrivateKey)
	return
}
