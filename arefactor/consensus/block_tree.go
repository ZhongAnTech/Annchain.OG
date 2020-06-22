package consensus

import (
	"fmt"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type BlockTree struct {
	Logger    *logrus.Logger
	Ledger    consensus_interface.Ledger
	PaceMaker *PaceMaker
	F         int
	MyIdIndex int
	Report    *soccerdash.Reporter

	pendingBlkTree PendingBlockTree // tree of blocks pending commitment

}

func (t *BlockTree) InitGenesisOrLatest() {
	t.updateHighQC(&consensus_interface.QC{
		VoteData: consensus_interface.VoteInfo{
			Id:               "genesis",
			Round:            0,
			ParentId:         "genesis-1",
			ParentRound:      0,
			GrandParentId:    "genesis-2",
			GrandParentRound: 0,
			ExecStateId:      "genesis-state",
		},
		Signatures: nil,
	})
	t.Ledger.Speculate("", "genesis", "0")
	t.Ledger.Commit("genesis")
	//t.pendingBlkTree.Add(&Block{
	//	Round:    0,
	//	Payload:  "genesispayload",
	//	ParentQC: nil,
	//	Id:       "genesis",
	//})
}

func (t *BlockTree) InitDefault() {
	//t.pendingBlkTree = PendingBlockTree{
	//	MyId:   t.MyIdIndex,
	//	Logger: t.Logger,
	//}
	//t.pendingBlkTree.InitDefault()
}

func (t *BlockTree) ProcessCommit(id string) {
	t.Ledger.Commit(id)
	t.pendingBlkTree.Commit(id)
}

func (t *BlockTree) ExecuteAndInsert(p *consensus_interface.Block) {
	// it is a proposal message
	executeStateId := t.Ledger.Speculate(p.ParentQC.VoteData.Id, p.Id, p.Payload)
	t.Logger.WithFields(logrus.Fields{
		"round":          p.Round,
		"total":          t.Ledger.cache[p.Id].total,
		"blockid":        p.Id,
		"blockContent":   p.Payload,
		"executeStateId": executeStateId,
	}).Info("Block Executed")
	t.Report.Report("blockId", p.Id, false)
	t.Report.Report("executeStateId", executeStateId, false)
	t.pendingBlkTree.Add(p)
	if p.ParentQC.VoteData.Round > t.highQC.VoteData.Round {
		t.Logger.WithField("old", t.highQC).WithField("new", p.ParentQC).Info("highQC updated")
		t.updateHighQC(p.ParentQC)
	} else {
		t.Logger.WithField("old", t.highQC).WithField("new", p.ParentQC).Warn("highQC is not updated")
	}

}

func (t *BlockTree) GenerateProposal(currentRound int, payload string) *consensus_interface.ContentProposal {
	time.Sleep(time.Second * 1)
	//time.Sleep(time.Millisecond * 2)
	return &consensus_interface.ContentProposal{
		Proposal: Block{
			Round:    currentRound,
			Payload:  payload,
			ParentQC: t.highQC,
			Id:       Hash(fmt.Sprintf("%d %s %s", currentRound, payload, t.highQC)),
		},
		TC: t.PaceMaker.lastTC,
	}
}

func (t *BlockTree) updateHighQC(qc *consensus_interface.QC) {
	t.highQC = qc
	t.Report.Report("t.highQC", t.highQC.VoteData.Round, false)
}
