package core

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"math/rand"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type TxType int

const (
	TxTypeGenesis TxType = iota
	TxTypeLocal
	TxTypeRemote
	TxTypeRejudge
)

type TxStatus int

const (
	TxStatusNotExist TxStatus = iota
	TxStatusQueue
	TxStatusTip
	TxStatusBadTx
	TxStatusPending
)

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

type TxQuality int

const (
	TxQualityIsBad TxQuality = iota
	TxQualityIsGood
	TxQualityIsFatal
)

type hashOrderRemoveType byte

const (
	noRemove hashOrderRemoveType = iota
	removeFromFront
	removeFromEnd
)

type TxPool struct {
	conf TxPoolConfig
	dag  *Dag

	queue    chan *txEvent // queue stores txs that need to validate later
	tips     *TxMap        // tips stores all the tips
	badtxs   *TxMap
	pendings *TxMap
	flows    *AccountFlows
	txLookup *txLookUp // txLookUp stores all the txs for external query

	close chan struct{}

	mu sync.RWMutex
	wg sync.WaitGroup // for TxPool Stop()

	onNewTxReceived      map[string]chan types.Txi       // for notifications of new txs.
	OnBatchConfirmed     []chan map[types.Hash]types.Txi // for notifications of confirmation.
	OnNewLatestSequencer []chan bool                     //for broadcasting new latest sequencer to record height
	txNum                uint32
	maxWeight            uint64
}

func (pool *TxPool) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"queue":      len(pool.queue),
		"event":      len(pool.onNewTxReceived),
		"txlookup":   len(pool.txLookup.txs),
		"tips":       len(pool.tips.txs),
		"badtxs":     len(pool.badtxs.txs),
		"latest_seq": int(pool.dag.latestSequencer.Number()),
	}
}

func NewTxPool(conf TxPoolConfig, d *Dag) *TxPool {
	pool := &TxPool{
		conf:             conf,
		dag:              d,
		queue:            make(chan *txEvent, conf.QueueSize),
		tips:             NewTxMap(),
		badtxs:           NewTxMap(),
		pendings:         NewTxMap(),
		flows:            NewAccountFlows(),
		txLookup:         newTxLookUp(),
		close:            make(chan struct{}),
		onNewTxReceived:  make(map[string]chan types.Txi),
		OnBatchConfirmed: []chan map[types.Hash]types.Txi{},
	}
	return pool
}

type TxPoolConfig struct {
	QueueSize              int `mapstructure:"queue_size"`
	TipsSize               int `mapstructure:"tips_size"`
	ResetDuration          int `mapstructure:"reset_duration"`
	TxVerifyTime           int `mapstructure:"tx_verify_time"`
	TxValidTime            int `mapstructure:"tx_valid_time"`
	TimeOutPoolQueue       int `mapstructure:"timeout_pool_queue_ms"`
	TimeoutSubscriber      int `mapstructure:"timeout_subscriber_ms"`
	TimeoutConfirmation    int `mapstructure:"timeout_confirmation_ms"`
	TimeoutLatestSequencer int `mapstructure:"timeout_latest_seq_ms"`
}

func DefaultTxPoolConfig() TxPoolConfig {
	config := TxPoolConfig{
		QueueSize:              100,
		TipsSize:               1000,
		ResetDuration:          10,
		TxVerifyTime:           2,
		TxValidTime:            100,
		TimeOutPoolQueue:       10000,
		TimeoutSubscriber:      10000,
		TimeoutConfirmation:    10000,
		TimeoutLatestSequencer: 10000,
	}
	return config
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
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.poolStatus()
}
func (pool *TxPool) poolStatus() (int, int, int) {
	return pool.txLookup.Stats()
}

// Get get a transaction or sequencer according to input hash,
// if tx not exists return nil
func (pool *TxPool) Get(hash types.Hash) types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.get(hash)
}

func (pool *TxPool) GetTxNum() uint32 {
	return atomic.LoadUint32(&pool.txNum)
}

func (pool *TxPool) GetMaxWeight() uint64 {
	return atomic.LoadUint64(&pool.maxWeight)
}

func (pool *TxPool) get(hash types.Hash) types.Txi {
	return pool.txLookup.Get(hash)
}

