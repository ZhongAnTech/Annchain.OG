package hotstuff_event

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
)

/**
Implemented according to
HotStuff: BFT Consensus in the Lens of Blockchain
Maofan Yin, Dahlia Malkhi, Michael K. Reiter, Guy Golan Gueta and Ittai Abraham
*/

type Partner struct {
	MessageHub       Hub
	Ledger           *Ledger
	PeerIds          []string
	MyIdIndex        int
	N                int
	F                int
	PaceMaker        *PaceMaker
	Safety           *Safety
	BlockTree        *BlockTree
	ProposerElection *ProposerElection
	Logger           *logrus.Logger

	quit chan bool
}

func (n *Partner) InitDefault() {
	n.quit = make(chan bool)
}
func (n *Partner) Start() {
	messageChannel, err := n.MessageHub.GetChannel(n.PeerIds[n.MyIdIndex])
	if err != nil {
		logrus.WithError(err).WithField("peerId", n.PeerIds[n.MyIdIndex]).Warn("failed to get channel for peer")
	}
	for {
		logrus.Info("partner round start")
		select {
		case <-n.quit:
			return
		case msg := <-messageChannel:
			n.Logger.WithField("msgType", msg.Typev.String()).WithField("msgc", msg).Info("received message")

			switch msg.Typev {
			case Proposal:
				logrus.Info("in proposal")
				n.ProcessProposalMessage(msg)
			case Vote:
				logrus.Info("in vote")
				n.ProcessVoteMessage(msg)
			case Timeout:
				logrus.Info("in timeout")
				n.PaceMaker.ProcessRemoteTimeout(msg)
			//case LocalTimeout:
			//	n.PaceMaker.LocalTimeoutRound()
			default:
				panic("unsupported typev")
			}
		case <-n.PaceMaker.timer.C:
			logrus.Warn("timeout")
			n.PaceMaker.LocalTimeoutRound()
		}
		logrus.Info("partner round end")
	}
}

func (n *Partner) Stop() {
	close(n.quit)
}

func (n *Partner) Name() string {
	return fmt.Sprintf("Node %d", n.MyIdIndex)
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
		n.Logger.WithField("pRound", p.Round).WithField("currentRound", currentRound).Warn("current round not match.")
		return
	}
	if msg.SenderId != n.PeerIds[n.ProposerElection.GetLeader(currentRound)] {
		n.Logger.WithField("msg.SenderId", msg.SenderId).WithField("current leader", n.ProposerElection.GetLeader(currentRound)).Warn("current leader not match.")
		return
	}
	n.BlockTree.ExecuteAndInsert(&p)
	voteMsg := n.Safety.MakeVote(p.Id, p.Round, p.ParentQC)
	if voteMsg != nil {
		voteAggregator := n.ProposerElection.GetLeader(currentRound + 1)
		outMsg := &Msg{
			Typev:    Vote,
			SenderId: n.PeerIds[n.MyIdIndex],
			Content:  voteMsg,
			Sig: Signature{
				PartnerId: n.MyIdIndex,
				Signature: voteMsg.SignatureTarget(),
			},
		}
		n.MessageHub.Deliver(outMsg, n.PeerIds[voteAggregator], "ProcessProposalMessage"+strconv.Itoa(n.PaceMaker.CurrentRound))
	}
}

func (n *Partner) ProcessVoteMessage(msg *Msg) {
	contentVote := msg.Content.(*ContentVote)
	n.BlockTree.ProcessVote(contentVote, msg.Sig)
}

func (n *Partner) ProcessCertificates(qc *QC) {
	n.PaceMaker.AdvanceRound(qc, "ProcessCertificates"+strconv.Itoa(n.PaceMaker.CurrentRound))
	n.Safety.UpdatePreferredRound(qc)
	if qc.LedgerCommitInfo.CommitStateId != "" {
		n.BlockTree.ProcessCommit(qc.VoteInfo.GrandParentId)
	}
}

func (n *Partner) ProcessNewRoundEvent() {
	if n.MyIdIndex != n.ProposerElection.GetLeader(n.PaceMaker.CurrentRound) {
		// not the leader
		n.Logger.Trace("I'm not the leader so just return")
		return
	}
	//b := n.BlockTree.GenerateProposal(n.PaceMaker.CurrentRound, strconv.Itoa(RandInt()))
	b := n.BlockTree.GenerateProposal(n.PaceMaker.CurrentRound, "1")
	n.Logger.WithField("proposal", b).Trace("I'm the current leader")
	fmt.Printf("[%d] pp %d %s\n", n.MyIdIndex, b.Proposal.Round, b.Proposal.Payload)
	n.MessageHub.DeliverToThemIncludingMe(&Msg{
		Typev:    Proposal,
		SenderId: n.PeerIds[n.MyIdIndex],
		Content:  b,
		Sig: Signature{
			PartnerId: n.MyIdIndex,
			Signature: b.SignatureTarget(),
		},
	}, n.PeerIds, "ProcessNewRoundEvent"+strconv.Itoa(n.PaceMaker.CurrentRound))
}

func (n *Partner) SaveConsensusState() {
	n.Logger.WithFields(logrus.Fields{
		"lastVoteRound":  n.Safety.lastVoteRound,
		"preferredRound": n.Safety.preferredRound,
		"pendingBlkTree": n.BlockTree.pendingBlkTree,
	}).Trace("Persist" + strconv.Itoa(n.PaceMaker.CurrentRound))
}

type ProposerElection struct {
	N int
}

func (e ProposerElection) GetLeader(round int) int {
	return round % e.N
}
