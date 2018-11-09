package downloader

import (
	"fmt"
	"github.com/annchain/OG/types"
)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

type insertTxsFn func(seq *types.Sequencer, txs types.Txs) error

// dataPack is a data message returned by a peer for some query.
type dataPack interface {
	PeerId() string
	Items() int
	Stats() string
}

// headerPack is a batch of block headers returned by a peer.
type headerPack struct {
	peerID  string
	headers []*types.SequencerHeader
}

func (p *headerPack) PeerId() string { return p.peerID }
func (p *headerPack) Items() int     { return len(p.headers) }
func (p *headerPack) Stats() string  { return fmt.Sprintf("%d", len(p.headers)) }

// bodyPack is a batch of block bodies returned by a peer.
//sequencer[i] includes txs transactions[i]
type bodyPack struct {
	peerID       string
	transactions [][]*types.Tx
	//sequencer    *types.Sequencer
	sequencers []*types.Sequencer
}

/*
func (p *bodyPack) Sequencer() *types.Sequencer {
	return p.sequencer
}
*/

func (p *bodyPack) Sequencers() []*types.Sequencer {
	return p.sequencers
}

func (p *bodyPack) PeerId() string { return p.peerID }
func (p *bodyPack) Items() int {
	return len(p.transactions)
}
func (p *bodyPack) Stats() string { return fmt.Sprintf("%d", len(p.transactions)) }

// statePack is a batch of states returned by a peer.
type statePack struct {
	peerID string
	states [][]byte
}

func (p *statePack) PeerId() string { return p.peerID }
func (p *statePack) Items() int     { return len(p.states) }
func (p *statePack) Stats() string  { return fmt.Sprintf("%d", len(p.states)) }
