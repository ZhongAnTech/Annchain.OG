package core

import (
	"container/list"
	"errors"
	"fmt"
	"sort"
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

var (
	ErrDupilcate = errors.New("Duplicate tx found in txlookup")
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

	queue    chan *txEvent // queue stores txs that need to validate later
	tips     *TxMap        // tips stores all the tips
	badtxs   *TxMap
	pendings *TxMap
	flows    *AccountFlows
	txLookup *txLookUp // txLookUp stores all the txs for external query

	close chan struct{}

	mu sync.RWMutex
	wg sync.WaitGroup // for TxPool Stop()

	OnNewTxReceived  []chan types.Txi                // for notifications of new txs.
	OnBatchConfirmed []chan map[types.Hash]types.Txi // for notifications of confirmation.
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
		conf:             conf,
		dag:              d,
		queue:            make(chan *txEvent, conf.QueueSize),
		tips:             NewTxMap(),
		badtxs:           NewTxMap(),
		pendings:         NewTxMap(),
		flows:            NewAccountFlows(),
		txLookup:         newTxLookUp(),
		close:            make(chan struct{}),
		OnNewTxReceived:  []chan types.Txi{},
		OnBatchConfirmed: []chan map[types.Hash]types.Txi{},
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

func DefaultTxPoolCOnfig() TxPoolConfig {
	config := TxPoolConfig{
		QueueSize:     100,
		TipsSize:      1000,
		ResetDuration: 10,
		TxVerifyTime:  2,
		TxValidTime:   100,
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
	return pool.txLookup.Stats()
}

// Get get a transaction or sequencer according to input hash,
// if tx not exists return nil
func (pool *TxPool) Get(hash types.Hash) types.Txi {
	return pool.txLookup.Get(hash)
}

// GetByNonce get a tx or sequencer from account flows by sender's address and tx's nonce.
func (pool *TxPool) GetByNonce(addr types.Address, nonce uint64) types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

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
	return pool.txLookup.Status(hash)
}

func (pool *TxPool) RegisterOnNewTxReceived(c chan types.Txi) {
	pool.OnNewTxReceived = append(pool.OnNewTxReceived, c)
}

// // for OG visualizer
// func (pool *TxPool) GetRandomTips(n int) (v []types.Txi) {
// 	pool.mu.RLock()
// 	defer pool.mu.RUnlock()

// 	tips := pool.tips.GetAllValues()
// 	pendings := pool.pendings.GetAllValues()
// 	alltxs := append(tips, pendings...)

// 	indices := generateRandomIndices(n, len(alltxs))
// 	for _, i := range indices {
// 		v = append(v, alltxs[i])
// 	}
// 	return v
// }

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
				err = ErrDupilcate
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
// tip pool, those tips will be removed from tips but stored in pending.
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

	// move parents to pending
	for _, pHash := range tx.Parents() {
		status := pool.GetStatus(pHash)
		if status != TxStatusTip {
			log.WithField("parent", pHash).WithField("tx", tx).Debugf("parent is not a tip")
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
			pool.txLookup.Remove(pHash)
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
	// check if the nonce is duplicate
	txinpool := pool.flows.GetTxByNonce(tx.Sender(), tx.GetNonce())
	if txinpool != nil {
		return bad
	}
	txindag := pool.dag.GetTxByNonce(tx.Sender(), tx.GetNonce())
	if txindag != nil {
		return bad
	}
	// check if the tx itself has no conflicts with local ledger
	stateFrom := pool.flows.GetBalanceState(tx.Sender())
	if stateFrom == nil {
		originBalance := pool.dag.GetBalance(tx.Sender())
		stateFrom = NewBalanceState(originBalance)
	}
	// if ( the value that 'from' already spent )
	// 	+ ( the value that 'from' newly spent )
	// 	> ( balance of 'from' in db )
	totalspent := math.NewBigInt(0)
	if totalspent.Value.Add(stateFrom.spent.Value, tx.Value.Value).Cmp(
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
	log.WithField("seq id", seq.Id).WithField("count", len(elders)).Warn("tx being confirmed by seq")
	batch, err := pool.verifyConfirmBatch(seq, elders)
	if err != nil {
		log.WithField("error", err).Warnf("verifyConfirmBatch error: %v", err)
		return err
	}
	// push batch to dag
	if err := pool.dag.Push(batch); err != nil {
		log.WithField("error", err).Warnf("dag Push error: %v", err)
		return err
	}
	// remove elders from pool
	for _, elder := range elders {
		status := pool.GetStatus(elder.GetTxHash())
		if status == TxStatusBadTx {
			pool.badtxs.Remove(elder.GetTxHash())
		}
		if status == TxStatusTip {
			pool.tips.Remove(elder.GetTxHash())
		}
		if status == TxStatusPending {
			pool.pendings.Remove(elder.GetTxHash())
		}
		pool.flows.Confirm(elder)
		pool.txLookup.Remove(elder.GetTxHash())
	}

	pool.flows.Add(seq)

	pool.tips.Add(seq)
	pool.txLookup.SwitchStatus(seq.GetTxHash(), TxStatusTip)

	log.WithField("seq id", seq.Id).WithField("seq", seq).Debug("finished confirm seq")
	// notification
	for _, c := range pool.OnBatchConfirmed {
		c <- elders
	}

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
			}
		}
	}
	return batch
}

func (pool *TxPool) verifyConfirmBatch(seq *types.Sequencer, elders map[types.Hash]types.Txi) (*ConfirmBatch, error) {
	// statistics of the confirmation term.
	// sums up the related address' income and outcome values
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
			txhashes = append(txhashes, tx.GetTxHash())
		}
	}
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
		if nErr := pool.verifyNonce(addr, &nonces); nErr != nil {
			return nil, nErr
		}
	}
	//reverse the txhashes to keeep partial order , accelerate processing speed
	n := len(txhashes)
	for i := 0; i < n/2; i++ {
		txhashes[i], txhashes[n-1-i] = txhashes[n-i-1], txhashes[i]
	}

	cb := &ConfirmBatch{}
	cb.Seq = seq
	cb.Batch = batch
	cb.TxHashes = &txhashes
	return cb, nil
}