func (pool *TxPool) Has(hash types.Hash) bool {
	return pool.Get(hash) != nil
}

func (pool *TxPool) IsLocalHash(hash types.Hash) bool {
	if pool.Has(hash) {
		return true
	}
	if pool.dag.Has(hash) {
		return true
	}
	return false
}

// GetHashOrder returns a hash list of txs in pool, ordered by the
// time that txs added into pool.
func (pool *TxPool) GetHashOrder() []types.Hash {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getHashOrder()
}

func (pool *TxPool) getHashOrder() []types.Hash {
	return pool.txLookup.GetOrder()
}

// GetByNonce get a tx or sequencer from account flows by sender's address and tx's nonce.
func (pool *TxPool) GetByNonce(addr types.Address, nonce uint64) types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getByNonce(addr, nonce)
}
func (pool *TxPool) getByNonce(addr types.Address, nonce uint64) types.Txi {
	return pool.flows.GetTxByNonce(addr, nonce)
}

// GetLatestNonce get the latest nonce of an address
func (pool *TxPool) GetLatestNonce(addr types.Address) (uint64, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.flows.GetLatestNonce(addr)
}

// GetStatus gets the current status of a tx
func (pool *TxPool) GetStatus(hash types.Hash) TxStatus {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getStatus(hash)
}
func (pool *TxPool) getStatus(hash types.Hash) TxStatus {
	return pool.txLookup.Status(hash)
}

func (pool *TxPool) RegisterOnNewTxReceived(c chan types.Txi, chanName string) {
	log.Tracef("RegisterOnNewTxReceived with chan: %s", chanName)
	pool.onNewTxReceived[chanName] = c
}

// GetRandomTips returns n tips randomly.
func (pool *TxPool) GetRandomTips(n int) (v []types.Txi) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// select n random hashes
	values := pool.tips.GetAllValues()
	indices := generateRandomIndices(n, len(values))

	for _, i := range indices {
		v = append(v, values[i])
	}
	return v
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

// Remove totally removes a tx from pool, it checks badtxs, tips,
// pendings and txlookup.
func (pool *TxPool) Remove(tx types.Txi, removeType hashOrderRemoveType) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.remove(tx, removeType)
}

func (pool *TxPool) remove(tx types.Txi, removeType hashOrderRemoveType) {
	status := pool.getStatus(tx.GetTxHash())
	if status == TxStatusBadTx {
		pool.badtxs.Remove(tx.GetTxHash())
	}
	if status == TxStatusTip {
		pool.tips.Remove(tx.GetTxHash())
		pool.flows.Remove(tx)
	}
	if status == TxStatusPending {
		pool.pendings.Remove(tx.GetTxHash())
		pool.flows.Remove(tx)
	}
	pool.txLookup.Remove(tx.GetTxHash(), removeType)
}

// ClearAll removes all the txs in the pool.
//
// Note that ClearAll should only be called when solving conflicts
// during a sequencer confirmation time.
func (pool *TxPool) ClearAll() {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.clearAll()
}

func (pool *TxPool) clearAll() {
	pool.badtxs = NewTxMap()
	pool.tips = NewTxMap()
	pool.pendings = NewTxMap()
	pool.flows = NewAccountFlows()
	pool.txLookup = newTxLookUp()
}

