package core

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
)

type Dag struct {
	db ogdb.Database

	statePending	ogdb.StateDB		
	txPending		map[types.Hash]types.Txi		

	genesis			types.Txi
	currentSeqencer	*types.Sequencer

	TxPool		*TxPool

	close 	chan struct{}

	wg	sync.WaitGroup
	mu	sync.RWMutex
}

func (dag *Dag) Start() {
	log.Infof("TxPool Start")

	go dag.loop()
}

func (dag *Dag) Stop() {
	close(dag.close) 
	
	log.Infof("TxPool Stopped")
}

// Genesis returns the genesis tx of dag
func (dag *Dag) Genesis() types.Txi {
	return dag.genesis
}

// CurrentSequencer returns the latest sequencer stored in dag
func (dag *Dag) CurrentSequencer() *types.Sequencer {
	// TODO 
	return nil
}

// AddTx trys to insert a tx from tx pool to dag network pending list
func (dag *Dag) AddTx(types.Txi)  {
	// TODO
}

// GetTx gets tx from dag network indexs by tx hash. This function querys 
// txPending first and then search in db.
func (dag *Dag) GetTx(types.Hash) types.Txi {
	// TODO
	return nil
}

// RollBack rolls back the dag network.
func (dag *Dag) RollBack() {
	// TODO
}




func (dag *Dag) loop() {

	for {
		select {
		// TODO
		default:
			break
		}
	}
}


