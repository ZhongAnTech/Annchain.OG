package consensus

import "github.com/annchain/OG/types/p2p_message"

// ConsensusProcessor provides a engine for parnter/client to reach consensus using different methods
type ConsensusProcessor interface {
	// start processing consensus
	Start()
	// stop processing consensus
	Stop()
	// receive incoming message
	HandleMessage(message p2p_message.Message)
}
