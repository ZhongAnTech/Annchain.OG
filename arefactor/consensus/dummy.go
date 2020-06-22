package consensus

import "github.com/annchain/OG/arefactor/consensus_interface"

type DefaultProposalContext struct {
	PaceMaker        *PaceMaker
	PendingBlockTree *PendingBlockTree
}

func (d DefaultProposalContext) GetCurrentRound() int64 {
	return d.PaceMaker.CurrentRound
}

func (d DefaultProposalContext) GetHighQC() *consensus_interface.QC {
	return d.PendingBlockTree.GetHighQC()
}
