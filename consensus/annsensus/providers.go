package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/account"
)

type BftPartnerProvider interface {
	GetBftPartnerInstance(context ConsensusContext) bft.BftPartner
}

type DkgPartnerProvider interface {
	GetDkgPartnerInstance(context ConsensusContext) (dkg.DkgPartner, error)
}

type DefaultAnnsensusPartnerProvider struct {
	MyAccountProvider account.AccountProvider // interface to the ledger
	ProposalGenerator bft.ProposalGenerator   // interface to the ledger
	ProposalValidator bft.ProposalValidator   // interface to the ledger
	DecisionMaker     bft.DecisionMaker       // interface to the ledger
	BftAdatper        BftMessageAdapter
	DkgAdatper        DkgMessageAdapter
	PeerOutgoing      AnnsensusPeerCommunicatorOutgoing
}

func (d *DefaultAnnsensusPartnerProvider) GetDkgPartnerInstance(context ConsensusContext) (dkgPartner dkg.DkgPartner, err error) {
	if d.PeerOutgoing == nil || d.DkgAdatper == nil {
		panic("wrong init")
	}
	dkgComm := NewProxyDkgPeerCommunicator(d.DkgAdatper, d.PeerOutgoing)
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

func (d *DefaultAnnsensusPartnerProvider) GetBftPartnerInstance(context ConsensusContext) bft.BftPartner {
	if d.PeerOutgoing == nil || d.BftAdatper == nil {
		panic("wrong init")
	}

	bftComm := NewProxyBftPeerCommunicator(d.BftAdatper, d.PeerOutgoing)

	bftPartner := bft.NewDefaultBFTPartner(
		context.GetTerm().PartsNum,
		context.GetMyBftId(),
		context.GetBlockTime(),
		bftComm, //BftPeerCommunicatorIncoming
		bftComm, //BftPeerCommunicatorOutgoing
		d.ProposalGenerator,
		d.ProposalValidator,
		d.DecisionMaker,
		DkgToBft(context.GetTerm().AllPartPublicKeys),
	)
	bftPartner.GetBftPeerCommunicatorIncoming()
	return bftPartner
}

func DkgToBft(dkgInfo []dkg.PartPub) []bft.BftPeer {
	var peerInfos []bft.BftPeer
	for _, peer := range dkgInfo {
		peerInfos = append(peerInfos, bft.BftPeer{
			Id:             peer.Peer.Id,
			PublicKey:      peer.Peer.PublicKey,
			Address:        peer.Peer.Address,
			PublicKeyBytes: peer.Peer.PublicKeyBytes,
		})
	}
	return peerInfos
}
