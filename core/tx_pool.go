package core

import (
	"sync"
	log "github.com/sirupsen/logrus"

	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common"
)

type TxPool struct {
	mu 		sync.RWMutex
	wg		sync.WaitGroup 		// for TxPool Stop()	

	// TODO change type later
	queue 		[]types.TX		// queue stores txs that need to validate later 
	// TODO change type later
	tips		[]types.TX		// tips stores all the tips
	txLookup	*txLookUp		// txLookUp stores all the txs for external query
}

func NewTxPool() *TxPool {
	pool := &TxPool{
		txLookup: newTxLookUp(),
	}

	return pool
}

// Start begin the txpool sevices
func (pool *TxPool) Start() {
	log.Infof("TxPool Start")
	// TODO

	pool.wg.Add(1)
	go pool.loop()
}

// Stop stops all the txpool sevices
func (pool *TxPool) Stop() {
	// TODO

	pool.wg.Wait()

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
func (pool *TxPool) Get(hash common.Hash) types.TX {
	return pool.txLookup.get(hash)
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

// AddLocalTx adds a tx to txpool if it is valid, note that if success it returns nil.
// AddLocalTx only process tx that sent by local node.
func (pool *TxPool) AddLocalTx(tx types.TX) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	return pool.addTx(tx)
}

// AddLocalTxs adds a list of txs to txpool if they are valid. It returns 
// the process result of each tx with an error list. AddLocalTxs only process 
// txs that sent by local node.
func (pool *TxPool) AddLocalTxs(txs []types.TX) []error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	result := make([]error, len(txs))
	for _, tx := range txs {
		result = append(result, pool.addTx(tx))
	}
	return result
}

// AddRemoteTx adds a tx to txpool if it is valid. AddRemoteTx only process tx 
// sent by remote nodes, and will hold extra functions to prevent from ddos 
// (large amount of invalid tx sent from one node in a short time) attack. 
func (pool *TxPool) AddRemoteTx(tx types.TX) error {

	return nil
}

// AddRemoteTxs 
func (pool *TxPool) AddRemoteTxs(tx []types.TX) []error {
	
	return []error{}
}

func (pool *TxPool) loop() {
	defer log.Debugf("TxPool.loop() terminates")
	defer pool.wg.Done()

	// TODO
}

// addTx does not hold any txpool locks, every function that call this should 
// handle the lock by itself.
func (pool *TxPool) addTx(tx types.TX) error {
	// TODO

	select {

	}

	log.Debugf("addTx: %s", tx.Hash().Hex())
	return nil
}

// func (pool *TxPool) enqueueTx(tx types.TX) error {

// 	return nil
// }


type txLookUp struct {
	mu		sync.RWMutex
	txs		map[common.Hash]types.TX
}

func newTxLookUp() *txLookUp {
	return &txLookUp{
		txs: 	make(map[common.Hash]types.TX),
	}
}

func (t *txLookUp) get(h common.Hash) types.TX {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.txs[h]
}

func (t *txLookUp) add(tx types.TX) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.txs[tx.Hash()] = tx
}

func (t *txLookUp) remove(h common.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.txs, h)
}

func (t *txLookUp) count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return len(t.txs)
}



