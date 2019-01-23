package consensus

import "github.com/annchain/OG/types"

type IDag interface {
	LatestSequencer() *types.Sequencer
	GetSequencer(hash types.Hash, id uint64) *types.Sequencer
}

type ConsensusEngine interface {
	// TODO
	// consider if this engine interface should be implemented.
}
