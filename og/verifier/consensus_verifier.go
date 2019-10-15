package verifier

import (
	"github.com/annchain/OG/og/archive"
	"github.com/annchain/OG/og/protocol_message"
)

//consensus related verification
type ConsensusVerifier struct {
	VerifyCampaign   func(cp *protocol_message.Campaign) bool
	VerifyTermChange func(cp *protocol_message.TermChange) bool
	VerifySequencer  func(cp *protocol_message.Sequencer) bool
}

func (c *ConsensusVerifier) Verify(t protocol_message.Txi) bool {
	switch tx := t.(type) {
	case *protocol_message.Tx:
		return true
	case *archive.Archive:
		return true
	case *protocol_message.ActionTx:
		return true
	case *protocol_message.Sequencer:
		return c.VerifySequencer(tx)
	case *protocol_message.Campaign:
		return c.VerifyCampaign(tx)
	case *protocol_message.TermChange:
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
