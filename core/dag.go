package core

import (
	// "fmt"
	"sync"

	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type DagConfig struct {
}

type Dag struct {
	conf		DagConfig

	db			ogdb.Database
	accessor	*Accessor	

	genesis        types.Txi
	latestSeqencer *types.Sequencer

	close chan struct{}

	wg sync.WaitGroup
	mu sync.RWMutex
}

func NewDag(conf DagConfig, db ogdb.Database) *Dag {
	dag := &Dag{
		conf:		conf,
		db:			db, 
		accessor:	NewAccessor(db),
	}

	return dag
}

func (dag *Dag) Start() {
	log.Infof("TxPool Start")

	// go dag.loop()
}

func (dag *Dag) Stop() {
	close(dag.close)
	dag.wg.Wait()
	log.Infof("TxPool Stopped")
}

// Genesis returns the genesis tx of dag
func (dag *Dag) Genesis() types.Txi {
	return dag.genesis
}

// LatestSequencer returns the latest sequencer stored in dag
func (dag *Dag) LatestSequencer() *types.Sequencer {
	return dag.latestSeqencer
}

// Push trys to move a tx from tx pool to dag db.
func (dag *Dag) Push(tx types.Txi) error {
	return dag.push(tx)
}

// GetTx gets tx from dag network indexed by tx hash. This function querys 
// ogdb only.
func (dag *Dag) GetTx(hash types.Hash) types.Txi {
	return dag.getTx(hash)
}

// RollBack rolls back the dag network.
func (dag *Dag) RollBack() {
	// TODO
}

func (dag *Dag) push(tx types.Txi) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()

	return dag.accessor.WriteTransaction(tx)
}

func (dag *Dag) getTx(hash types.Hash) types.Txi {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()

	return dag.accessor.ReadTransaction(hash)
}

// func (dag *Dag) loop() {

// 	for {
// 		select {
// 		// TODO

// 		default:
// 			break
// 		}
// 	}
// }









