package core

import (
	"fmt"
	"sync"
	"time"
	// "github.com/spf13/viper"
	log "github.com/sirupsen/logrus"

	"github.com/annchain/OG/types"
	// "github.com/annchain/OG/common"
	"math/rand"
)

const (
	local int = iota
	remote
)

const (
	TxStatusNotExist int = iota
	TxStatusQueue
	TxStatusTip
	TxStatusBadTx
	TxStatusPending
)

type TxPool struct {
	conf TxPoolConfig
	dag  *Dag

	queue     chan *txEvent // queue stores txs that need to validate later
	tips      *TxMap        // tips stores all the tips
	badtxs    *TxMap
	txPending *TxMap
	txLookup  *txLookUp // txLookUp stores all the txs for external query

	close chan struct{}

	mu sync.RWMutex
	wg sync.WaitGroup // for TxPool Stop()
}

func NewTxPool(conf TxPoolConfig, d *Dag) *TxPool {
	pool := &TxPool{
		conf:      conf,
		dag:       d,
		queue:     make(chan *txEvent, conf.QueueSize),
		tips:      NewTxMap(),
		badtxs:    NewTxMap(),
		txPending: NewTxMap(),
		txLookup:  newTxLookUp(),
		close:     make(chan struct{}),
	}
	return pool
}

type TxPoolConfig struct {
	QueueSize     int `mapstructure:"queue_size"`
	TipsSize      int `mapstructure:"tips_size"`
	ResetDuration int `mapstructure:"reset_duration"`
	TxVerifyTime  int `mapstructure:"tx_verify_time"`
	TxValidTime   int `mapstructure:"tx_valid_time"`
}

// type dag interface {
// 	GetTx(hash types.Hash) types.Txi
// 	Push(tx types.Txi) error
// }
type txEvent struct {
	txEnv        *txEnvelope
	callbackChan chan error
}
type txEnvelope struct {
	tx     types.Txi
	txType int
	status int
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

// generate [count] unique random number within range [0, upper)
// if count > upper, use all available indices
func generateRandomIndices(count int, upper int) []int {
	if count > upper {
		count = upper
	}
	// avoid dup
	generated := make(map[int]struct{})
	for count > 0 {
		i := rand.Intn(upper)
		if _, ok := generated[i]; ok {
			continue
		}
		generated[i] = struct{}{}
	}
	arr := make([]int, 0, len(generated))
	for k := range generated {
		arr = append(arr, k)
	}
	return arr
}

// GetRandomTips returns n tips randomly.
func (pool *TxPool) GetRandomTips(n int) (v []types.Txi) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// select n random hashes
	indices := generateRandomIndices(n, len(pool.tips.txs))

	// slice of keys
	keys := make([]types.Hash, len(pool.tips.txs))
	for k := range pool.tips.txs {
		keys = append(keys, k)
	}

	for i := range indices {
		v = append(v, pool.tips.txs[keys[i]])
	}
	return v
}

// GetAllTips returns all the tips in TxPool.
func (pool *TxPool) GetAllTips() map[types.Hash]types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.tips.txs
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

// Remove removes a tx from tx pool. Only be called if a tx moves to dag,
// or a tx need to be clear in reset().
func (pool *TxPool) remove(hash types.Hash) {
	if pool.tips.Get(hash) != nil {
		pool.tips.Remove(hash)
	}
	if pool.badtxs.Get(hash) != nil {
		pool.badtxs.Remove(hash)
	}
	if pool.txPending.Get(hash) != nil {
		pool.txPending.Remove(hash)
	}
	pool.txLookup.remove(hash)
}

func (pool *TxPool) loop() {
	defer log.Debugf("TxPool.loop() terminates")

	pool.wg.Add(1)
	defer pool.wg.Done()

	resetTimer := time.NewTicker(time.Duration(pool.conf.ResetDuration) * time.Second)

	for {
		select {
		case <-pool.close:
			return

		case txEvent := <-pool.queue:
			var err error
			tx := txEvent.txEnv.tx
			switch tx := tx.(type) {
			case *types.Tx:
				err = pool.commit(tx)
			case *types.Sequencer:
				err = pool.confirm(tx)
			}
			txEvent.callbackChan <- err

		// TODO case reset?
		case <-resetTimer.C:
			pool.reset()
		}
	}
}

