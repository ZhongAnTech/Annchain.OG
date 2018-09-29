package consensus

import "github.com/annchain/OG/types"

type IDag interface {
	LatestSequencer() *types.Sequencer
	GetSequencer(hash types.Hash, id uint64) *types.Sequencer
}
