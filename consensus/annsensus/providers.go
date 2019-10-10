package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/account"
)

type BftPartnerProvider interface {
	GetBftPartnerInstance(context ConsensusContextProvider) bft.BftPartner
}

type DkgPartnerProvider interface {
	GetDkgPartnerInstance(context ConsensusContextProvider) (dkg.DkgPartner, error)
}

type DefaultAnnsensusPartnerProvider struct {
	myAccountProvider     account.AccountProvider // interface to the ledger
	proposalGenerator     bft.ProposalGenerator   // interface to the ledger
	proposalValidator     bft.ProposalValidator   // interface to the ledger
	decisionMaker         bft.DecisionMaker       // interface to the ledger
	annsensusCommunicator *AnnsensusCommunicator
}

func NewDefaultAnnsensusPartnerProvider(
	myAccountProvider account.AccountProvider, proposalGenerator bft.ProposalGenerator,
	proposalValidator bft.ProposalValidator, decisionMaker bft.DecisionMaker,
	annsensusCommunicator *AnnsensusCommunicator) *DefaultAnnsensusPartnerProvider {
	return &DefaultAnnsensusPartnerProvider{
		myAccountProvider:     myAccountProvider,
		proposalGenerator:     proposalGenerator,
		proposalValidator:     proposalValidator,
		decisionMaker:         decisionMaker,
		annsensusCommunicator: annsensusCommunicator,
	}
}

func (d DefaultAnnsensusPartnerProvider) GetDkgPartnerInstance(context ConsensusContextProvider) (dkgPartner dkg.DkgPartner, err error) {
	dkgComm := NewProxyDkgPeerCommunicator(d.annsensusCommunicator)
	dkgPartner, err = dkg.NewDefaultDkgPartner(
		context.GetSuite(),
		context.GetTermId(),
		context.GetNbParticipants(),
		context.GetThreshold(),
		context.GetAllPartPubs(),
		context.GetMyPartSec(),
		dkgComm,
		dkgComm)
	return
}

func (d DefaultAnnsensusPartnerProvider) GetBftPartnerInstance(context ConsensusContextProvider) bft.BftPartner {
	bftComm := NewProxyBftPeerCommunicator(d.annsensusCommunicator)

	bftPartner := bft.NewDefaultBFTPartner(
		context.GetNbParticipants(),
		context.GetMyBftId(),
		context.GetBlockTime(),
		bftComm,
		bftComm,
		d.proposalGenerator,
		d.proposalValidator,
		d.decisionMaker,
	)
	return bftPartner
}
