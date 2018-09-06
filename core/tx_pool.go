package core

import (
	"fmt"
	"sync"
	"time"
	// "github.com/spf13/viper"
	log "github.com/sirupsen/logrus"

	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/math"
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

	queue		chan *txEvent // queue stores txs that need to validate later
	tips		*TxMap        // tips stores all the tips
	badtxs		*TxMap
	poolPending *pending
	txLookup	*txLookUp // txLookUp stores all the txs for external query

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
		txLookup:  newTxLookUp(),
		close:     make(chan struct{}),
	}
	pool.poolPending = NewPending(pool)
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
	return pool.txLookup.Stats()
}

// Get get a transaction or sequencer according to input hash,
// if tx not exists return nil
func (pool *TxPool) Get(hash types.Hash) types.Txi {
	return pool.txLookup.Get(hash)
}

// GetStatus gets the current status of a tx
func (pool *TxPool) GetStatus(hash types.Hash) int {
	return pool.txLookup.Status(hash)
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

// // Remove removes a tx from tx pool. Only be called if a tx moves to dag,
// // or a tx need to be clear in reset().
// func (pool *TxPool) remove(hash types.Hash) {
// 	if pool.tips.Get(hash) != nil {
// 		pool.tips.Remove(hash)
// 	}
// 	if pool.badtxs.Get(hash) != nil {
// 		pool.badtxs.Remove(hash)
// 	}
// 	if pool.poolPending.Get(hash) != nil {
// 		pool.poolPending.Remove(hash)
// 	}
// 	pool.txLookup.remove(hash)
// }

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
		pool.txLookup.Add(te.txEnv)
	case <-timer.C:
		return fmt.Errorf("addTx timeout, cannot add tx to queue, tx hash: %s", tx.MinedHash().Hex())
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

	log.Debugf("successfully add tx: %s", tx.MinedHash().Hex())
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
		pool.txLookup.SwitchStatus(tx.MinedHash(), TxStatusBadTx)
		return nil
	}
	// move parents to txpending
	for _, pHash := range tx.Parents() {
		parent := pool.tips.Get(pHash)
		if parent == nil  {
			break
		}
		// remove sequencer from tips
		if parent.GetType() == types.TxBaseTypeSequencer {
			pool.tips.Remove(pHash)
			pool.txLookup.Remove(pHash)
			continue
		} 
		// move normal tx to pending
		pool.tips.Remove(pHash)
		pool.poolPending.Add(parent)
		pool.txLookup.SwitchStatus(pHash, TxStatusPending)
	}
	pool.tips.Add(tx)
	pool.txLookup.SwitchStatus(tx.MinedHash(), TxStatusTip)
	return nil
}

var (
	okay = false
	bad = true
)
func (pool *TxPool) isBadTx(tx *types.Tx) bool {
	// check if the tx's parents are bad txs
	for _, parentHash := range tx.Parents() {
		if pool.badtxs.Get(parentHash) != nil {
			return bad
		}
	}

	// check if the tx itself has no conflicts with local ledger
	stateFrom, okFrom := pool.poolPending.state[tx.From] 
	if !okFrom {
		stateFrom = NewPendingState(tx.From, pool.dag.GetBalance(tx.From))
	}
	// if ( the value that 'from' already spent )
	// 	+ ( the value that 'from' newly spent ) 
	// 	> ( balance of 'from' in db )
	newNeg := math.NewBigInt(0)
	if newNeg.Value.Add(stateFrom.neg.Value, tx.Value.Value).Cmp(
		stateFrom.originBalance.Value) > 0 {
		return bad
	}

	return okay
}

// confirm pushes a batch of txs that confirmed by a sequencer to the dag.
func (pool *TxPool) confirm(seq *types.Sequencer) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// get sequencer's unconfirmed elders
	elders := make(map[types.Hash]types.Txi)
	pool.seekElders(elders, seq)
	// verify the elders
	batch, err := pool.verifyConfirmBatch(seq, elders); 
	if err != nil {
		return err
	}
	// move elders to dag
	for _, elder := range elders {
		pool.poolPending.Confirm(elder.MinedHash())

		status := pool.GetStatus(elder.MinedHash())
		if status == TxStatusBadTx {
			pool.badtxs.Remove(elder.MinedHash())
		}
		if status == TxStatusTip {
			pool.tips.Remove(elder.MinedHash())
		}
		pool.txLookup.Remove(elder.MinedHash())
	}
	pool.dag.Push(batch)
	pool.tips.Add(seq)
	pool.txLookup.SwitchStatus(seq.MinedHash(), TxStatusTip)
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
		parent := pool.poolPending.Get(pHash)
		pool.seekElders(batch, parent)
	}
	return
}

