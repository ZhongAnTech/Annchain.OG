package core

import (
	"fmt"
	"sync"

	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type DagConfig struct {
}

type Dag struct {
	conf DagConfig

	db ogdb.Database

	statePending ogdb.StateDB
	txPending    *TxPending

	genesis        types.Txi
	latestSeqencer *types.Sequencer

	close chan struct{}

	wg sync.WaitGroup
	mu sync.RWMutex
}

func NewDag(conf DagConfig, db ogdb.Database) *Dag {
	dag := &Dag{
		conf:      conf,
		db:        db,
		txPending: NewTxPending(),
	}

	return dag
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

// LatestSequencer returns the latest sequencer stored in dag
func (dag *Dag) LatestSequencer() *types.Sequencer {
	return dag.latestSeqencer
}

// Commit trys to move a tx from tx pool to dag network pending list
func (dag *Dag) Commit(tx types.Txi) error {
	return dag.commit(tx)
}

// GetTx gets tx from dag network indexs by tx hash. This function querys
// txPending first and then search in db.
func (dag *Dag) GetTx(hash types.Hash) types.Txi {
	// TODO
	// 1. check pending
	// 2. check db

	return nil
}

// RollBack rolls back the dag network.
func (dag *Dag) RollBack() {
	// TODO
}

func (dag *Dag) commit(tx types.Txi) error {
	// dag.mu.Lock()
	// defer dag.mu.Unlock()

	if dag.GetTx(tx.Hash()) != nil {
		return fmt.Errorf("tx inserted already exists, hash: %s", tx.Hash().Hex())
	}
	switch tx := tx.(type) {
	case *types.Tx:
		return dag.pendingTx(tx)
	case *types.Sequencer:
		return dag.pendingSequencer(tx)
	default:
		return fmt.Errorf("unknown tx type: %v", tx)
	}
}

func (dag *Dag) pendingTx(tx *types.Tx) error {
	dag.txPending.Add(tx)
	// TODO

	return nil
}

func (dag *Dag) pendingSequencer(seq *types.Sequencer) error {
	// TODO
	return nil
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

type TxPending struct {
	txs map[types.Hash]types.Txi
	mu  sync.RWMutex
}

func NewTxPending() *TxPending {
	tp := &TxPending{
		txs: make(map[types.Hash]types.Txi),
	}
	return tp
}
func (tp *TxPending) Get(hash types.Hash) types.Txi {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	return tp.txs[hash]
}
func (tp *TxPending) Exists(tx types.Txi) bool {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	if tp.txs[tx.Hash()] == nil {
		return false
	}
	return true
}
func (tp *TxPending) Remove(hash types.Hash) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	delete(tp.txs, hash)
}
func (tp *TxPending) Add(tx types.Txi) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.txs[tx.Hash()] == nil {
		tp.txs[tx.Hash()] = tx
	}
}