func (pool *TxPool) loop() {
	defer log.Tracef("TxPool.loop() terminates")

	pool.wg.Add(1)
	defer pool.wg.Done()

	resetTimer := time.NewTicker(time.Duration(pool.conf.ResetDuration) * time.Second)

	for {
		select {
		case <-pool.close:
			return

		case txEvent := <-pool.queue:
			log.WithField("tx", txEvent.txEnv.tx).Trace("get tx from queue")

			var err error
			tx := txEvent.txEnv.tx
			// check if tx is duplicate
			if pool.get(tx.GetTxHash()) != nil {
				log.WithField("tx", tx).Warn("Duplicate tx found in txlookup")
				txEvent.callbackChan <- types.ErrDuplicateTx
				continue
			}
			pool.mu.Lock()
			pool.txLookup.Add(txEvent.txEnv)
			switch tx := tx.(type) {
			case *types.Tx:
				err = pool.commit(tx)
				//if err is not nil , item removed inside commit
				if err == nil {
					atomic.AddUint32(&pool.txNum, 1)
					maxWeight := atomic.LoadUint64(&pool.maxWeight)
					if maxWeight < tx.GetWeight() {
						atomic.StoreUint64(&pool.maxWeight, tx.GetWeight())
					}
					tx.GetBase().Height = pool.dag.LatestSequencer().Height + 1 //temporary height ,will be re write after confirm
				}
			case *types.Sequencer:
				err = pool.confirm(tx)
				if err != nil {
					pool.txLookup.Remove(txEvent.txEnv.tx.GetTxHash(), removeFromEnd)
				} else {
					atomic.StoreUint32(&pool.txNum, 0)
					maxWeight := atomic.LoadUint64(&pool.maxWeight)
					if maxWeight < tx.GetWeight() {
						atomic.StoreUint64(&pool.maxWeight, tx.GetWeight())
					}
				}
			}
			pool.mu.Unlock()

			txEvent.callbackChan <- err

			// TODO case reset?
		case <-resetTimer.C:
			pool.reset()
		}
	}
}

// addTx adds tx to the pool queue and wait to become tip after validation.
func (pool *TxPool) addTx(tx types.Txi, senderType TxType) error {
	log.WithField("tx", tx).Trace("start addTx")

	te := &txEvent{
		callbackChan: make(chan error),
		txEnv: &txEnvelope{
			tx:     tx,
			txType: senderType,
			status: TxStatusQueue,
		},
	}
	pool.queue <- te
	// <-ffchan.NewTimeoutSenderShort(pool.queue, te, "poolAddTx").C

	// waiting for callback
	select {
	case err := <-te.callbackChan:
		if err != nil {
			return err
		}
		// notify all subscribers of newTxEvent
		for name, subscriber := range pool.onNewTxReceived {
			log.WithField("tx", tx).Trace("notify subscriber: ", name)
			subscriber <- tx
			// <-ffchan.NewTimeoutSenderShort(subscriber, tx, "notifySubscriber").C
		}
	}

	log.WithField("tx", tx).Trace("successfully added tx to txPool")
	return nil
}

// commit commits tx to tips pool. commit() checks if this tx is bad tx and moves
// bad tx to badtx list other than tips list. If this tx proves any txs in the
// tip pool, those tips will be removed from tips but stored in pending.
func (pool *TxPool) commit(tx *types.Tx) error {
	log.WithField("tx", tx).Trace("start commit tx")

	// check tx's quality.
	txquality := pool.isBadTx(tx)
	if txquality == TxQualityIsFatal {
		pool.remove(tx, removeFromEnd)
		return fmt.Errorf("tx is surely incorrect to commit, hash: %s", tx.GetTxHash().String())
	}
	if txquality == TxQualityIsBad {
		log.Tracef("bad tx: %s", tx.String())
		pool.badtxs.Add(tx)
		pool.txLookup.SwitchStatus(tx.GetTxHash(), TxStatusBadTx)
		return nil
	}

	// move parents to pending
	for _, pHash := range tx.Parents() {
		status := pool.getStatus(pHash)
		if status != TxStatusTip {
			log.WithField("parent", pHash).WithField("tx", tx).
				Tracef("parent is not a tip")
			continue
		}
		parent := pool.tips.Get(pHash)
		if parent == nil {
			log.WithField("parent", pHash).WithField("tx", tx).
				Warn("parent status is tip but can not find in tips")
			continue
		}
		// remove sequencer from pool
		if parent.GetType() == types.TxBaseTypeSequencer {
			pool.tips.Remove(pHash)
			pool.txLookup.Remove(pHash, removeFromFront)
			continue
		}
		// move parent to pending
		pool.tips.Remove(pHash)
		pool.pendings.Add(parent)
		pool.txLookup.SwitchStatus(pHash, TxStatusPending)
	}
	// add tx to pool
	if pool.flows.Get(tx.Sender()) == nil {
		originBalance := pool.dag.GetBalance(tx.Sender())
		pool.flows.ResetFlow(tx.Sender(), originBalance)
	}
	pool.flows.Add(tx)
	pool.tips.Add(tx)
	pool.txLookup.SwitchStatus(tx.GetTxHash(), TxStatusTip)

	log.WithField("tx", tx).Tracef("finished commit tx")
	return nil
}

