package core

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"math/rand"
)

const (
	TxTypeGenesis TxType = iota
	TxTypeLocal
	TxTypeRemote
)

type TxType int

const (
	TxStatusNotExist TxStatus = iota
	TxStatusQueue
	TxStatusTip
	TxStatusBadTx
	TxStatusPending
)

type TxStatus int

func (ts *TxStatus) String() string {
	switch *ts {
	case TxStatusBadTx:
		return "BadTx"
	case TxStatusNotExist:
		return "NotExist"
	case TxStatusPending:
		return "Pending"
	case TxStatusQueue:
		return "Queueing"
	case TxStatusTip:
		return "Tip"
	default:
		return "UnknownStatus"
	}
}

type TxPool struct {
	conf TxPoolConfig
	dag  *Dag

	queue       chan *txEvent // queue stores txs that need to validate later
	tips        *TxMap        // tips stores all the tips
	badtxs      *TxMap
	poolPending *pending
	txLookup    *txLookUp // txLookUp stores all the txs for external query

	close chan struct{}

	mu sync.RWMutex
	wg sync.WaitGroup // for TxPool Stop()

	OnNewTxReceived []chan types.Txi // for notifications of new txs.
}

func (pool *TxPool) GetBenchmarks() map[string]int {
	return map[string]int{
		"queue":      len(pool.queue),
		"event":      len(pool.OnNewTxReceived),
		"txlookup":   len(pool.txLookup.txs),
		"tips":       len(pool.tips.txs),
		"badtxs":     len(pool.badtxs.txs),
		"latest_seq": int(pool.dag.latestSeqencer.Id),
	}
}

