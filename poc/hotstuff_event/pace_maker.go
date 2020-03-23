package hotstuff_event

import (
	"github.com/sirupsen/logrus"
	"time"
)

type PaceMaker struct {
	MyId             int
	CurrentRound     int
	Safety           *Safety
	MessageHub       *Hub
	BlockTree        *BlockTree
	ProposerElection *ProposerElection
	Partner          *Partner
	Logger           *logrus.Logger

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

	if _, ok := m.timeoutsPerRound[contentTimeout.Round]; !ok {
		collector := SignatureCollector{}
		collector.InitDefault()
		m.timeoutsPerRound[contentTimeout.Round] = collector
	}
	collector := m.timeoutsPerRound[contentTimeout.Round]
	collector.Collect(msg.Sig)

	if collector.Count() == m.Partner.F*2+1 {
		m.AdvanceRound(&QC{
			VoteInfo: VoteInfo{
				Round: contentTimeout.Round,
			},
			LedgerCommitInfo: LedgerCommitInfo{},
			Signatures:       collector.AllSignatures(),
		}, "remote tc got")
	}
}

func (m *PaceMaker) LocalTimeoutRound() {
	if _, ok := m.timeoutsPerRound[m.CurrentRound]; !ok {
		collector := SignatureCollector{}
		collector.InitDefault()
		m.timeoutsPerRound[m.CurrentRound] = collector
	} else {
		collector := m.timeoutsPerRound[m.CurrentRound]
		if collector.Has(m.MyId) {
			return
		}
	}

	m.Safety.IncreaseLastVoteRound(m.CurrentRound)
	m.Partner.SaveConsensusState()

	timeoutMsg := m.MakeTimeoutMessage()
	m.MessageHub.SendToAllButMe(timeoutMsg, m.MyId, "LocalTimeoutRound")
	collector := m.timeoutsPerRound[m.CurrentRound]
	collector.Collect(timeoutMsg.Sig)
}

func (m *PaceMaker) AdvanceRound(qc *QC, reason string) {
	m.Logger.WithField("qc", qc).WithField("reason", reason).Info("advancing round")
	latestRound := qc.VoteInfo.Round
	if latestRound < m.CurrentRound {
		m.Logger.WithField("qc", qc).WithField("currentRound", m.CurrentRound).WithField("reason", reason).Info("qc round is less than current round so do not advance")
		return
	}
	m.StopLocalTimer(latestRound)
	m.CurrentRound = latestRound + 1
	m.Logger.WithField("latestRound", latestRound).WithField("currentRound", m.CurrentRound).WithField("reason", reason).Info("round advanced")
	if m.MyId != m.ProposerElection.GetLeader(m.CurrentRound) {
		m.MessageHub.Send(&Msg{
			Typev:    Vote,
			Sig:      Signature{},
			SenderId: 0,
			Content: &ContentVote{
				VoteInfo:         qc.VoteInfo,
				LedgerCommitInfo: qc.LedgerCommitInfo,
				Signatures:       qc.Signatures,
			},
		}, m.ProposerElection.GetLeader(m.CurrentRound), "AdvanceRound:"+reason)

	}
	m.StartLocalTimer(m.CurrentRound, m.GetRoundTimer(m.CurrentRound))
	m.Partner.ProcessNewRoundEvent()
}

func (m *PaceMaker) MakeTimeoutMessage() *Msg {
	content := &ContentTimeout{Round: m.CurrentRound, HighQC: m.BlockTree.highQC}
	return &Msg{
		Typev: Timeout,
		Sig: Signature{
			PartnerId: m.MyId,
			Signature: content.SignatureTarget(),
		},
		SenderId: m.MyId,
		Content:  content,
	}
}

func (m *PaceMaker) StopLocalTimer(r int) {
	m.timer.Stop()
}

func (m *PaceMaker) GetRoundTimer(round int) time.Duration {
	return time.Second * 5
}

func (m *PaceMaker) StartLocalTimer(round int, duration time.Duration) {
	m.timer.Reset(duration)
}