func (pool *TxPool) isBadTx(tx *types.Tx) TxQuality {
	// check if the tx's parents exists and if is badtx
	for _, parentHash := range tx.Parents() {
		// check if tx in pool
		if pool.get(parentHash) != nil {
			if pool.getStatus(parentHash) == TxStatusBadTx {
				log.WithField("tx", tx).Tracef("bad tx, parent %s is bad tx", parentHash.String())
				return TxQualityIsBad
			}
			continue
		}
		// check if tx in dag
		if pool.dag.GetTx(parentHash) == nil {
			log.WithField("tx", tx).Tracef("fatal tx, parent %s is not exist", parentHash.String())
			return TxQualityIsFatal
		}
	}

	// check if nonce is duplicate
	txinpool := pool.flows.GetTxByNonce(tx.Sender(), tx.GetNonce())
	if txinpool != nil {
		if txinpool.GetTxHash() == tx.GetTxHash() {
			log.WithField("tx", tx).Error("duplicated tx in pool. Why received many times")
			return TxQualityIsFatal
		}
		log.WithField("tx", tx).WithField("existing", txinpool).Trace("bad tx, duplicate nonce found in pool")
		return TxQualityIsBad
	}
	txindag := pool.dag.GetTxByNonce(tx.Sender(), tx.GetNonce())
	if txindag != nil {
		if txindag.GetTxHash() == tx.GetTxHash() {
			log.WithField("tx", tx).Error("duplicated tx in dag. Why received many times")
		}
		log.WithField("tx", tx).WithField("existing", txindag).Trace("bad tx, duplicate nonce found in dag")
		return TxQualityIsFatal
	}

	// check if the tx itself has no conflicts with local ledger
	stateFrom := pool.flows.GetBalanceState(tx.Sender())
	if stateFrom == nil {
		originBalance := pool.dag.GetBalance(tx.Sender())
		stateFrom = NewBalanceState(originBalance)
	}
	// if tx's value is larger than its balance, return fatal.
	if tx.Value.Value.Cmp(stateFrom.OriginBalance().Value) > 0 {
		log.WithField("tx", tx).Tracef("fatal tx, tx's value larger than balance")
		return TxQualityIsFatal
	}
	// if ( the value that 'from' already spent )
	// 	+ ( the value that 'from' newly spent )
	// 	> ( balance of 'from' in db )
	totalspent := math.NewBigInt(0)
	if totalspent.Value.Add(stateFrom.spent.Value, tx.Value.Value).Cmp(
		stateFrom.originBalance.Value) > 0 {
		log.WithField("tx", tx).Tracef("bad tx, total spent larget than balance")
		return TxQualityIsBad
	}

	return TxQualityIsGood
}

// confirm pushes a batch of txs that confirmed by a sequencer to the dag.
func (pool *TxPool) confirm(seq *types.Sequencer) error {
	log.WithField("seq", seq).Trace("start confirm seq")

	// check if sequencer is correct
	checkErr := pool.isBadSeq(seq)
	if checkErr != nil {
		return checkErr
	}
	// get sequencer's unconfirmed elders
	elders, errElders := pool.seekElders(seq)
	if errElders != nil {
		return errElders
	}
	// verify the elders
	log.WithField("seq height", seq.Height).WithField("count", len(elders)).Info("tx being confirmed by seq")
	batch, err := pool.verifyConfirmBatch(seq, elders)
	if err != nil {
		log.WithField("error", err).Errorf("verifyConfirmBatch error: %v", err)
		return err
	}
	// push batch to dag
	if err := pool.dag.Push(batch); err != nil {
		log.WithField("error", err).Errorf("dag Push error: %v", err)
		return err
	}
	// solve conflicts of txs in pool
	pool.solveConflicts(elders)

	if pool.flows.Get(seq.Sender()) == nil {
		originBalance := pool.dag.GetBalance(seq.Sender())
		pool.flows.ResetFlow(seq.Sender(), originBalance)
	}
	pool.flows.Add(seq)
	pool.tips.Add(seq)
	pool.txLookup.SwitchStatus(seq.GetTxHash(), TxStatusTip)

	// notification
	for _, c := range pool.OnBatchConfirmed {
		c <- elders
	}
	for _, c := range pool.OnNewLatestSequencer {
		c <- true
	}

	log.WithField("seq height", seq.Height).WithField("seq", seq).Trace("finished confirm seq")
	return nil
}