func NewTxPool(conf TxPoolConfig, d *Dag) *TxPool {
	pool := &TxPool{
		conf:            conf,
		dag:             d,
		queue:           make(chan *txEvent, conf.QueueSize),
		tips:            NewTxMap(),
		badtxs:          NewTxMap(),
		txLookup:        newTxLookUp(),
		close:           make(chan struct{}),
		OnNewTxReceived: []chan types.Txi{},
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
type txEvent struct {
	txEnv        *txEnvelope
	callbackChan chan error
}
type txEnvelope struct {
	tx     types.Txi
	txType TxType
	status TxStatus
}

// Start begin the txpool sevices
func (pool *TxPool) Start() {
	log.Infof("TxPool Start")
	go pool.loop()
}

// Stop stops all the txpool sevices
func (pool *TxPool) Stop() {
	close(pool.close)
	pool.wg.Wait()

	log.Infof("TxPool Stopped")
}

func (pool *TxPool) Init(genesis *types.Sequencer) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	genesisEnvelope := &txEnvelope{}
	genesisEnvelope.tx = genesis
	genesisEnvelope.status = TxStatusTip
	genesisEnvelope.txType = TxTypeGenesis
	pool.txLookup.Add(genesisEnvelope)
	pool.tips.Add(genesis)

	log.Infof("TxPool finish init")
}

func (pool *TxPool) Name() string {
	return "TxPool"
}

// PoolStatus returns the current number of 
// tips, bad txs and pending txs stored in pool.
func (pool *TxPool) PoolStatus() (int, int, int) {
	return pool.txLookup.Stats()
}

// Get get a transaction or sequencer according to input hash,
// if tx not exists return nil
func (pool *TxPool) Get(hash types.Hash) types.Txi {
	return pool.txLookup.Get(hash)
}

// GetByNonce get a tx or sequencer from pool by sender's address and tx's nonce.
func (pool *TxPool) GetByNonce(addr types.Address, nonce uint64) types.Txi {
	return pool.txLookup.GetByNonce(addr, nonce)
}

// GetStatus gets the current status of a tx
func (pool *TxPool) GetStatus(hash types.Hash) TxStatus {
	return pool.txLookup.Status(hash)
}

// generate [count] unique random numbers within range [0, upper)
// if count > upper, use all available indices
func generateRandomIndices(count int, upper int) []int {
	if count > upper {
		count = upper
	}
	// avoid dup
	generated := make(map[int]struct{})
	for count > len(generated) {
		i := rand.Intn(upper)
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
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// select n random hashes
	values := pool.tips.GetAllValues()
	//log.WithField("parent", types.TxsToString(values)).Debug("current tips")
	indices := generateRandomIndices(n, len(values))
	//log.WithFields(log.Fields{
	//	"require":   n,
	//	"totalTips": len(values),
	//}).Info("getting random tips")

	for _, i := range indices {
		v = append(v, values[i])
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
	return pool.addTx(tx, TxTypeLocal)
}

// AddLocalTxs adds a list of txs to txpool if they are valid. It returns
// the process result of each tx with an error list. AddLocalTxs only process
// txs that sent by local node.
func (pool *TxPool) AddLocalTxs(txs []types.Txi) []error {
	result := make([]error, len(txs))
	for _, tx := range txs {
		result = append(result, pool.addTx(tx, TxTypeLocal))
	}
	return result
}

// AddRemoteTx adds a tx to txpool if it is valid. AddRemoteTx only process tx
// sent by remote nodes, and will hold extra functions to prevent from ddos
// (large amount of invalid tx sent from one node in a short time) attack.
func (pool *TxPool) AddRemoteTx(tx types.Txi) error {
	return pool.addTx(tx, TxTypeRemote)
}

// AddRemoteTxs works as same as AddRemoteTx but processes a list of txs
func (pool *TxPool) AddRemoteTxs(txs []types.Txi) []error {
	result := make([]error, len(txs))
	for _, tx := range txs {
		result = append(result, pool.addTx(tx, TxTypeRemote))
	}
	return result
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
			if pool.Get(tx.GetTxHash()) != nil {
				log.WithField("tx", tx).Warn("Duplicate tx found in txlookup")
				err = fmt.Errorf("Duplicate tx found in txlookup")
				txEvent.callbackChan <- err
				continue
			}

			pool.txLookup.Add(txEvent.txEnv)
			switch tx := tx.(type) {
			case *types.Tx:
				err = pool.commit(tx)
			case *types.Sequencer:
				err = pool.confirm(tx)
			}
			if err != nil {
				pool.txLookup.Remove(txEvent.txEnv.tx.GetTxHash())
			}

			txEvent.callbackChan <- err

		// TODO case reset?
		case <-resetTimer.C:
			pool.reset()
		}
	}
}

// addTx adds tx to the pool queue and wait to become tip after validation.
func (pool *TxPool) addTx(tx types.Txi, senderType TxType) error {
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

	pool.queue <- te

	//select {
	//case pool.queue <- te:
	// race condition
	//pool.txLookup.Add(te.txEnv)
	//case <-timer.C:
	//	return fmt.Errorf("addTx timeout, cannot add tx to queue, tx hash: %s", tx.String())
	//}

	// waiting for callback
	select {
	case err := <-te.callbackChan:
		if err != nil {
			return err
		}
		// notify all subscribers of newTxEvent
		for _, subscriber := range pool.OnNewTxReceived {
			log.Debug("notify subscriber")
			subscriber <- tx
		}
	}

	log.WithField("tx", tx).Debug("successfully added tx to txPool")
	return nil
}

// commit commits tx to tips pool. commit() checks if this tx is bad tx and moves
// bad tx to badtx list other than tips list. If this tx proves any txs in the
// tip pool, those tips will be removed from tips but stored in txpending.
func (pool *TxPool) commit(tx *types.Tx) error {
	log.WithField("tx", tx).Debugf("start commit tx")

	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.isBadTx(tx) {
		log.Debugf("bad tx: %s", tx.String())
		pool.badtxs.Add(tx)
		pool.txLookup.SwitchStatus(tx.GetTxHash(), TxStatusBadTx)
		return nil
	}
	// move parents to txpending
	for _, pHash := range tx.Parents() {
		status := pool.GetStatus(pHash)
		if status != TxStatusTip {
			log.WithField("parent", pHash).WithField("tx", tx).Info("parent is not a tip")
			break
		}
		parent := pool.tips.Get(pHash)
		if parent == nil {
			log.WithField("parent", pHash).WithField("tx", tx).Info("parent not in tips")
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
	pool.txLookup.SwitchStatus(tx.GetTxHash(), TxStatusTip)

	log.WithField("tx", tx).Debugf("finished commit tx")
	return nil
}

func (pool *TxPool) isBadTx(tx *types.Tx) bool {
	var (
		okay = false
		bad  = true
	)

	// check if the tx's parents are bad txs
	for _, parentHash := range tx.Parents() {
		if pool.badtxs.Get(parentHash) != nil {
			return bad
		}
	}

	// check if the nonce is correct
	txinpool := pool.GetByNonce(tx.Sender(), tx.GetNonce())
	if txinpool != nil {
		return bad
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
	log.WithField("seq", seq).Debug("start confirm seq")

	pool.mu.Lock()
	defer pool.mu.Unlock()

	// get sequencer's unconfirmed elders
	elders := pool.seekElders(seq)
	// verify the elders
	log.WithField("seq id",seq.Id).WithField("count", len(elders)).Warn("tx being confirmed by seq")
	batch, err := pool.verifyConfirmBatch(seq, elders)
	if err != nil {
		return err
	}
	// move elders to dag
	for _, elder := range elders {
		status := pool.GetStatus(elder.GetTxHash())
		if status == TxStatusBadTx {
			pool.badtxs.Remove(elder.GetTxHash())
		}
		if status == TxStatusTip {
			pool.tips.Remove(elder.GetTxHash())
		}
		if status == TxStatusPending {
			pool.poolPending.Confirm(elder.GetTxHash())
		}
		pool.txLookup.Remove(elder.GetTxHash())
	}
	pool.dag.Push(batch)
	pool.tips.Add(seq)
	pool.txLookup.SwitchStatus(seq.GetTxHash(), TxStatusTip)

	log.WithField("seq id",seq.Id).WithField("seq", seq).Debug("finished confirm seq")
	return nil
}

func (pool *TxPool) seekElders(baseTx types.Txi) map[types.Hash]types.Txi {
	batch := make(map[types.Hash]types.Txi)

	inSeekingPool := map[types.Hash]int{}
	seekingPool := list.New()
	for _, parentHash := range baseTx.Parents() {
		seekingPool.PushBack(parentHash)
	}
	for seekingPool.Len() > 0 {
		elderHash := seekingPool.Remove(seekingPool.Front()).(types.Hash)
		elder := pool.Get(elderHash)
		if elder == nil {
			continue
		}
		if batch[elder.GetTxHash()] == nil {
			batch[elder.GetTxHash()] = elder
		}
		for _, elderParentHash := range elder.Parents() {
			if _, in := inSeekingPool[elderParentHash]; !in {
				seekingPool.PushBack(elderParentHash)
				inSeekingPool[elderParentHash] = 0
				/*log.WithField("len", seekingPool.Len()).
					WithField("tx", baseTx).
					WithField("as", len(inSeekingPool)).
					WithField("elder", elder).
					WithField("elderParentHash", elderParentHash).
					Debug("seekingpool")
				*/
			}
		}
	}
	return batch
}

func (pool *TxPool) verifyConfirmBatch(seq *types.Sequencer, elders map[types.Hash]types.Txi) (*ConfirmBatch, error) {
	// statistics of the confirmation term.
	// sums up the related address's income and outcome values
	var txhashes types.Hashs
	batch := map[types.Address]*BatchDetail{}
	for _, txi := range elders {
		switch tx := txi.(type) {
		case *types.Sequencer:
			break
		case *types.Tx:
			batchFrom, okFrom := batch[tx.From]
			if !okFrom {
				batchFrom = &BatchDetail{}
				batchFrom.TxList = make(map[types.Hash]types.Txi)
				batchFrom.Neg = math.NewBigInt(0)
				batchFrom.Pos = math.NewBigInt(0)
				batch[tx.From] = batchFrom
			}
			batchFrom.TxList[tx.GetTxHash()] = tx
			batchFrom.Neg.Value.Add(batchFrom.Neg.Value, tx.Value.Value)

			batchTo, okTo := batch[tx.To]
			if !okTo {
				batchTo = &BatchDetail{}
				batchTo.TxList = make(map[types.Hash]types.Txi)
				batchTo.Neg = math.NewBigInt(0)
				batchTo.Pos = math.NewBigInt(0)
				batch[tx.To] = batchTo
			}
			batchTo.Pos.Value.Add(batchTo.Pos.Value, tx.Value.Value)
			txhashes = append(txhashes, tx.GetTxHash())
		}
	}
	for addr, batchDetail := range batch {
		confirmedBalance := pool.dag.GetBalance(addr)
		// if balance of addr < outcome value of addr, then verify failed
		if confirmedBalance.Value.Cmp(batchDetail.Neg.Value) < 0 {
			return nil, fmt.Errorf("the balance of addr %s is not enough", addr.String())
		}
	}

	cb := &ConfirmBatch{}
	cb.Seq = seq
	cb.Batch = batch
	//reverse the txhashes to keeep partial order , accelerate processing speed
	n := len(txhashes)
	for i := 0; i < n/2; i++ {
		txhashes[i], txhashes[n-1-i] = txhashes[n-i-1], txhashes[i]
	}
	cb.TxHashes = &txhashes
	return cb, nil
}

// reset clears the tips that haven't been proved for a long time
func (pool *TxPool) reset() {
	// TODO
}

type pending struct {
	pool *TxPool

	txs   *TxMap
	state map[types.Address]*pendingState
}

func NewPending(pool *TxPool) *pending {
	return &pending{
		pool:  pool,
		txs:   NewTxMap(),
		state: make(map[types.Address]*pendingState),
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
	neg           *math.BigInt
	pos           *math.BigInt
	originBalance *math.BigInt
}

func NewPendingState(addr types.Address, balance *math.BigInt) *pendingState {
	return &pendingState{
		neg:           math.NewBigInt(0),
		pos:           math.NewBigInt(0),
		originBalance: balance,
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

func (tm *TxMap) GetAllKeys() []types.Hash {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var keys []types.Hash
	// slice of keys
	for k := range tm.txs {
		keys = append(keys, k)
	}
	return keys
}

func (tm *TxMap) GetAllValues() []types.Txi {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var values []types.Txi
	// slice of keys
	for _, v := range tm.txs {
		values = append(values, v)
	}
	return values
}

func (tm *TxMap) Exists(tx types.Txi) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if _, ok := tm.txs[tx.GetTxHash()]; !ok {
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

	if _, ok := tm.txs[tx.GetTxHash()]; !ok {
		tm.txs[tx.GetTxHash()] = tx
	}
}



