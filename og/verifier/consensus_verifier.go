package verifier

import (
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
)

//consensus related verification
type ConsensusVerifier struct {
	VerifyCampaign   func(cp *tx_types.Campaign) bool
	VerifyTermChange func(cp *tx_types.TermChange) bool
	VerifySequencer  func(cp *tx_types.Sequencer) bool
}

func (c *ConsensusVerifier) Verify(t types.Txi) bool {
	switch tx := t.(type) {
	case *tx_types.Tx:
		return true
	case *tx_types.Archive:
		return true
	case *tx_types.ActionTx:
		return true
	case *tx_types.Sequencer:
		return c.VerifySequencer(tx)
	case *tx_types.Campaign:
		return c.VerifyCampaign(tx)
	case *tx_types.TermChange:
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
