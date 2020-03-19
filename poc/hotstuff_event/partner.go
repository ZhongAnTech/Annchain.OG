package hotstuff_event

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

/**
Implemented according to
HotStuff: BFT Consensus in the Lens of Blockchain
Maofan Yin, Dahlia Malkhi, Michael K. Reiter, Guy Golan Gueta and Ittai Abraham
*/

type Partner struct {
	MessageHub       *Hub
	Ledger           *Ledger
	MyId             int
	N                int
	F                int
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
	if msg.SenderId != n.ProposerElection.GetLeader(currentRound) {
		return
	}
	n.BlockTree.ExecuteAndInsert(&p)
	voteMsg := n.Safety.MakeVote(p.Id, p.Round, p.ParentQC)
	if voteMsg != nil {
		voteAggregator := n.ProposerElection.GetLeader(currentRound + 1)
		outMsg := &Msg{
			Typev:    VOTE,
			SenderId: n.MyId,
			Content:  voteMsg,
			Sig: Signature{
				PartnerId: n.MyId,
				Signature: voteMsg.SignatureTarget(),
			},
		}
		n.MessageHub.Send(outMsg, voteAggregator)
	}
}

func (n *Partner) ProcessVoteMessage(msg *Msg) {
	contentVote := msg.Content.(*ContentVote)
	n.BlockTree.ProcessVote(contentVote)
}

func (n *Partner) ProcessCertificates(qc *QC) {
	n.PaceMaker.AdvanceRound(qc.VoteInfo.Round)
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
	b := n.BlockTree.GenerateProposal(n.PaceMaker.CurrentRound, RandString(15))
	n.MessageHub.SendToAllButMe(&Msg{
		Typev:    PROPOSAL,
		SenderId: n.MyId,
		Content:  b,
		Sig: Signature{
			PartnerId: n.MyId,
			Signature: b.SignatureTarget(),
		},
	}, n.MyId)
}

type ProposerElection struct {
	N int
}

func (e ProposerElection) GetLeader(round int) int {
	return round % e.N
}
