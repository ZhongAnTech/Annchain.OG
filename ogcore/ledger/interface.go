package ledger

import "github.com/annchain/OG/og/types"

type GenesisGenerator interface {
	WriteGenesis(dag *Dag) *types.Sequencer
}
