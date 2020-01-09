package verifier

import (
	"github.com/annchain/OG/consensus/campaign"
	"github.com/annchain/OG/og/types"
)

//consensus related verification
type ConsensusVerifier struct {
	VerifyCampaign   func(cp *campaign.Campaign) bool
	VerifyTermChange func(cp *campaign.TermChange) bool
	VerifySequencer  func(cp *types.Sequencer) bool
}

func (c *ConsensusVerifier) Verify(t types.Txi) bool {
	// TODO: verify consensus
	//switch tx := t.(type) {
	//case *types.Tx:
	//	return true
	//case *types.Archive:
	//	return true
	//case *types.ActionTx:
	//	return true
	//case *types.Sequencer:
	//	return c.VerifySequencer(tx)
	//case *campaign.Campaign:
	//	return c.VerifyCampaign(tx)
	//case *campaign.TermChange:
	//	return c.VerifyTermChange(tx)
	//default:
	//	return false
	//}
	//return false
	return true

}

func (c *ConsensusVerifier) Name() string {
	return "ConsensusVerifier"
}

func (v *ConsensusVerifier) Independent() bool {
	return false
}

func (c *ConsensusVerifier) String() string {
	return c.Name()
}