func (pool *TxPool) verifyNonce(addr types.Address, noncesP *nonceHeap) error {
	sort.Sort(noncesP)
	nonces := *noncesP

	has, hErr := pool.dag.HasLatestNonce(addr)
	if hErr != nil {
		return fmt.Errorf("check nonce in db err: %v", hErr)
	}
	if has {
		latestNonce, nErr := pool.dag.GetLatestNonce(addr)
		if nErr != nil {
			return fmt.Errorf("get latest nonce err: %v", nErr)
		}
		if nonces[0] != latestNonce+1 {
			return fmt.Errorf("nonce %d is not the next one of latest nonce %d", nonces[0], latestNonce)
		}
	} else {
		if nonces[0] != uint64(0) {
			return fmt.Errorf("nonce %d is not zero when there is no nonce in db", nonces[0])
		}
	}

	for i := 1; i < nonces.Len(); i++ {
		if nonces[i] != nonces[i-1]+1 {
			return fmt.Errorf("nonce order mismatch, addr: %s, preNonce: %d, curNonce: %d", addr.String(), nonces[i-1], nonces[i])
		}
	}

	return nil
}

// reset clears the tips that haven't been proved for a long time
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
	txs map[types.Hash]*txEnvelope
	mu  sync.RWMutex
}

func newTxLookUp() *txLookUp {
	return &txLookUp{
		txs: make(map[types.Hash]*txEnvelope),
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
	t.txs[txEnv.tx.GetTxHash()] = txEnv
}

// Remove tx from txLookUp
func (t *txLookUp) Remove(h types.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.remove(h)
}
func (t *txLookUp) remove(h types.Hash) {
	delete(t.txs, h)
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


func ( pool *TxPool)IsDupicateErr(err error) bool {
	return err == ErrDupilcate
}