// isBadSeq checks if a sequencer is correct.
func (pool *TxPool) isBadSeq(seq *types.Sequencer) error {
	// check if the nonce is duplicate
	seqindag := pool.dag.GetTxByNonce(seq.Sender(), seq.GetNonce())
	if seqindag != nil {
		return fmt.Errorf("bad seq,duplicate nonce %d found in dag", seq.GetNonce())
	}
	return nil
}

// seekElders finds all the unconfirmed elders of baseTx.
func (pool *TxPool) seekElders(baseTx types.Txi) (map[types.Hash]types.Txi, error) {
	batch := make(map[types.Hash]types.Txi)

	inSeekingPool := map[types.Hash]int{}
	seekingPool := []types.Hash{}
	for _, parentHash := range baseTx.Parents() {
		seekingPool = append(seekingPool, parentHash)
	}
	for len(seekingPool) > 0 {
		elderHash := seekingPool[0]
		seekingPool = seekingPool[1:]

		elder := pool.get(elderHash)
		if elder == nil {
			elder = pool.dag.GetTx(elderHash)
			if elder == nil {
				return nil, fmt.Errorf("can't find elder %s", elderHash.String())
			}
			continue
		}
		if batch[elder.GetTxHash()] == nil {
			batch[elder.GetTxHash()] = elder
		}
		for _, elderParentHash := range elder.Parents() {
			if _, in := inSeekingPool[elderParentHash]; !in {
				seekingPool = append(seekingPool, elderParentHash)
				inSeekingPool[elderParentHash] = 0
			}
		}
	}
	return batch, nil
}

// verifyConfirmBatch verifies if the elders are correct.
// If passes all verifications, it returns a batch for pushing to dag.
func (pool *TxPool) verifyConfirmBatch(seq *types.Sequencer, elders map[types.Hash]types.Txi) (*ConfirmBatch, error) {
	// statistics of the confirmation term.
	// sums up the related address' income and outcome values
	batch := map[types.Address]*BatchDetail{}
	for _, txi := range elders {
		// return error if a sequencer confirm a tx that has same nonce as itself.
		if txi.Sender() == seq.Sender() && txi.GetNonce() == seq.GetNonce() {
			return nil, fmt.Errorf("seq's nonce is the same as a tx it confirmed, nonce: %d, tx hash: %s",
				seq.GetNonce(), txi.GetTxHash().String())
		}
		switch tx := txi.(type) {
		case *types.Sequencer:
			break
		case *types.Tx:
			batchFrom, okFrom := batch[tx.From]
			if !okFrom {
				batchFrom = &BatchDetail{}
				batchFrom.TxList = NewTxList()
				batchFrom.Neg = math.NewBigInt(0)
				batchFrom.Pos = math.NewBigInt(0)
				batch[tx.From] = batchFrom
			}
			batchFrom.TxList.put(tx)
			batchFrom.Neg.Value.Add(batchFrom.Neg.Value, tx.Value.Value)

			batchTo, okTo := batch[tx.To]
			if !okTo {
				batchTo = &BatchDetail{}
				batchTo.TxList = NewTxList()
				batchTo.Neg = math.NewBigInt(0)
				batchTo.Pos = math.NewBigInt(0)
				batch[tx.To] = batchTo
			}
			batchTo.Pos.Value.Add(batchTo.Pos.Value, tx.Value.Value)
		}
	}

	// verify balance and nonce
	for addr, batchDetail := range batch {
		// check balance
		// if balance < outcome, then verify failed
		confirmedBalance := pool.dag.GetBalance(addr)
		if confirmedBalance.Value.Cmp(batchDetail.Neg.Value) < 0 {
			return nil, fmt.Errorf("the balance of addr %s is not enough", addr.String())
		}

		// check nonce order
		nonces := *batchDetail.TxList.keys
		if !(nonces.Len() > 0) {
			continue
		}
		if nErr := pool.verifyNonce(addr, &nonces, seq); nErr != nil {
			return nil, nErr
		}
	}

	// construct tx hashes
	var txhashes types.Hashes
	for _, hash := range pool.getHashOrder() {
		elder, in := elders[hash]
		if !in {
			continue
		}
		if elder.GetType() == types.TxBaseTypeNormal {
			txhashes = append(txhashes, hash)
		}
	}

	cb := &ConfirmBatch{}
	cb.Seq = seq
	cb.Batch = batch
	cb.TxHashes = &txhashes
	return cb, nil
}

