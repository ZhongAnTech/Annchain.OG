package core

import (
	"sync"
	log "github.com/sirupsen/logrus"

	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common"
)

type TxPool struct {
	lock 	sync.RWMutex
	wg		sync.WaitGroup 		// for TxPool Stop()	

	queue 		[]*types.TX		// TODO change type later
	tips		[]*types.TX		// TODO change type later
	txLookup	*txLookUp		// for tx look up, stores all the txs
}

type txLookUp struct {
	// TODO
}

func NewTxPool() *TxPool {
	return &TxPool{}
}

// Start begin the txpool sevices
func (pool *TxPool) Start() {
	log.Infof("TxPool Start")
	// TODO
}

// Stop stops all the txpool sevices
func (pool *TxPool) Stop() {
	// TODO

	log.Infof("TxPool Stopped")
}

// PoolStatus returns the current status of txpool. Including tips count,
// sequencer count, normal tx count etc.
func (pool *TxPool) PoolStatus() {
	// TODO
	return 
}

// Get get a transaction or sequencer according to input hash, 
// if tx not exists return nil
func (pool *TxPool) Get(hash common.Hash) *types.TX {
	// TODO
	return nil
}

// GetRandomTips returns n tips randomly. 
func (pool *TxPool) GetRandomTips(n int) {
	// TODO
	return
}

// GetAllTips returns all the tips in TxPool.
func (pool *TxPool) GetAllTips() {
	// TODO
	return
}

// AddTx adds a tx to txpool if it is valid, note that If success it returns nil.
func (pool *TxPool) AddTx(tx *types.TX) error {
	// TODO
	return nil
}

// AddTxs adds a list of txs to txpool if they are valid. It returns the process 
// result of each tx with an error list.
func (pool *TxPool) AddTxs(txs []*types.TX) []error {
	// TODO
	return nil
}