// addTx adds tx to the pool queue and wait to become tip after validation.
func (pool *TxPool) addTx(tx types.Txi, senderType int) error {
	timer := time.NewTimer(time.Duration(pool.conf.TxVerifyTime) * time.Second)
	defer timer.Stop()

	te := &txEvent{
		callbackChan: make(chan error),
		txEnv: &txEnvelope{
			tx:     tx,
			txType: senderType,
			status: TxStatusQueue,
		},
	}

	select {
	case pool.queue <- te:
		pool.txLookup.add(te.txEnv)
	case <-timer.C:
		return fmt.Errorf("addTx timeout, cannot add tx to queue, tx hash: %s", tx.GetBase().Hash.Hex())
	}

	select {
	case err := <-te.callbackChan:
		if err != nil {
			return err
		}
		// case <-timer.C:
		// 	// close(te.callbackChan)
		// 	return fmt.Errorf("addTx timeout, tx takes too much time, tx hash: %s", tx.MinedHash().Hex())
	}

	log.Debugf("successfully add tx: %s", tx.GetBase().Hash.Hex())
	return nil
}

// commit commits tx to tips pool. commit() checks if this tx is bad tx and moves
// bad tx to badtx list other than tips list. If this tx proves any txs in the
// tip pool, those tips will be removed from tips but stored in txpending.
func (pool *TxPool) commit(tx *types.Tx) error {
	if pool.tips.Count() >= pool.conf.TipsSize {
		return fmt.Errorf("tips pool reaches max size")
	}
	if pool.isBadTx(tx) {
		pool.badtxs.Add(tx)
		pool.txLookup.switchStatus(tx.MinedHash(), TxStatusBadTx)
		return nil
	}
	// move parents to txpending
	for _, pHash := range tx.Parents() {
		if parent := pool.tips.Get(pHash); parent != nil {
			pool.txPending.Add(parent)
			pool.tips.Remove(pHash)
			pool.txLookup.switchStatus(pHash, TxStatusPending)
		}
	}
	pool.tips.Add(tx)
	pool.txLookup.switchStatus(tx.MinedHash(), TxStatusTip)
	return nil
}

func (pool *TxPool) isBadTx(tx *types.Tx) bool {
	// TODO
	// check if tx's parents are bad tx
	// check if tx itself is valid
	return false
}

// confirm pushes a batch of txs that confirmed by a sequencer to the dag.
func (pool *TxPool) confirm(seq *types.Sequencer) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// get sequencer's unconfirmed elders
	elders := make(map[types.Hash]types.Txi)
	pool.seekElders(elders, seq)
	// verify the elders
	if err := pool.verifyConfirmBatch(elders); err != nil {
		return err
	}
	// move elders to dag
	for _, elder := range elders {
		pool.dag.Push(elder)
		pool.remove(elder.MinedHash())
	}
	pool.tips.Add(seq)
	pool.txLookup.switchStatus(seq.MinedHash(), TxStatusTip)
	return nil
}

func (pool *TxPool) seekElders(batch map[types.Hash]types.Txi, baseTx types.Txi) {
	if baseTx == nil || pool.dag.GetTx(baseTx.MinedHash()) != nil {
		return
	}
	if batch[baseTx.MinedHash()] == nil {
		batch[baseTx.MinedHash()] = baseTx
	}
	for _, pHash := range baseTx.Parents() {
		parent := pool.txPending.Get(pHash)
		pool.seekElders(batch, parent)
	}
	return
}

func (pool *TxPool) verifyConfirmBatch(batch map[types.Hash]types.Txi) error {
	// TODO
	// verify the conflicts
	// implement later after the finish of ogdb
	return nil
}

// reset clears the tips that haven't been proved for a long time
func (pool *TxPool) reset() {
	// TODO
}

type txLookUp struct {
	txs map[types.Hash]*txEnvelope
	mu  sync.RWMutex
}

func newTxLookUp() *txLookUp {
	return &txLookUp{
		txs: make(map[types.Hash]*txEnvelope),
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

	t.txs[txEnv.tx.GetBase().Hash] = txEnv
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

type TxMap struct {
	txs map[types.Hash]types.Txi
	mu  sync.RWMutex
}

func NewTxMap() *TxMap {
	tm := &TxMap{
		txs: make(map[types.Hash]types.Txi),
	}
	return tm
}
func (tm *TxMap) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return len(tm.txs)
}
func (tm *TxMap) Get(hash types.Hash) types.Txi {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.txs[hash]
}
func (tm *TxMap) Exists(tx types.Txi) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.txs[tx.MinedHash()] == nil {
		return false
	}
	return true
}
func (tm *TxMap) Remove(hash types.Hash) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.txs, hash)
}
func (tm *TxMap) Add(tx types.Txi) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.txs[tx.MinedHash()] == nil {
		tm.txs[tx.MinedHash()] = tx
	}
}
