package core

import (
	// "fmt"
	"sync"

	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"github.com/annchain/OG/common/math"
)

type DagConfig struct {
}

type Dag struct {
	conf DagConfig

	db       ogdb.Database
	accessor *Accessor

	genesis        types.Txi
	latestSeqencer *types.Sequencer

	close chan struct{}

	wg sync.WaitGroup
	mu sync.RWMutex
}

func NewDag(conf DagConfig, db ogdb.Database) *Dag {
	dag := &Dag{
		conf:     conf,
		db:       db,
		accessor: NewAccessor(db),
	}

	return dag
}

type confirmBatch struct { 
	seq		*types.Sequencer
	batch	map[types.Address]Batch
}
type Batch struct {
	txList	map[types.Hash]types.Txi
	neg		*math.BigInt
	pos		*math.BigInt
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
func (dag *Dag) Push(batch *confirmBatch) error {
	return dag.push(batch)
}

// GetTx gets tx from dag network indexed by tx hash. This function querys
// ogdb only.
func (dag *Dag) GetTx(hash types.Hash) types.Txi {
	return dag.getTx(hash)
}

// GetBalance read the confirmed balance of an address from ogdb.
func (dag *Dag) GetBalance(addr types.Address) *math.BigInt {
	return dag.accessor.ReadBalance(addr)
}

// RollBack rolls back the dag network.
func (dag *Dag) RollBack() {
	// TODO
}

func (dag *Dag) push(batch *confirmBatch) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()

	// TODO update state
	// dag.accessor.WriteTransaction(tx)
	return nil
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
