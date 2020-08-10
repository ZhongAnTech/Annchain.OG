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

// TODO ledger locker

func (ol *OgLedger) Speculate(proposal Proposal) (stateRoot ogTypes.Hash, err error) {
	seq, ok := proposal.(*types.Sequencer)
	if !ok {
		return nil, fmt.Errorf("OG ledger only support sequencer")
	}
	elderList, _, err := ol.pool.SeekElders(seq)
	if err != nil {
		return nil, fmt.Errorf("seek elder error: %v", err)
	}

	pushBatch := &PushBatch{
		Seq: seq,
		Txs: elderList,
	}
	root, err := ol.dag.Speculate(pushBatch)
	if err != nil {
		return nil, fmt.Errorf("dag speculate err: %v", err)
	}
	return root, nil
}

func (ol *OgLedger) Push(seq *types.Sequencer) (err error) {
	// TODO handle bad sequencer, make sure parent sequencer has been pushed before.

	elders, err := ol.pool.PreConfirm(seq)
	if err != nil {
		return err
	}
	pushBatch := &PushBatch{
		Seq: seq,
		Txs: elders,
	}
	return ol.dag.Push(pushBatch)
}

func (ol *OgLedger) Commit(seqHash ogTypes.Hash) (err error) {
	err = ol.pool.Confirm(seqHash)
	if err != nil {
		return err
	}
	return ol.dag.Commit(seqHash)
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
