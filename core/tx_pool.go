package core

import (
	"fmt"
	"time"
	"sync"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"

	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common"
)

const (
	local int = iota
	remote
)

type TxPool struct {
	conf			TxPoolConfig

	queueLocal 		chan *txEvent		// queue stores txs that need to validate later 
	queueRemote		chan *txEvent
	// TODO change type later
	tips		[]types.TX		// tips stores all the tips
	txLookup	*txLookUp		// txLookUp stores all the txs for external query

	closeChan		chan bool

	mu 			sync.RWMutex
	wg			sync.WaitGroup 		// for TxPool Stop()	
}
func NewTxPool(conf TxPoolConfig) *TxPool {
	pool := &TxPool{
		conf:			conf,
		// TODO replace the viper GetInt key with real one
		queueLocal:		make(chan *txEvent, viper.GetInt("txpool.xxx.xxxx")),
		queueRemote:	make(chan *txEvent, viper.GetInt("txpool.xxx.xxxx")),
		txLookup: 		newTxLookUp(),
		closeChan:		make(chan bool),
	}
	return pool
}
type TxPoolConfig struct {
	TxValidateTime	time.Duration
}

// Start begin the txpool sevices
func (pool *TxPool) Start() {
	log.Infof("TxPool Start")
	// TODO

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

	return pool.addTx(tx, local)
}

// AddLocalTxs adds a list of txs to txpool if they are valid. It returns 
// the process result of each tx with an error list. AddLocalTxs only process 
// txs that sent by local node.
func (pool *TxPool) AddLocalTxs(txs []types.TX) []error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	result := make([]error, len(txs))
	for _, tx := range txs {
		result = append(result, pool.addTx(tx, local))
	}
	return result
}

// AddRemoteTx adds a tx to txpool if it is valid. AddRemoteTx only process tx 
// sent by remote nodes, and will hold extra functions to prevent from ddos 
// (large amount of invalid tx sent from one node in a short time) attack. 
func (pool *TxPool) AddRemoteTx(tx types.TX) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	return pool.addTx(tx, remote)
}

// AddRemoteTxs works as same as AddRemoteTx but processes a list of txs
func (pool *TxPool) AddRemoteTxs(txs []types.TX) []error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	result := make([]error, len(txs))
	for _, tx := range txs {
		result = append(result, pool.addTx(tx, remote))
	}
	return result
}

func (pool *TxPool) loop() {
	defer log.Debugf("TxPool.loop() terminates")

	pool.wg.Add(1)
	defer pool.wg.Done()

	// TODO
}

// addTx does not hold any txpool locks, every function that calls this should 
// handle the lock by itself.
func (pool *TxPool) addTx(tx types.TX, senderType int) error {
	timer := time.NewTimer(pool.conf.TxValidateTime)
	defer timer.Stop()

	te := &txEvent{
		tx: 			tx,
		callbackChan:	make(chan error),
	}

	switch senderType {
	case local:
		select {
		case pool.queueLocal <- te:
			break
		case <-timer.C:
			return fmt.Errorf("addTx timeout, cannot add tx to local queue, tx hash: %s", tx.Hash().Hex())
		}
		break	
	case remote:
		select {
		case pool.queueRemote <- te:
			break
		case <-timer.C:
			return fmt.Errorf("addTx timeout, cannot add tx to remote queue, tx hash: %s", tx.Hash().Hex())
		}
		break
	default:
		return fmt.Errorf("unknown senderType")
	}

	select {
	case err := <- te.callbackChan:
		if err != nil {	return err }
	// case <-timer.C:
	// 	return fmt.Errorf("addLocalTx timeout, tx hash: %s", tx.Hash().Hex())
	}

	log.Debugf("successfully add locak tx: %s", tx.Hash().Hex())
	return nil
}

// addLocalTx does not hold any txpool locks, every function that calls this should 
// handle the lock by itself.
// TODO delete this function later?
func (pool *TxPool) addLocalTx(tx types.TX) error {
	timer := time.NewTimer(pool.conf.TxValidateTime)
	defer timer.Stop()

	te := &txEvent{
		tx: 			tx,
		callbackChan:	make(chan error),
	}

	select {
	case pool.queueLocal <- te:
		break
	case <-timer.C:
		return fmt.Errorf("addLocalTx timeout, cannot add tx to local queue, tx hash: %s", tx.Hash().Hex())
	}

	select {
	case err := <- te.callbackChan:
		if err != nil {	return err }
	}

	log.Debugf("successfully add locak tx: %s", tx.Hash().Hex())
	return nil
}

// addRemoteTx does not hold any txpool locks, every function that calls this should 
// handle the lock by itself.
// TODO delete this function later?
func (pool *TxPool) addRemoteTx(tx types.TX) error {
	timer := time.NewTimer(pool.conf.TxValidateTime)
	defer timer.Stop()

	te := &txEvent{
		tx: 			tx,
		callbackChan:	make(chan error),
	}

	select {
	case pool.queueRemote <- te:
		break
	case <-timer.C:
		return fmt.Errorf("addRemoteTx timeout, cannot add tx to remote queue, tx hash: %s", tx.Hash().Hex())
	}

	select {
	case err := <- te.callbackChan:
		if err != nil {	return err }
	}

	log.Debugf("successfully add locak tx: %s", tx.Hash().Hex())
	return nil
}


type txLookUp struct {
	txs		map[common.Hash]types.TX
	mu		sync.RWMutex
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

type txEvent struct {
	tx				types.TX
	callbackChan	chan error
}