func (pool *TxPool) verifyConfirmBatch(seq *types.Sequencer, elders map[types.Hash]types.Txi) (*confirmBatch, error) {
	// TODO
	// verify the conflicts
	// implement later after the finish of ogdb
	
	// for _, tx := range elders {

	// }

	return nil, nil
}

// reset clears the tips that haven't been proved for a long time
func (pool *TxPool) reset() {
	// TODO
}

type pending struct {
	pool	*TxPool

	txs		*TxMap
	state	map[types.Address]*pendingState
}
func NewPending(pool *TxPool) *pending{
	return &pending{
		pool:	pool,
		txs:	NewTxMap(),
		state:	make(map[types.Address]*pendingState),
	}
}

func (p *pending) Get(hash types.Hash) types.Txi {
	return p.txs.Get(hash)
}

// Add insert a normal tx into pool's pending and update the pending state.
func (p *pending) Add(tx types.Txi) error {
	switch tx := tx.(type) {
	case *types.Tx:
		// increase neg state of 'from'
		stateFrom, okFrom := p.state[tx.From] 
		if !okFrom {
			balance := p.pool.dag.GetBalance(tx.From)
			stateFrom = NewPendingState(tx.From, balance)
		}
		stateFrom.neg.Value.Add(stateFrom.neg.Value, tx.Value.Value)
		// increase pos state of 'to' 
		stateTo, okTo := p.state[tx.To]
		if !okTo {
			balance := p.pool.dag.GetBalance(tx.To)
			stateTo = NewPendingState(tx.To, balance)
		}
		stateTo.pos.Value.Add(stateTo.pos.Value, tx.Value.Value)
	case *types.Sequencer:
		break
	default:
		return fmt.Errorf("unknown tx type")
	}
	p.txs.Add(tx)
	return nil
}

func (p *pending) Confirm(hash types.Hash) {
	tx := p.txs.Get(hash)
	switch tx := tx.(type) {
	case *types.Sequencer:
		break
	case *types.Tx:
		delete(p.state, tx.From)
		delete(p.state, tx.To)
	}
	p.txs.Remove(hash)
}

// type pendingState map[types.Address]pendingAddrState
type pendingState struct {
	neg				*math.BigInt
	pos				*math.BigInt
	originBalance	*math.BigInt
}
func NewPendingState(addr types.Address, balance *math.BigInt) *pendingState {
	return &pendingState{
		neg:			math.NewBigInt(0),
		pos:			math.NewBigInt(0),
		originBalance:	balance,
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

	// TODO return nil as interface{} will not be nil
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

type txLookUp struct {
	txs map[types.Hash]*txEnvelope
	mu  sync.RWMutex
}

func newTxLookUp() *txLookUp {
	return &txLookUp{
		txs: make(map[types.Hash]*txEnvelope),
	}
}
func (t *txLookUp) Get(h types.Hash) types.Txi {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// TODO return nil as interface{} will not be nil
	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.tx
	}
	return nil
}
func (t *txLookUp) Add(txEnv *txEnvelope) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.txs[txEnv.tx.MinedHash()] = txEnv
}
func (t *txLookUp) Remove(h types.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.txs, h)
}
func (t *txLookUp) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.txs)
}
func (t *txLookUp) Stats() (int, int) {
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
func (t *txLookUp) Status(h types.Hash) int {
	t.mu.RLock()
	defer t.mu.Unlock()

	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.status
	}
	return TxStatusNotExist
}
func (t *txLookUp) SwitchStatus(h types.Hash, status int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if txEnv := t.txs[h]; txEnv != nil {
		txEnv.status = status
	}
}

