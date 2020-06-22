package consensus

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/sirupsen/logrus"
)

// Consensus tree state maintenance
//
type PendingBlockTree struct {
	Logger         *logrus.Logger
	cache          map[string]*consensus_interface.Block
	childRelations map[string][]string
	highQC         *consensus_interface.QC    // highest known QC
	Ledger         consensus_interface.Ledger // Ledger should be operated by
}

func (t *PendingBlockTree) ExecuteProposal(block *consensus_interface.Block) {
	t.AddBranch(block)
	executeStateId := t.Ledger.Speculate(block.ParentQC.VoteData.Id, block.Id, block.Payload)
	logrus.WithField("executeStateId", executeStateId).Debug("executed block")
}

func (t *PendingBlockTree) ExecuteProposalAsync(block *consensus_interface.Block) {
	panic("implement me")
}

func (t *PendingBlockTree) String() string {
	return fmt.Sprintf("[PBT: cache %d relation %d]", len(t.cache), len(t.childRelations))
}

func (t *PendingBlockTree) InitDefault() {
	t.cache = make(map[string]*consensus_interface.Block)
	t.childRelations = make(map[string][]string)
}

func (t *PendingBlockTree) AddBranch(p *consensus_interface.Block) {
	t.cache[p.Id] = p

	vs, ok := t.childRelations[p.ParentQC.VoteData.Id] // get parent QC block id
	if !ok {
		vs = []string{}
	}
	vs = append(vs, p.Id)
	t.childRelations[p.ParentQC.VoteData.Id] = vs
}

func (t *PendingBlockTree) Commit(id string) {
	//fmt.Printf("[%d] Block %s\n", t.MyId, id)
	t.Ledger.Commit(id)

	t.Logger.WithField("id", id).Debug("block commit")
}

func (t *PendingBlockTree) GetBlock(id string) (block *consensus_interface.Block, err error) {
	if v, ok := t.cache[id]; ok {
		block = v
		return
	}
	err = errors.New("block not found")
	return
}

func (t *PendingBlockTree) GetHighQC() *consensus_interface.QC {
	return t.highQC
}

func (t *PendingBlockTree) EnsureHighQC(qc *consensus_interface.QC) {
	if qc.VoteData.Round > t.highQC.VoteData.Round {
		t.highQC = qc
	}
}