// solveConflicts remove elders from txpool and reprocess
// all txs in the pool in order to make sure all txs are
// correct after seq confirmation.
func (pool *TxPool) solveConflicts(elders map[types.Hash]types.Txi) {
	// remove elders from pool.txLookUp.txs
	//
	// pool.txLookUp.remove() will try remove tx from txLookup.order
	// and it will call hash.Cmp() to check if two hashes are the same.
	// For a large amount of txs to remove, hash.Cmp() will cost a O(n^2)
	// complexity. To solve this, solveConflicts will just remove the tx
	// from txLookUp.txs, which is a map struct, and find out txs left
	// which in this case is stored in txsInPool. As for the txs should
	// be removed from pool, pool.clearall() will be called to remove all
	// the txs in the pool including pool.txLookUp.order.

	txsInPool := []types.Txi{}
	// remove elders from pool
	for elserHash := range elders {
		pool.txLookup.remove(elserHash, noRemove)
	}

	for _, hash := range pool.txLookup.getorder() {
		tx := pool.get(hash)
		if tx == nil {
			continue
		}
		// TODO sequencer is removed from txpool later when calling
		// pool.clearall() but not added back. Try figure out if this
		// will cause any problem.
		if tx.GetType() == types.TxBaseTypeSequencer {

			//log.WithField("found seq",tx).Warn("should never come here")

			continue
		}
		txsInPool = append(txsInPool, tx)
	}
	pool.clearAll()
	for _, tx := range txsInPool {
		log.WithField("tx", tx).Tracef("start rejudge")
		txEnv := &txEnvelope{
			tx:     tx,
			txType: TxTypeRejudge,
			status: TxStatusQueue,
		}
		pool.txLookup.Add(txEnv)
		pool.commit(tx.(*types.Tx))
	}
}

func (pool *TxPool) verifyNonce(addr types.Address, noncesP *nonceHeap, seq *types.Sequencer) error {
	sort.Sort(noncesP)
	nonces := *noncesP

	latestNonce, nErr := pool.dag.GetLatestNonce(addr)
	if (nErr != nil) && (nErr != types.ErrNonceNotExist) {
		return fmt.Errorf("get latest nonce err: %v", nErr)
	}
	if nErr == types.ErrNonceNotExist {
		if nonces[0] != uint64(0) {
			return fmt.Errorf("nonce %d is not zero when there is no nonce in db, addr: %s", nonces[0], addr.String())
		}
	} else {
		if nonces[0] != latestNonce+1 {
			return fmt.Errorf("nonce %d is not the next one of latest nonce %d, addr: %s", nonces[0], latestNonce, addr.String())
		}
	}

	for i := 1; i < nonces.Len(); i++ {
		if nonces[i] != nonces[i-1]+1 {
			return fmt.Errorf("nonce order mismatch, addr: %s, preNonce: %d, curNonce: %d", addr.String(), nonces[i-1], nonces[i])
		}
	}

	if seq.Sender().Hex() == addr.Hex() {
		if seq.GetNonce() != nonces[len(nonces)-1]+1 {
			return fmt.Errorf("seq's nonce is not the next nonce of confirm list, seq nonce: %d, latest nonce in confirm list: %d", seq.GetNonce(), nonces[len(nonces)-1])
		}
	}

	return nil
}

