package consensus

import "github.com/annchain/OG/arefactor/consensus_interface"

//func (d DefaultProposalContext) GetHighQC() *consensus_interface.QC {
//	return d.PendingBlockTree.GetHighQC()
//}

type DefaultProposalContextProvider struct {
	PaceMaker        *PaceMaker
	PendingBlockTree *PendingBlockTree
	Safety           *Safety
}

func (d DefaultProposalContextProvider) GetProposalContext() *consensus_interface.ProposalContext {
	return &consensus_interface.ProposalContext{
		CurrentRound: d.PaceMaker.CurrentRound,
		HighQC:       d.Safety.ConsensusState().HighQC,
		TC:           d.PaceMaker.lastTC,
	}
}
