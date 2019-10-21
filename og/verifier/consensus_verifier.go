package verifier

import (
	"github.com/annchain/OG/consensus/campaign"
	"github.com/annchain/OG/og/archive"
	"github.com/annchain/OG/og/protocol/ogmessage"
)

//consensus related verification
type ConsensusVerifier struct {
	VerifyCampaign   func(cp *campaign.Campaign) bool
	VerifyTermChange func(cp *campaign.TermChange) bool
	VerifySequencer  func(cp *ogmessage.Sequencer) bool
}

func (c *ConsensusVerifier) Verify(t ogmessage.Txi) bool {
	switch tx := t.(type) {
	case *ogmessage.Tx:
		return true
	case *archive.Archive:
		return true
	case *ogmessage.ActionTx:
		return true
	case *ogmessage.Sequencer:
		return c.VerifySequencer(tx)
	case *campaign.Campaign:
		return c.VerifyCampaign(tx)
	case *campaign.TermChange:
		return c.VerifyTermChange(tx)
	default:
		return false
	}
	return false
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
