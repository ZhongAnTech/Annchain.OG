package core

import (
	"fmt"
	"time"
	"sync"
	// "github.com/spf13/viper"
	log "github.com/sirupsen/logrus"

	"github.com/annchain/OG/types"
	// "github.com/annchain/OG/common"
)

const (
	local int = iota
	remote
)

const (
	TxStatusNotExist int = iota
	TxStatusQueue 
	TxStatusTip	
)

type TxPool struct {
	conf			TxPoolConfig
	dag				dag

	queue 			chan *txEvent				// queue stores txs that need to validate later 
	tips			map[types.Hash]types.Txi		// tips stores all the tips
	txLookup		*txLookUp					// txLookUp stores all the txs for external query

	close			chan struct{}

	mu 		sync.RWMutex
	wg		sync.WaitGroup 		// for TxPool Stop()	
}
func NewTxPool(conf TxPoolConfig, d dag) *TxPool {
	pool := &TxPool{
		conf:			conf,
		dag:			d,
		queue:			make(chan *txEvent, conf.TxPoolQueueSize),
		txLookup: 		newTxLookUp(),
		close:			make(chan struct{}),
	}
	return pool
}

type TxPoolConfig struct {
	TxPoolQueueSize	int
	TxPoolTipsSize	int
	TxValidateTime	time.Duration
}
type dag interface {
	AddTx(types.Txi)
}
type txEvent struct {
	txEnv			*txEnvelope
	callbackChan	chan error
}
type txEnvelope struct {
	tx		types.Txi
	txType	int
	status	int
}

// Start begin the txpool sevices
func (pool *TxPool) Start() {
	log.Infof("TxPool Start")
	// TODO

	go pool.loop()
}

// Stop stops all the txpool sevices
func (pool *TxPool) Stop() {
	close(pool.close)
	pool.wg.Wait()

	log.Infof("TxPool Stopped")
}

// PoolStatus returns the current status of txpool. 
func (pool *TxPool) PoolStatus() (int, int) {
	return pool.txLookup.stats()
}

// Get get a transaction or sequencer according to input hash, 
// if tx not exists return nil
func (pool *TxPool) Get(hash types.Hash) types.Txi {
	return pool.txLookup.get(hash)
}

// GetStatus gets the current status of a tx
func (pool *TxPool) GetStatus(hash types.Hash) int {
	return pool.txLookup.status(hash)
}

// GetRandomTips returns n tips randomly. 
func (pool *TxPool) GetRandomTips(n int) map[types.Hash]types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	result := map[types.Hash]types.Txi{}
	i := 0
	for k, v := range pool.tips {
		if i >= n {
			return result
		}
		result[k] = v
		i = i + 1 
	}
	return result
}

// GetAllTips returns all the tips in TxPool.
func (pool *TxPool) GetAllTips() map[types.Hash]types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.tips
}

// AddLocalTx adds a tx to txpool if it is valid, note that if success it returns nil.
// AddLocalTx only process tx that sent by local node.
func (pool *TxPool) AddLocalTx(tx types.Txi) error {
	return pool.addTx(tx, local)
}

// AddLocalTxs adds a list of txs to txpool if they are valid. It returns 
// the process result of each tx with an error list. AddLocalTxs only process 
// txs that sent by local node.
func (pool *TxPool) AddLocalTxs(txs []types.Txi) []error {
	result := make([]error, len(txs))
	for _, tx := range txs {
		result = append(result, pool.addTx(tx, local))
	}
	return result
}

// AddRemoteTx adds a tx to txpool if it is valid. AddRemoteTx only process tx 
// sent by remote nodes, and will hold extra functions to prevent from ddos 
// (large amount of invalid tx sent from one node in a short time) attack. 
func (pool *TxPool) AddRemoteTx(tx types.Txi) error {
	return pool.addTx(tx, remote)
}

// AddRemoteTxs works as same as AddRemoteTx but processes a list of txs
func (pool *TxPool) AddRemoteTxs(txs []types.Txi) []error {
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

	for {
		select {
		case <-pool.close:
			return

		case txEvent := <-pool.queue:
			txEnv := txEvent.txEnv
			if err := pool.validateTx(txEnv.tx); err != nil {
				txEvent.callbackChan <- err
				continue
			}
			if err := pool.commit(txEnv.tx); err != nil {
				txEvent.callbackChan <- err
				continue
			}
			pool.txLookup.switchStatus(txEnv.tx.Hash(), TxStatusTip)
			txEvent.callbackChan <- nil
		// TODO case reset?
		}
	}
}

// addTx push tx to the pool queue and wait to become tip after validation. 
func (pool *TxPool) addTx(tx types.Txi, senderType int) error {
	timer := time.NewTimer(pool.conf.TxValidateTime)
	defer timer.Stop()

	te := &txEvent{
		callbackChan: make(chan error),
		txEnv: &txEnvelope{
			tx: 	tx,
			txType:	senderType,
			status:	TxStatusQueue,	
		},
	}

	select {
	case pool.queue <- te:
		pool.txLookup.add(te.txEnv)
		break
	case <-timer.C:
		return fmt.Errorf("addTx timeout, cannot add tx to queue, tx hash: %s", tx.Hash().Hex())
	}

	select {
	case err := <- te.callbackChan:
		if err != nil {	return err }
	// case <-timer.C:
	// 	return fmt.Errorf("addLocalTx timeout, tx hash: %s", tx.Hash().Hex())
	}

	log.Debugf("successfully add tx: %s", tx.Hash().Hex())
	return nil
}

func (pool *TxPool) validateTx(tx types.Txi) error {
	// TODO

	return nil
}

// commit commits tx to tip pool. if this tx proves any txs in the tip pool, those 
// tips will be removed from pool but stored in dag.
func (pool *TxPool) commit(tx types.Txi) error { 
	if len(pool.tips) >= pool.conf.TxPoolTipsSize { 
		return fmt.Errorf("tips pool reaches max size")
	}
	for _, hash := range tx.Parents() { 
		if parent, ok := pool.tips[hash]; ok {
			pool.dag.AddTx(parent)
			delete(pool.tips, hash)
			pool.txLookup.remove(hash)
		}
	}
	pool.tips[tx.Hash()] = tx
	return nil
}


type txLookUp struct {
	txs		map[types.Hash]*txEnvelope
	mu		sync.RWMutex
}
func newTxLookUp() *txLookUp {
	return &txLookUp{
		txs: 	make(map[types.Hash]*txEnvelope),
	}
}
func (t *txLookUp) get(h types.Hash) types.Txi {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.tx
	}
	return nil
}
func (t *txLookUp) add(txEnv *txEnvelope) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.txs[txEnv.tx.Hash()] = txEnv
}
func (t *txLookUp) remove(h types.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.txs, h)
} 
func (t *txLookUp) count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return len(t.txs)
}
func (t *txLookUp) stats() (int, int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	queue, tips := 0, 0
	for _, v := range t.txs {
		if v.txType == TxStatusQueue {
			queue += 1
		} else {
			tips += 1
		}
	}
	return queue, tips
}
func (t *txLookUp) status(h types.Hash) int {
	t.mu.RLock()
	defer t.mu.Unlock()

	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.status
	}
	return TxStatusNotExist
}
func (t *txLookUp) switchStatus(h types.Hash, status int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if txEnv := t.txs[h]; txEnv != nil {
		txEnv.status = status
	}
}