// reset clears the txs that conflicts with sequencer
func (pool *TxPool) reset() {
	// TODO
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

type txLookUp struct {
	order []types.Hash
	txs   map[types.Hash]*txEnvelope
	mu    sync.RWMutex
}

func newTxLookUp() *txLookUp {
	return &txLookUp{
		order: []types.Hash{},
		txs:   make(map[types.Hash]*txEnvelope),
	}
}

// Get tx from txLookUp by hash
func (t *txLookUp) Get(h types.Hash) types.Txi {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.get(h)
}
func (t *txLookUp) get(h types.Hash) types.Txi {
	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.tx
	}
	return nil
}

// Add tx into txLookUp
func (t *txLookUp) Add(txEnv *txEnvelope) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.add(txEnv)
}
func (t *txLookUp) add(txEnv *txEnvelope) {
	if _, ok := t.txs[txEnv.tx.GetTxHash()]; ok {
		return
	}

	t.order = append(t.order, txEnv.tx.GetTxHash())
	t.txs[txEnv.tx.GetTxHash()] = txEnv
}

// Remove tx from txLookUp
func (t *txLookUp) Remove(h types.Hash, removeType hashOrderRemoveType) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.remove(h, removeType)
}

func (t *txLookUp) remove(h types.Hash, removeType hashOrderRemoveType) {
	switch removeType {
	case noRemove:
		log.Trace("no remove")
	case removeFromFront:
		for i, hash := range t.order {
			if hash.Cmp(h) == 0 {
				t.order = append(t.order[:i], t.order[i+1:]...)
				break
			}
		}

	case removeFromEnd:
		for i := len(t.order) - 1; i >= 0; i-- {
			hash := t.order[i]
			if hash.Cmp(h) == 0 {
				t.order = append(t.order[:i], t.order[i+1:]...)
				break
			}
		}
	default:
		panic("unknown remove type")
	}
	delete(t.txs, h)
}

// RemoveTx removes tx from txLookUp.txs only, ignore the order.
func (t *txLookUp) RemoveTxFromMapOnly(h types.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.removeTxFromMapOnly(h)
}
func (t *txLookUp) removeTxFromMapOnly(h types.Hash) {
	delete(t.txs, h)
}

// RemoveByIndex removes a tx by its order index
func (t *txLookUp) RemoveByIndex(i int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.removeByIndex(i)
}

func (t *txLookUp) removeByIndex(i int) {
	if (i < 0) || (i >= len(t.order)) {
		return
	}
	hash := t.order[i]
	t.order = append(t.order[:i], t.order[i+1:]...)
	delete(t.txs, hash)
}

// Order returns hash list of txs in pool, ordered by the time
// it added into pool.
func (t *txLookUp) GetOrder() []types.Hash {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.getorder()
}

func (t *txLookUp) getorder() []types.Hash {
	return t.order
}

// Order returns hash list of txs in pool, ordered by the time
// it added into pool.
func (t *txLookUp) ResetOrder() {
	t.mu.RLock()
	defer t.mu.RUnlock()
	t.resetOrder()
}

func (t *txLookUp) resetOrder() {
	t.order = nil
}

// Count returns the total number of txs in txLookUp
func (t *txLookUp) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.count()
}

func (t *txLookUp) count() int {
	return len(t.txs)
}

// Stats returns the count of tips, bad txs, pending txs in txlookup
func (t *txLookUp) Stats() (int, int, int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.stats()
}
func (t *txLookUp) stats() (int, int, int) {
	tips, badtx, pending := 0, 0, 0
	for _, v := range t.txs {
		if v.status == TxStatusTip {
			tips += 1
		} else if v.status == TxStatusBadTx {
			badtx += 1
		} else if v.status == TxStatusPending {
			pending += 1
		}
	}
	return tips, badtx, pending
}

// Status returns the status of a tx
func (t *txLookUp) Status(h types.Hash) TxStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.status(h)
}
func (t *txLookUp) status(h types.Hash) TxStatus {
	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.status
	}
	return TxStatusNotExist
}

// SwitchStatus switches the tx status
func (t *txLookUp) SwitchStatus(h types.Hash, status TxStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.switchstatus(h, status)
}
func (t *txLookUp) switchstatus(h types.Hash, status TxStatus) {
	if txEnv := t.txs[h]; txEnv != nil {
		txEnv.status = status
	}
}
