package hotstuff_event

import "time"

type PaceMaker struct {
	MyId             int
	CurrentRound     int
	Safety           *Safety
	MessageHub       *Hub
	BlockTree        *BlockTree
	ProposerElection *ProposerElection
	Partner          *Partner

	timeoutsPerRound map[int]map[int]bool // round : sender list:true
}

func (m *PaceMaker) ProcessRemoteTimeout(msg *Msg) {
	contentTimeout := msg.Content.(*ContentTimeout)
	m.timeoutsPerRound[contentTimeout.Round][msg.SenderId] = true
	if len(m.timeoutsPerRound[contentTimeout.Round]) == m.Partner.F*2+1 {
		m.AdvanceRound(contentTimeout.Round)
	}
}

func (m *PaceMaker) LocalTimeoutRound() {
	if _, ok := m.timeoutsPerRound[m.CurrentRound][m.MyId]; ok {
		return
	}
	m.Safety.IncreaseLastVoteRound(m.CurrentRound)
	SaveConsensusSate()

	timeoutMsg := m.MakeTimeoutMessage()
	m.MessageHub.SendToAllButMe(timeoutMsg, m.MyId)

	m.timeoutsPerRound[m.CurrentRound][m.MyId] = true
}

func (m *PaceMaker) AdvanceRound(latestRound int) {
	if latestRound < m.CurrentRound {
		return
	}
	m.StopLocalTimer(latestRound)
	m.CurrentRound = latestRound + 1
	if m.MyId != m.ProposerElection.GetLeader(m.CurrentRound) {
		m.MessageHub.Send(qc, m.ProposerElection.GetLeader(m.CurrentRound))
	}
	m.StartLocalTimer(m.CurrentRound, m.GetRoundTimer(m.CurrentRound))
	m.Partner.ProcessNewRoundEvent()
}

func (m *PaceMaker) MakeTimeoutMessage() *Msg {
	content := ContentTimeout{Round: m.CurrentRound}
	return &Msg{
		Typev: TIMEOUT,
		Sig: Signature{
			PartnerId: m.MyId,
			Signature: content.SignatureTarget(),
		},
		SenderId: m.MyId,
		Content:  content,
	}
}

func (m *PaceMaker) StopLocalTimer(r int) {

}

func (m *PaceMaker) GetRoundTimer(round int) time.Duration {
	return time.Second * 5
}

func (m *PaceMaker) StartLocalTimer(round int, timer time.Duration) {

}
