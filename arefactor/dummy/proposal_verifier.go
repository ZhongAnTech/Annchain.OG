package dummy

import "github.com/annchain/OG/arefactor/consensus_interface"

type DummyProposalVerifier struct {
}

func (d DummyProposalVerifier) VerifyProposal(proposal *consensus_interface.ContentProposal) *consensus_interface.VerifyResult {
	return &consensus_interface.VerifyResult{
		Ok: true,
	}
}

func (d DummyProposalVerifier) VerifyProposalAsync(proposal *consensus_interface.ContentProposal) {
	panic("implement me")
}
