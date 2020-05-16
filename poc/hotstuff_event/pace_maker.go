package hotstuff_event

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

type PaceMaker struct {
	PeerIds          []string
	MyIdIndex        int
	CurrentRound     int
	Safety           *Safety
	MessageHub       Hub
	BlockTree        *BlockTree
	ProposerElection *ProposerElection
	Partner          *Partner
	Logger           *logrus.Logger
	lastTC           *TC
	timeoutsPerRound map[int]SignatureCollector // round : sender list:true
	timer            *time.Timer
	quit             chan bool
}

func (m *PaceMaker) InitDefault() {
	m.quit = make(chan bool)
	m.timeoutsPerRound = make(map[int]SignatureCollector)
	m.timer = time.NewTimer(time.Second * 2)
}

func (m *PaceMaker) ProcessRemoteTimeout(msg *Msg) {
	contentTimeout := msg.Content.(*ContentTimeout)
	m.Partner.ProcessCertificates(contentTimeout.HighQC, contentTimeout.TC, "RTM")

	if _, ok := m.timeoutsPerRound[contentTimeout.Round]; !ok {
		collector := SignatureCollector{}
		collector.InitDefault()
		m.timeoutsPerRound[contentTimeout.Round] = collector
	}
	collector := m.timeoutsPerRound[contentTimeout.Round]
	collector.Collect(msg.Sig)
	m.Logger.WithField("detail", fmt.Sprintf("from %s round %d. Collector round %d have %d votes: %s", PrettyId(msg.SenderId), contentTimeout.Round, contentTimeout.Round, collector.Count(), collector.AllSignatures())).
		Debug("T got")

	if collector.Count() == m.Partner.F*2+1 {
		m.Logger.WithField("round", contentTimeout.Round).Info("TC got")
		m.AdvanceRound(nil, &TC{
			Round:      contentTimeout.Round,
			Signatures: collector.AllSignatures(),
		}, "remote tc got")
	}
}

func (m *PaceMaker) LocalTimeoutRound() {
	var collector SignatureCollector
	if _, ok := m.timeoutsPerRound[m.CurrentRound]; !ok {
		collector = SignatureCollector{}
		collector.InitDefault()
		m.timeoutsPerRound[m.CurrentRound] = collector
	} else {
		// different from paper: to keep sending timeout message so that TC may be generated if there is someone missed or later joined in.
		collector = m.timeoutsPerRound[m.CurrentRound]
	}

	m.Safety.IncreaseLastVoteRound(m.CurrentRound)
	m.Partner.SaveConsensusState()

	timeoutMsg := m.MakeTimeoutMessage()
	m.MessageHub.DeliverToThemIncludingMe(timeoutMsg, m.PeerIds, "LocalTimeoutRound")
	collector.Collect(timeoutMsg.Sig)
	logrus.Info("reset timer")
	m.timer.Reset(m.GetRoundTimer(m.CurrentRound))
}

func (m *PaceMaker) AdvanceRound(qc *QC, tc *TC, reason string) {
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

	m.Partner.Report.Report("CurrentRound", m.CurrentRound, false)

	m.lastTC = tc
	m.Logger.WithField("latestRound", latestRound).WithField("currentRound", m.CurrentRound).WithField("reason", reason).Warn("round advanced")
	if m.MyIdIndex != m.ProposerElection.GetLeader(m.CurrentRound) {
		content := &ContentVote{
			QC: qc,
			TC: tc,
		}

		m.MessageHub.Deliver(&Msg{

			Typev: Vote,
			Sig: Signature{
				PartnerIndex: m.MyIdIndex,
				Signature:    content.SignatureTarget(),
			},
			SenderId: m.PeerIds[m.MyIdIndex],
			Content:  content,
		}, m.PeerIds[m.ProposerElection.GetLeader(m.CurrentRound)], "AdvanceRound:"+reason)
	}
	m.StartLocalTimer(m.CurrentRound, m.GetRoundTimer(m.CurrentRound))
	m.Partner.ProcessNewRoundEvent()
}

func (m *PaceMaker) MakeTimeoutMessage() *Msg {
	content := &ContentTimeout{Round: m.CurrentRound, HighQC: m.BlockTree.highQC, TC: m.lastTC}
	return &Msg{
		Typev: Timeout,
		Sig: Signature{
			PartnerIndex: m.MyIdIndex,
			Signature:    content.SignatureTarget(),
		},
		SenderId: m.PeerIds[m.MyIdIndex],
		Content:  content,
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
