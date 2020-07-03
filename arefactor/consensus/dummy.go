package consensus

type DefaultProposalContext struct {
	PaceMaker        *PaceMaker
	PendingBlockTree *PendingBlockTree
}

func (d DefaultProposalContext) GetCurrentRound() int {
	return d.PaceMaker.CurrentRound
}

//func (d DefaultProposalContext) GetHighQC() *consensus_interface.QC {
//	return d.PendingBlockTree.GetHighQC()
//}
