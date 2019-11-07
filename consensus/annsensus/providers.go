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
	annsensusCommunicator *ProxyAnnsensusPeerCommunicator
}

func NewDefaultAnnsensusPartnerProvider(
	myAccountProvider account.AccountProvider,
	proposalGenerator bft.ProposalGenerator,
	proposalValidator bft.ProposalValidator,
	decisionMaker bft.DecisionMaker,
	annsensusCommunicator *ProxyAnnsensusPeerCommunicator) *DefaultAnnsensusPartnerProvider {
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
	currentTerm := context.GetTerm()
	dkgPartner, err = dkg.NewDefaultDkgPartner(
		currentTerm.Suite,
		currentTerm.Id,
		currentTerm.PartsNum,
		currentTerm.Threshold,
		currentTerm.AllPartPublicKeys,
		context.GetMyPartSec(),
		dkgComm,
		dkgComm)
	return
}

func (d DefaultAnnsensusPartnerProvider) GetBftPartnerInstance(context ConsensusContextProvider) bft.BftPartner {
	bftComm := NewProxyBftPeerCommunicator(d.annsensusCommunicator)

	bftPartner := bft.NewDefaultBFTPartner(
		context.GetTerm().PartsNum,
		context.GetMyBftId(),
		context.GetBlockTime(),
		bftComm,
		bftComm,
		d.proposalGenerator,
		d.proposalValidator,
		d.decisionMaker,
		DkgToBft(context.GetTerm().AllPartPublicKeys),
	)
	return bftPartner
}

func DkgToBft(dkgInfo []dkg.PartPub) []bft.PeerInfo {
	var peerInfos []bft.PeerInfo
	for _, peer := range dkgInfo {
		peerInfos = append(peerInfos, bft.PeerInfo{
			Id:             peer.Peer.Id,
			PublicKey:      peer.Peer.PublicKey,
			Address:        peer.Peer.Address,
			PublicKeyBytes: peer.Peer.PublicKeyBytes,
		})
	}
	return peerInfos
}