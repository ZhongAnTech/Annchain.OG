package core

import ogTypes "github.com/annchain/OG/arefactor/og_interface"

type OgLedger struct {
	pool *TxPool
	dag *Dag
}

type Proposal interface {
	PrevBlock() ogTypes.Hash
}

func (ol *OgLedger) Speculate(proposal Proposal) (blockHash ogTypes.Hash, stateRoot ogTypes.Hash) {



}

func (ol *OgLedger) Commit(blockHash ogTypes.Hash) error {

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

