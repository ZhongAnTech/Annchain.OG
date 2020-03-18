package hotstuff_event

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

/**
Implemented according to
HotStuff: BFT Consensus in the Lens of Blockchain
Maofan Yin, Dahlia Malkhi, Michael K. Reiter, Guy Golan Gueta and Ittai Abraham
*/

type Partner struct {
	MessageHub       *Hub
	Ledger           map[string]*Node
	MyId             int
	N                int
	F                int
	LeaderFunc       func(n int, currentRound int) int
	PaceMaker        *PaceMaker
	Safety           *Safety
	BlockTree        *BlockTree
	ProposerElection *ProposerElection
	logger           *logrus.Logger

	quit chan bool
}

func (n *Partner) InitDefault() {
	n.quit = make(chan bool)
	n.logger = SetupOrderedLog(n.MyId)
}
func (n *Partner) Start() {
	messageChannel := n.MessageHub.GetChannel(n.MyId)
	for {
		select {
		case <-n.quit:
			return
		case msg := <-messageChannel:
			switch msg.Typev {
			case PROPOSAL:
				n.ProcessProposalMessage(msg)
			case VOTE:
				n.ProcessVoteMessage(msg)
			case TIMEOUT:
				n.PaceMaker.ProcessRemoteTimeout(msg)
			case LOCAL_TIMEOUT:
				n.PaceMaker.LocalTimeoutRound()
			default:
			}
		}
	}
}

func (n *Partner) Stop() {
	panic("implement me")
}

func (n *Partner) Name() string {
	return fmt.Sprintf("Node %d", n.MyId)
}

func (n *Partner) CreateLeaf(node *Node) (newNode Node) {
	return Node{
		Previous: node.content,
		content:  RandString(15), // get some random string,
	}
}

func (n *Partner) ProcessProposalMessage(msg *Msg) {
	p := msg.Content.(*ContentProposal).Proposal

	n.ProcessCertificates(p.ParentQC)
	currentRound := n.PaceMaker.CurrentRound
	if p.Round != currentRound {
		return
	}
	if msg.SenderId != n.LeaderFunc(n.N, currentRound) {
		return
	}
	n.BlockTree.ExecuteAndInsert(msg)
	voteMsg := n.Safety.MakeVote(p.Id, p.Round, p.ParentQC)
	if voteMsg != nil {
		voteAggregator := n.LeaderFunc(n.N, currentRound+1)
		outMsg := &Msg{
			Typev:    VOTE,
			SenderId: n.MyId,
			Content:  voteMsg,
		}
		n.Sign(outMsg)
		n.MessageHub.Send(outMsg, voteAggregator)
	}
}

func (n *Partner) Sign(msg *Msg) {
	msg.Sig = Signature{
		PartnerId: n.MyId,
		Vote:      true,
	}
}

func (n *Partner) ProcessVoteMessage(msg *Msg) {
	n.BlockTree.ProcessVote(msg)
}

func (n *Partner) ProcessCertificates(qc *QC) {
	n.PaceMaker.AdvanceRound(qc)
	n.Safety.UpdatePreferredRound(qc)
	if qc.LedgerCommitInfo.CommitStateId != nil {
		n.BlockTree.ProcessCommit(qc.GrandParentId)
	}
}

func (n *Partner) ProcessNewRoundEvent() {
	if n.MyId != n.ProposerElection.GetLeader(n.PaceMaker.CurrentRound) {
		// not the leader
		return
	}
	b := n.BlockTree.GenerateProposal(n.PaceMaker.CurrentRound)
	n.MessageHub.SendToAllButMe(&Msg{
		Typev:    PROPOSAL,
		ParentQC: nil,
		Round:    0,
		SenderId: 0,
		Id:       0,
		Sig: Signature{
			PartnerId: n.MyId,
			Vote:      true,
		}
		,
	})
}

type PaceMaker struct {
	MyId             int
	CurrentRound     int
	TimeoutsPerRound map[int]map[int]bool // round : sender list:true
	Safety           *Safety
	MessageHub       *Hub
	BlockTree        *BlockTree
	ProposerElection *ProposerElection
	Partner          *Partner
}

func (m *PaceMaker) ProcessRemoteTimeout(msg *Msg) {
	contentTimeout := msg.Content.(*ContentTimeout)
	m.TimeoutsPerRound[contentTimeout.Round][msg.SenderId] = true
	if len(m.TimeoutsPerRound[contentTimeout.Round]) == m.Partner.F*2+1 {
		m.AdvanceRound(contentTimeout.Round)
	}
}

func (m *PaceMaker) LocalTimeoutRound() {
	if v, ok := m.TimeoutsPerRound[m.CurrentRound][m.MyId]; ok {
		return
	}
	m.Safety.IncreaseLastVoteRound(m.CurrentRound)
	m.SaveConsensusSate()
	timeoutMsg := m.MakeTimeoutMessage()
	m.MessageHub.SendToAllButMe(timeoutMsg, m.MyId)
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
	return &Msg{
		Typev:    TIMEOUT,
		ParentQC: nil,
		Round:    m.BlockTree.PendingBlkTree.HighQC,
		SenderId: m.MyId,
		Id:       0,
		Sig: Signature{
			PartnerId: m.MyId,
			Vote:      true,
		},
	}
}

func (m *PaceMaker) SaveConsensusSate() {

}

func (m *PaceMaker) StopLocalTimer(r int) {

}

func (m *PaceMaker) GetRoundTimer(round int) time.Duration {

}

func (m *PaceMaker) StartLocalTimer(round int, timer time.Duration) {

}

type Safety struct {
	//	voteMsg := &Msg{
	//	Typev:    VOTE,
	//	ParentQC: msg.ParentQC,
	//	Round:    msg.Round,
	//	SenderId: nil,
	//	Id:       msg.Id,
	//}
}

func (s Safety) UpdatePreferredRound(qc *QC) {

}

func (s Safety) MakeVote(id string, round int, qc *QC) *ContentVote {

}

func (s Safety) IncreaseLastVoteRound(round int) {

}

type ProposerElection struct {
}

func (e ProposerElection) GetLeader(round int) int {

}

type BlockTree struct {
	PendingBlkTree interface{}
	PendingVotes   interface{}
	HighQC         *QC
}

func (t BlockTree) ProcessVote(msg *Msg) {

}

func (t BlockTree) ProcessCommit(id int) {

}

func (t BlockTree) ExecuteAndInsert(msg *Msg) {

}

func (t BlockTree) GenerateProposal(round int) interface{} {

}
