package hotstuff_event

import (
	"fmt"
	"github.com/latifrons/soccerdash"
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
	Report           *soccerdash.Reporter
	quit             chan bool
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
		logrus.Trace("partner loop round start")
		select {
		case <-n.quit:
			return
		case msg := <-messageChannel:
			n.Logger.WithField("msgType", msg.Typev.String()).WithField("msgc", msg).Info("received message")
			if ok := n.signatureOk(msg); !ok {
				fmt.Println(msg)
				panic("signature invalid")
				//continue
			}

			switch msg.Typev {
			case Proposal:
				logrus.Info("handling proposal")
				n.ProcessProposalMessage(msg)
			case Vote:
				logrus.Info("handling vote")
				n.ProcessVoteMessage(msg)
			case Timeout:
				logrus.Info("handling timeout")
				n.PaceMaker.ProcessRemoteTimeout(msg)
			// sync, not a core protocol of the LibraBFT but necessary
			case SyncRequest:
				logrus.Info("handling sync request")
				n.ProcessSyncRequest(msg)
			case SyncResponse:
				logrus.Info("handling sync response")
				n.ProcessSyncResponse(msg)
			default:
				panic("unsupported typev")
			}
		case <-n.PaceMaker.timer.C:
			logrus.WithField("round", n.PaceMaker.CurrentRound).Warn("timeout")
			n.PaceMaker.LocalTimeoutRound()
		}
		logrus.Trace("partner loop round end")
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

func (n *Partner) handleBlock(block *Block) {
	n.ProcessCertificates(block.ParentQC, p.TC, "handleBlock")
	n.BlockTree.ExecuteAndInsert(block)
}

func (n *Partner) ProcessProposalMessage(msg *Msg) {
	p := msg.Content.(*ContentProposal)

	currentRound := n.PaceMaker.CurrentRound

	if p.Proposal.Round != currentRound {
		n.Logger.WithField("pRound", p.Proposal.Round).WithField("currentRound", currentRound).Warn("current round not match.")
		return
	}

	if msg.SenderId != n.PeerIds[n.ProposerElection.GetLeader(currentRound)] {
		n.Logger.WithField("msg.SenderId", msg.SenderId).WithField("current leader", n.ProposerElection.GetLeader(currentRound)).Warn("current leader not match.")
		return
	}

	n.handleBlock(&p.Proposal)

	voteMsg := n.Safety.MakeVote(p.Proposal.Id, p.Proposal.Round, p.Proposal.ParentQC)
	if voteMsg != nil {
		voteAggregator := n.ProposerElection.GetLeader(currentRound + 1)
		outMsg := &Msg{
			Typev:    Vote,
			SenderId: n.PeerIds[n.MyIdIndex],
			Content:  voteMsg,
			Sig: Signature{
				PartnerIndex: n.MyIdIndex,
				Signature:    voteMsg.SignatureTarget(),
			},
		}
		n.MessageHub.Deliver(outMsg, n.PeerIds[voteAggregator], "ProcessProposalMessage"+strconv.Itoa(n.PaceMaker.CurrentRound))
	}
}

func (n *Partner) ProcessVoteMessage(msg *Msg) {
	contentVote := msg.Content.(*ContentVote)
	n.ProcessCertificates(contentVote.QC, contentVote.TC, "VoteM")
	n.BlockTree.ProcessVote(contentVote, msg.Sig)
}

func (n *Partner) ProcessCertificates(qc *QC, tc *TC, reason string) {
	n.PaceMaker.AdvanceRound(qc, tc, reason+"ProcessCertificates"+strconv.Itoa(n.PaceMaker.CurrentRound))
	if qc != nil {
		n.Safety.UpdatePreferredRound(qc)
		if qc.VoteData.ExecStateId != "" {
			n.BlockTree.ProcessCommit(qc.VoteData.ExecStateId)
		}
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
	n.Logger.WithField("proposal", b).Warn("I'm the current leader")
	n.Report.Report("leader", b.Proposal.Round, false)
	n.MessageHub.DeliverToThemIncludingMe(&Msg{
		Typev:    Proposal,
		SenderId: n.PeerIds[n.MyIdIndex],
		Content:  b,
		Sig: Signature{
			PartnerIndex: n.MyIdIndex,
			Signature:    b.SignatureTarget(),
		},
	}, n.PeerIds, "ProcessNewRoundEvent"+strconv.Itoa(n.PaceMaker.CurrentRound))
}

func (n *Partner) SaveConsensusState() {
	n.Logger.WithFields(logrus.Fields{
		"lastVoteRound":  n.Safety.lastVoteRound,
		"preferredRound": n.Safety.preferredRound,
	}).Trace("Persist" + strconv.Itoa(n.PaceMaker.CurrentRound))
}

func (n *Partner) signatureOk(msg *Msg) bool {
	if msg.Sig.PartnerIndex >= len(n.PeerIds) {
		return false
	}
	return msg.SenderId == n.PeerIds[msg.Sig.PartnerIndex]
}

func (n *Partner) ProcessSyncRequest(msg *Msg) {
	contentSyncRequest := msg.Content.(*ContentSyncRequest)
	blk, err := n.BlockTree.pendingBlkTree.GetBlock(contentSyncRequest.Id)
	if err != nil {
		logrus.WithError(err).Warn("block not found for sync")
	}
	contentSyncResponse := &ContentSyncResponse{Block: blk}
	// send response
	outMsg := &Msg{
		Typev:    SyncResponse,
		SenderId: n.PeerIds[n.MyIdIndex],
		Content:  contentSyncResponse,
	}
	n.MessageHub.Deliver(outMsg, msg.SenderId, "sync response")
}

func (n *Partner) RequestBlock(id string) {
	contentSyncRequest := &ContentSyncRequest{Id: id}
	// broadcast request
	outMsg := &Msg{
		Typev:    SyncRequest,
		SenderId: n.PeerIds[n.MyIdIndex],
		Content:  contentSyncRequest,
	}
	n.MessageHub.Broadcast(outMsg, "request block")
}

func (n *Partner) ProcessSyncResponse(msg *Msg) {
	contentSyncResponse := msg.Content.(*ContentSyncResponse)
	n.handleBlock(contentSyncResponse.Block)
}

type ProposerElection struct {
	N int
}

func (e ProposerElection) GetLeader(round int) int {
	return round % e.N
}
