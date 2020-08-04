package core

import (
	"fmt"
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/types"
)

type OgLedger struct {
	pool *TxPool
	dag  *Dag
}

type Proposal interface {
	PrevBlock() ogTypes.Hash
}

func (ol *OgLedger) Speculate(proposal Proposal) (stateRoot ogTypes.Hash, err error) {
	seq, ok := proposal.(*types.Sequencer)
	if !ok {
		return nil, fmt.Errorf("OG ledger only support sequencer")
	}

}

func (ol *OgLedger) Push(seq *types.Sequencer) error {
	// TODO handle bad sequencer, make sure parent sequencer has been pushed before.

}

func (ol *OgLedger) Commit(seqHash ogTypes.Hash) error {

}

func (ol *OgLedger) GetConsensusState() {

}

func (ol *OgLedger) SetConsensusState() {

}

func (ol *OgLedger) CurrentHeight() int64 {
	return 0
}

func (ol *OgLedger) CurrentCommittee() {

}
