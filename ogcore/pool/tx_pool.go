// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package pool

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/ledger"
	"github.com/annchain/OG/ogcore/state"
	"github.com/sirupsen/logrus"

	"math/rand"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/annchain/OG/common/math"
)

type TxType uint8

const (
	TxTypeGenesis TxType = iota
	TxTypeLocal
	TxTypeRemote
	TxTypeRejudge
)

type TxStatus uint8

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

type TxQuality uint8

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
	debuglog.NodeLogger
	EventBus eventbus.EventBus
	Config   TxPoolConfig
	Dag      ILedger

	queue    chan *txEnvelope // queue stores txs that need to validate later
	tips     *TxMap           // tips stores all the tips
	badtxs   *TxMap
	pendings *TxMap
	flows    *AccountFlows
	txLookup *txLookUp // txLookUp stores all the txs for external query
	cached   *cachedConfirm

	//quit chan struct{}

	mu sync.RWMutex
	wg sync.WaitGroup // for TxPool Stop()

	onTxiGetInPool         map[channelName]chan types.Txi   // for notifications of new txs.
	onConsensusTXConfirmed []chan map[types2.Hash]types.Txi // for notifications of  consensus tx confirmation.
	txNum                  uint32
	maxWeight              uint64
	//confirmStatus          *ConfirmStatus
}

func (pool *TxPool) HandlerDescription(et eventbus.EventType) string {
	switch et {
	case events.NewTxiDependencyFulfilledEventType:
		return "AddTxToPool"
	default:
		return "N/A"
	}
}

func (pool *TxPool) HandleEvent(ev eventbus.Event) {
	if pool.queue == nil {
		panic("not initialized.")
	}

	switch ev.GetEventType() {
	case events.NewTxiDependencyFulfilledEventType:
		evt := ev.(*events.NewTxiDependencyFulfilledEvent)
		// TODO: verify if nofeedback is still needded
		err := pool.AddRemoteTx(evt.Txi, true)
		utilfuncs.PanicIfError(err, "should not have any error now")
	default:
		pool.Logger.WithField("type", ev.GetEventType()).Warn("event type not supported")
	}
}

func (pool *TxPool) InitDefault() {
	pool.queue = make(chan *txEnvelope, pool.Config.QueueSize)
	pool.tips = NewTxMap()
	pool.badtxs = NewTxMap()
	pool.pendings = NewTxMap()
	pool.flows = NewAccountFlows()
	pool.txLookup = newTxLookUp()
	//pool.quit = make(chan struct{})
	pool.onTxiGetInPool = make(map[channelName]chan types.Txi)
	//pool.onBatchConfirmed = []chan map[common.Hash]types.Txi{}
	//pool.onConsensusTXConfirmed = []chan map[common.Hash]types.Txi{}
	//confirmStatus:          &ConfirmStatus{RefreshTime: time.Minute * time.Duration(Config.ConfirmStatusRefreshTime)},

}

func (pool *TxPool) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"queue":      len(pool.queue),
		"event":      len(pool.onTxiGetInPool),
		"txlookup":   len(pool.txLookup.txs),
		"tips":       len(pool.tips.txs),
		"badtxs":     len(pool.badtxs.txs),
		"latest_seq": int(pool.Dag.GetHeight()),
		"pendings":   len(pool.pendings.txs),
		"flows":      len(pool.flows.afs),
		"hashordr":   len(pool.txLookup.order),
	}
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
	//ConfirmStatusRefreshTime int //minute
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

type txEnvelope struct {
	tx         types.Txi
	txType     TxType
	status     TxStatus
	noFeedBack bool
}

// Start begin the txpool sevices
//func (pool *TxPool) Start() {
//	logrus.Infof("TxPool Start")
//	//goroutine.New(pool.loop)
//}

// Stop stops all the txpool sevices
//func (pool *TxPool) Stop() {
//	close(pool.quit)
//	pool.wg.Wait()
//
//	logrus.Infof("TxPool Stopped")
//}

func (pool *TxPool) Init(genesis *types.Sequencer) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	genesisEnvelope := &txEnvelope{}
	genesisEnvelope.tx = genesis
	genesisEnvelope.status = TxStatusTip
	genesisEnvelope.txType = TxTypeGenesis
	pool.txLookup.Add(genesisEnvelope)
	pool.tips.Add(genesis)

	pool.Logger.Infof("TxPool finish init")
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
func (pool *TxPool) Get(hash types2.Hash) types.Txi {
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

func (pool *TxPool) get(hash types2.Hash) types.Txi {
	return pool.txLookup.Get(hash)
}

func (pool *TxPool) Has(hash types2.Hash) bool {
	return pool.Get(hash) != nil
}

func (pool *TxPool) IsLocalHash(hash types2.Hash) bool {
	if pool.Has(hash) {
		return true
	}
	if pool.Dag.IsTxExists(hash) {
		return true
	}
	return false
}

// GetHashOrder returns a hash list of txs in pool, ordered by the
// time that txs added into pool.
func (pool *TxPool) GetHashOrder() types2.Hashes {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getHashOrder()
}

func (pool *TxPool) getHashOrder() types2.Hashes {
	return pool.txLookup.GetOrder()
}

// GetByNonce get a tx or sequencer from account flows by sender's address and tx's nonce.
func (pool *TxPool) GetByNonce(addr common.Address, nonce uint64) types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getByNonce(addr, nonce)
}
func (pool *TxPool) getByNonce(addr common.Address, nonce uint64) types.Txi {
	return pool.flows.GetTxByNonce(addr, nonce)
}

// GetLatestNonce get the latest nonce of an address
func (pool *TxPool) GetLatestNonce(addr common.Address) (uint64, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.flows.GetLatestNonce(addr)
}

// GetStatus gets the current status of a tx
func (pool *TxPool) GetStatus(hash types2.Hash) TxStatus {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getStatus(hash)
}

func (pool *TxPool) getStatus(hash types2.Hash) TxStatus {
	return pool.txLookup.Status(hash)
}

type channelName struct {
	name   string
	allMsg bool
}

func (c channelName) String() string {
	return c.name
}

func (pool *TxPool) RegisterOnNewTxReceived(c chan types.Txi, chanName string, allTx bool) {
	pool.Logger.Tracef("RegisterOnNewTxReceived with chan: %s ,all %v", chanName, allTx)
	chName := channelName{chanName, allTx}
	pool.onTxiGetInPool[chName] = c
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
func (pool *TxPool) GetAllTips() map[types2.Hash]types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.tips.txs
}

// AddLocalTx adds a tx to txpool if it is valid, note that if success it returns nil.
// AddLocalTx only process tx that sent by local node.
func (pool *TxPool) AddLocalTx(tx types.Txi, noFeedBack bool) error {
	return pool.addTx(tx, TxTypeLocal, noFeedBack)
}

// AddLocalTxs adds a list of txs to txpool if they are valid. It returns
// the process result of each tx with an error list. AddLocalTxs only process
// txs that sent by local node.
func (pool *TxPool) AddLocalTxs(txs []types.Txi, noFeedBack bool) []error {
	result := make([]error, len(txs))
	for _, tx := range txs {
		result = append(result, pool.addTx(tx, TxTypeLocal, noFeedBack))
	}
	return result
}

// AddRemoteTx adds a tx to txpool if it is valid. AddRemoteTx only process tx
// sent by remote nodes, and will hold extra functions to prevent from ddos
// (large amount of invalid tx sent from one node in a short time) attack.
func (pool *TxPool) AddRemoteTx(tx types.Txi, noFeedBack bool) error {
	return pool.addTx(tx, TxTypeRemote, noFeedBack)
}

// AddRemoteTxs works as same as AddRemoteTx but processes a list of txs
func (pool *TxPool) AddRemoteTxs(txs []types.Txi, noFeedBack bool) []error {
	result := make([]error, len(txs))
	for _, tx := range txs {
		result = append(result, pool.addTx(tx, TxTypeRemote, noFeedBack))
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
	status := pool.getStatus(tx.GetHash())
	if status == TxStatusBadTx {
		pool.badtxs.Remove(tx.GetHash())
	}
	if status == TxStatusTip {
		pool.tips.Remove(tx.GetHash())
		pool.flows.Remove(tx)
	}
	if status == TxStatusPending {
		pool.pendings.Remove(tx.GetHash())
		pool.flows.Remove(tx)
	}
	pool.txLookup.Remove(tx.GetHash(), removeType)
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

//func (pool *TxPool) loop() {
//	defer logrus.Tracef("TxPool.loop() terminates")
//
//	pool.wg.Add(1)
//	defer pool.wg.Done()
//
//	//resetTimer := time.NewTicker(time.Duration(pool.Config.ResetDuration) * time.Second)
//
//	for {
//		select {
//		case <-pool.quit:
//			logrus.Info("pool got quit signal quiting...")
//			return
//
//		case txEvent := <-pool.queue:
//
//			//txEvent.callbackChan <- err
//
//			//case <-resetTimer.C:
//			//pool.reset()
//		}
//	}
//}

// addTx adds tx to the pool queue and No async[wait to become tip after validation.]
func (pool *TxPool) addTx(tx types.Txi, senderType TxType, noFeedBack bool) error {
	pool.Logger.WithField("noFeedBack", noFeedBack).WithField("tx", tx).Tracef("start addTx, tx parents: %s", tx.GetParents().String())

	txEvent := &txEnvelope{
		tx:         tx,
		txType:     senderType,
		status:     TxStatusQueue,
		noFeedBack: noFeedBack,
	}

	pool.Logger.WithField("tx", txEvent.tx).Trace("get tx from queue")

	var err error
	//tx := txEvent.tx
	// check if tx is duplicate
	if pool.get(tx.GetHash()) != nil {
		pool.Logger.WithField("tx", tx).Warn("Duplicate tx found in txlookup")
		// use event bus
		//txEvent.callbackChan <- types.ErrDuplicateTx
		return nil
	}
	pool.mu.Lock()
	pool.txLookup.Add(txEvent)

	// form weight
	parentMaxWeight := uint64(0)
	for _, parentHash := range tx.GetParents() {
		parentTx := pool.get(parentHash)
		if parentTx == nil {
			parentTx = pool.Dag.GetTx(parentHash)
			if parentTx == nil {
				// not possible, since all tx sent from buffer should be completely fulfilled
				pool.Logger.WithField("tx", tx).Error("failed to find parents of tx")
				return errors.New("failed to find parents of tx")
			}
		}
		parentMaxWeight = math.MaxUint64(parentTx.GetWeight(), parentMaxWeight)
	}
	tx.SetWeight(parentMaxWeight + 1)

	switch tx := tx.(type) {
	case *types.Sequencer:
		err = pool.Confirm(tx)
		if err != nil {
			pool.txLookup.Remove(txEvent.tx.GetHash(), removeFromEnd)
		} else {
			atomic.StoreUint32(&pool.txNum, 0)
			maxWeight := atomic.LoadUint64(&pool.maxWeight)
			if maxWeight < tx.GetWeight() {
				atomic.StoreUint64(&pool.maxWeight, tx.GetWeight())
			}
		}
	default:
		err = pool.commit(tx)
		//if err is not nil , item removed inside commit
		if err == nil {
			atomic.AddUint32(&pool.txNum, 1)
			maxWeight := atomic.LoadUint64(&pool.maxWeight)
			if maxWeight < tx.GetWeight() {
				atomic.StoreUint64(&pool.maxWeight, tx.GetWeight())
			}
			tx.SetHeight(pool.Dag.LatestSequencer().Height + 1) //temporary height ,will be re write after Confirm
		}

	}
	pool.mu.Unlock()

	// TODO: Use event
	pool.Logger.WithField("tx", tx).Trace("successfully added tx to txPool")

	pool.EventBus.Route(&events.NewTxReceivedInPoolEvent{
		Tx: tx,
	})

	//te := &txEnvelope{
	//callbackChan: make(chan error),
	//}

	//if normalTx, ok := tx.(*types.Txi); ok {
	//	normalTx.Setconfirm()
	//	pool.confirmStatus.AddTxNum()
	//}

	// must be a sync method to avoid middle state of tx.
	// pool.queue <- txEnv

	// waiting for callback
	//select {
	//case err := <-te.callbackChan:
	//	if err != nil {
	//		return err
	//	}
	//	// notify all subscribers of newTxEvent
	//	for name, subscriber := range pool.onTxiGetInPool {
	//		logrus.WithField("tx", tx).Trace("notify subscriber: ", name)
	//		if !noFeedBack || name.allMsg {
	//			subscriber <- tx
	//		}
	//	}
	//}
	return nil
}

// commit commits tx to tips pool. commit() checks if this tx is bad tx and moves
// bad tx to badtx list other than tips list. If this tx proves any txs in the
// tip pool, those tips will be removed from tips but stored in pending.
func (pool *TxPool) commit(tx types.Txi) error {
	pool.Logger.WithField("tx", tx).Trace("start commit tx")

	// check tx's quality.
	txquality := pool.isBadTx(tx)
	if txquality == TxQualityIsFatal {
		pool.remove(tx, removeFromEnd)
		tx.SetValid(false)
		pool.Logger.WithField("tx", tx).Debug("set invalid")
		return fmt.Errorf("tx is surely incorrect to commit, hash: %s", tx.GetHash())
	}
	if txquality == TxQualityIsBad {
		pool.Logger.Tracef("bad tx: %s", tx)
		pool.badtxs.Add(tx)
		pool.txLookup.SwitchStatus(tx.GetHash(), TxStatusBadTx)
		return nil
	}

	// move parents to pending
	for _, pHash := range tx.GetParents() {
		status := pool.getStatus(pHash)
		if status != TxStatusTip {
			pool.Logger.WithField("parent", pHash).WithField("tx", tx).
				Tracef("parent is not a tip")
			continue
		}
		parent := pool.tips.Get(pHash)
		if parent == nil {
			pool.Logger.WithField("parent", pHash).WithField("tx", tx).
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
	//if tx.GetType() != types.TxBaseTypeArchive {
	//	// add tx to pool
	//	if pool.flows.Get(tx.Sender()) == nil {
	//		pool.flows.ResetFlow(tx.Sender(), state.NewBalanceSet())
	//	}
	//}
	if tx.GetType() == types.TxBaseTypeTx {
		txn := tx.(*types.Tx)
		pool.flows.GetBalanceState(txn.Sender(), txn.TokenId)
	}
	pool.flows.Add(tx)
	pool.Logger.WithField("flows.afs", pool.flows.afs).Warn("pool commit")
	pool.tips.Add(tx)
	pool.txLookup.SwitchStatus(tx.GetHash(), TxStatusTip)

	// TODO delete this line later.
	if pool.Logger.GetLevel() >= logrus.TraceLevel {
		pool.Logger.WithField("tx", tx).WithField("status", pool.getStatus(tx.GetHash())).Tracef("finished commit tx")
	}
	return nil
}

func (pool *TxPool) isBadTx(tx types.Txi) TxQuality {
	// check if the tx's parents exists and if is badtx
	for _, parentHash := range tx.GetParents() {
		// check if tx in pool
		if pool.get(parentHash) != nil {
			if pool.getStatus(parentHash) == TxStatusBadTx {
				pool.Logger.WithField("tx", tx).Tracef("bad tx, parent %s is bad tx", parentHash)
				return TxQualityIsBad
			}
			continue
		}
		// check if tx in Dag
		if pool.Dag.GetTx(parentHash) == nil {
			pool.Logger.WithField("tx", tx).Tracef("fatal tx, parent %s is not exist", parentHash)
			return TxQualityIsFatal
		}
	}

	//if tx.GetType() == types.TxBaseTypeArchive {
	//	return TxQualityIsGood
	//}

	// check if nonce is duplicate
	txinpool := pool.flows.GetTxByNonce(tx.Sender(), tx.GetNonce())
	if txinpool != nil {
		if txinpool.GetHash() == tx.GetHash() {
			pool.Logger.WithField("tx", tx).Error("duplicated tx in pool. Why received many times")
			return TxQualityIsFatal
		}
		pool.Logger.WithField("tx", tx).WithField("existing", txinpool).Trace("bad tx, duplicate nonce found in pool")
		return TxQualityIsBad
	}
	txindag := pool.Dag.GetTxByNonce(tx.Sender(), tx.GetNonce())
	if txindag != nil {
		if txindag.GetHash() == tx.GetHash() {
			pool.Logger.WithField("tx", tx).Error("duplicated tx in Dag. Why received many times")
		}
		pool.Logger.WithField("tx", tx).WithField("existing", txindag).Trace("bad tx, duplicate nonce found in Dag")
		return TxQualityIsFatal
	}

	latestNonce, e := pool.flows.GetLatestNonce(tx.Sender())
	pool.Logger.WithField("nonce", latestNonce).Info("latestNonce")

	if e == nil {
		if tx.GetNonce() != latestNonce+1 {
			pool.Logger.WithField("should be ", latestNonce+1).WithField("tx", tx).Error("nonce err")
			return TxQualityIsFatal
		}
	} else {
		latestNonce, nErr := pool.Dag.GetLatestNonce(tx.Sender())
		if nErr != nil {
			pool.Logger.Errorf("get latest nonce err: %v", nErr)
			return TxQualityIsFatal
		}
		if tx.GetNonce() != latestNonce+1 {
			pool.Logger.WithFields(logrus.Fields{
				"given":  tx.GetNonce(),
				"should": latestNonce + 1,
				"tx":     tx,
			}).Warn("bad tx. nonce should be sequential")
			return TxQualityIsFatal
		}
	}

	switch tx := tx.(type) {
	case *types.Tx:
		// check if the tx itself has no conflicts with local ledger
		stateFrom := pool.flows.GetBalanceState(tx.Sender(), tx.TokenId)
		if stateFrom == nil {
			originBalance := pool.Dag.GetBalance(tx.Sender(), tx.TokenId)
			stateFrom = NewBalanceState(originBalance)
		}

		// if tx's value is larger than its balance, return fatal.
		if tx.Value.Value.Cmp(stateFrom.OriginBalance().Value) > 0 {
			pool.Logger.WithField("tx", tx).Tracef("fatal tx, tx's value larger than balance")
			return TxQualityIsFatal
		}
		// if ( the value that 'from' already spent )
		// 	+ ( the value that 'from' newly spent )
		// 	> ( balance of 'from' in db )
		totalspent := math.NewBigInt(0)
		if totalspent.Value.Add(stateFrom.spent.Value, tx.Value.Value).Cmp(
			stateFrom.originBalance.Value) > 0 {
			pool.Logger.WithField("tx", tx).Tracef("bad tx, total spent larget than balance")
			return TxQualityIsBad
		}
	//case *archive.ActionTx:
	//	if tx.Action == archive.ActionTxActionIPO {
	//		//actionData := tx.ActionData.(*tx_types.PublicOffering)
	//		//actionData.TokenId = pool.Dag.GetLatestTokenId()
	//	}
	//	if tx.Action == archive.ActionTxActionSPO {
	//		actionData := tx.ActionData.(*archive.PublicOffering)
	//		if actionData.TokenId == 0 {
	//			logrus.WithField("tx ", tx).Warn("og token is disabled for publishing")
	//			return TxQualityIsFatal
	//		}
	//		token := pool.Dag.GetToken(actionData.TokenId)
	//		if token == nil {
	//			logrus.WithField("tx ", tx).Warn("token not found")
	//			return TxQualityIsFatal
	//		}
	//		if !token.ReIssuable {
	//			logrus.WithField("tx ", tx).Warn("token is disabled for second publishing")
	//			return TxQualityIsFatal
	//		}
	//		if token.Destroyed {
	//			logrus.WithField("tx ", tx).Warn("token is destroyed already")
	//			return TxQualityIsFatal
	//		}
	//		if token.Issuer != tx.Sender() {
	//			logrus.WithField("token ", token).WithField("you address", tx.Sender()).Warn("you have no authority to second publishing for this token")
	//			return TxQualityIsFatal
	//		}
	//	}
	//	if tx.Action == archive.ActionTxActionDestroy {
	//		actionData := tx.ActionData.(*archive.PublicOffering)
	//		if actionData.TokenId == 0 {
	//			logrus.WithField("tx ", tx).Warn("og token is disabled for withdraw")
	//			return TxQualityIsFatal
	//		}
	//		token := pool.Dag.GetToken(actionData.TokenId)
	//		if token == nil {
	//			logrus.WithField("tx ", tx).Warn("token not found")
	//			return TxQualityIsFatal
	//		}
	//		if token.Destroyed {
	//			logrus.WithField("tx ", tx).Warn("token is destroyed already")
	//			return TxQualityIsFatal
	//		}
	//		if token.Issuer != tx.Sender() {
	//			logrus.WithField("tx ", tx).WithField("token ", token).WithField("you address", tx.Sender()).Warn("you have no authority to second publishing for this token")
	//			return TxQualityIsFatal
	//		}
	//	}
	//case *campaign.Campaign:
	//	// TODO
	//case *campaign.TermChange:
	//	// TODO
	default:
		// TODO
	}

	return TxQualityIsGood
}

// PreConfirm simulates the Confirm process of a sequencer and store the related data
// into pool.cached. Once a real sequencer with same hash comes, reload cached data without
// any more calculates.
func (pool *TxPool) PreConfirm(seq *types.Sequencer) (hash types2.Hash, err error) {
	//TODO , exists a panic bug in statedb commit , fix later
	//and recover this later
	err = pool.IsBadSeq(seq)
	if err != nil {
		return types2.Hash{}, err
	}
	return types2.Hash{}, nil
	// disabled.
	//pool.mu.Lock()
	//defer pool.mu.Unlock()
	//
	//if seq.GetHeight() <= pool.Dag.GetHeight() {
	//	return common.Hash{}, fmt.Errorf("the height of seq to pre-Confirm is lower than "+
	//		"the latest seq in Dag. get height: %d, latest: %d", seq.GetHeight(), pool.Dag.GetHeight())
	//}
	//elders, batch, err := pool.confirmHelper(seq)
	//if err != nil {
	//	logrus.WithField("error", err).Errorf("Confirm error: %v", err)
	//	return common.Hash{}, err
	//}
	//rootHash, err := pool.Dag.PrePush(batch)
	//if err != nil {
	//	return common.Hash{}, err
	//}
	//pool.cached = newCachedConfirm(seq, rootHash, batch.Txs, elders)
	//return rootHash, nil
}

// Confirm pushes a batch of txs that confirmed by a sequencer to the Dag.
func (pool *TxPool) Confirm(seq *types.Sequencer) error {
	pool.Logger.WithField("seq", seq).Trace("start Confirm seq")

	var err error
	var batch *ledger.ConfirmBatch
	var elders map[types2.Hash]types.Txi
	if pool.cached != nil && seq.GetHash() == pool.cached.SeqHash() {
		batch = &ledger.ConfirmBatch{
			Seq: seq,
			Txs: pool.cached.Txs(),
		}
		elders = pool.cached.elders
	} else {
		elders, batch, err = pool.confirmHelper(seq)
		if err != nil {
			pool.Logger.WithField("error", err).Errorf("Confirm error: %v", err)
			return err
		}
	}
	pool.cached = nil

	err = pool.PushBatch(batch)
	if err != nil {
		return err
	}

	// notification
	pool.EventBus.Route(&events.SequencerBatchConfirmedEvent{Elders: elders})

	pool.EventBus.Route(&events.SequencerConfirmedEvent{Sequencer: seq})

	//for _, c := range pool.onBatchConfirmed {
	//	if status.NodeStopped {
	//		break
	//	}
	//	c <- elders
	//}
	//for _, c := range pool.onNewLatestSequencer {
	//	if status.NodeStopped {
	//		break
	//	}
	//	c <- true
	//}

	pool.Logger.WithField("seq height", seq.Height).WithField("seq", seq).Trace("finished Confirm seq")
	return nil
}

func (pool *TxPool) PushBatch(batch *ledger.ConfirmBatch) error {
	// push batch to Dag
	if err := pool.Dag.Push(batch); err != nil {
		pool.Logger.WithField("error", err).Errorf("Dag Push error: %v", err)
		return err
	}

	//for _, tx := range batch.Txs {
	//	if normalTx, ok := tx.(*types.Txi); ok {
	//		pool.confirmStatus.AddConfirm(normalTx.GetConfirm())
	//	}
	//}

	seq := batch.Seq

	// solve conflicts of txs in pool
	pool.solveConflicts(batch)
	// add seq to txpool
	if pool.flows.Get(seq.Sender()) == nil {
		pool.flows.ResetFlow(seq.Sender(), state.NewBalanceSet())
	}
	pool.flows.Add(seq)
	pool.tips.Add(seq)
	pool.txLookup.Add(&txEnvelope{tx: seq})
	pool.txLookup.SwitchStatus(seq.GetHash(), TxStatusTip)
	return nil
}

func (pool *TxPool) confirmHelper(seq *types.Sequencer) (map[types2.Hash]types.Txi, *ledger.ConfirmBatch, error) {
	// check if sequencer is correct
	checkErr := pool.isBadSeq(seq)
	if checkErr != nil {
		return nil, nil, checkErr
	}
	// get sequencer's unconfirmed elders
	elders, errElders := pool.seekElders(seq)
	if errElders != nil {
		return nil, nil, errElders
	}
	// verify the elders
	pool.Logger.WithField("seq height", seq.Height).WithField("count", len(elders)).Info("tx being confirmed by seq")
	batch, err := pool.verifyConfirmBatch(seq, elders)
	if err != nil {
		return nil, nil, err
	}
	return elders, batch, err
}

// isBadSeq checks if a sequencer is correct.
func (pool *TxPool) isBadSeq(seq *types.Sequencer) error {
	// check if the nonce is duplicate
	seqindag := pool.Dag.GetTxByNonce(seq.Sender(), seq.GetNonce())
	if seqindag != nil {
		return fmt.Errorf("bad seq,duplicate nonce %d found in Dag, existing %s ", seq.GetNonce(), seqindag)
	}
	if pool.Dag.LatestSequencer().Height+1 != seq.Height {
		return fmt.Errorf("bad seq hieght mismatch  height %d old_height %d", seq.Height, pool.Dag.GetHeight())
	}
	return nil
}

func (pool *TxPool) IsBadSeq(seq *types.Sequencer) error {
	// check if the nonce is duplicate
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.isBadSeq(seq)
}

// seekElders finds all the unconfirmed elders of baseTx.
func (pool *TxPool) seekElders(baseTx types.Txi) (map[types2.Hash]types.Txi, error) {
	batch := make(map[types2.Hash]types.Txi)

	inSeekingPool := map[types2.Hash]int{}
	seekingPool := types2.Hashes{}
	for _, parentHash := range baseTx.GetParents() {
		seekingPool = append(seekingPool, parentHash)
	}
	for len(seekingPool) > 0 {
		elderHash := seekingPool[0]
		seekingPool = seekingPool[1:]

		elder := pool.get(elderHash)
		if elder == nil {
			elder = pool.Dag.GetTx(elderHash)
			if elder == nil {
				return nil, fmt.Errorf("can't find elder %s", elderHash)
			}
			continue
		}
		if batch[elder.GetHash()] == nil {
			batch[elder.GetHash()] = elder
		}
		for _, elderParentHash := range elder.GetParents() {
			if _, in := inSeekingPool[elderParentHash]; !in {
				seekingPool = append(seekingPool, elderParentHash)
				inSeekingPool[elderParentHash] = 0
			}
		}
	}
	return batch, nil
}

// verifyConfirmBatch verifies if the elders are correct.
// If passes all verifications, it returns a batch for pushing to Dag.
func (pool *TxPool) verifyConfirmBatch(seq *types.Sequencer, elders map[types2.Hash]types.Txi) (*ledger.ConfirmBatch, error) {
	// statistics of the confirmation txs.
	// Sums up the related address' income and outcome values for later verify
	// and combine the txs as confirmbatch
	cTxs := types.Txis{}
	batch := map[common.Address]*BatchDetail{}
	for _, txi := range elders {
		if txi.GetType() != types.TxBaseTypeSequencer {
			cTxs = append(cTxs, txi)
		}

		//if txi.GetType() == types.TxBaseTypeArchive {
		//	continue
		//}

		if txi.GetType() == types.TxBaseTypeTx {
		}

		// return error if a sequencer Confirm a tx that has same nonce as itself.
		if txi.Sender() == seq.Sender() && txi.GetNonce() == seq.GetNonce() {
			return nil, fmt.Errorf("seq's nonce is the same as a tx it confirmed, nonce: %d, tx hash: %s",
				seq.GetNonce(), txi.GetHash())
		}

		switch tx := txi.(type) {
		case *types.Sequencer:
			break

		case *types.Tx:
			batchFrom, okFrom := batch[tx.Sender()]
			if !okFrom {
				batchFrom = &BatchDetail{}
				batchFrom.TxList = NewTxList()
				batchFrom.Neg = make(map[int32]*math.BigInt)
				//batchFrom.Pos = make(map[int32]*math.BigInt)
				batch[tx.Sender()] = batchFrom
			}
			batchFrom.TxList.put(tx)
			batchFrom.AddNeg(tx.TokenId, tx.Value)

		default:
			batchFrom, okFrom := batch[tx.Sender()]
			if !okFrom {
				batchFrom = &BatchDetail{}
				batchFrom.TxList = NewTxList()
				batchFrom.Neg = make(map[int32]*math.BigInt)
				batch[tx.Sender()] = batchFrom
			}
			batchFrom.TxList.put(tx)
		}
	}
	// verify balance and nonce
	for addr, batchDetail := range batch {
		// check balance
		// for every token, if balance < outcome, then verify failed
		for tokenID, spent := range batchDetail.Neg {
			confirmedBalance := pool.Dag.GetBalance(addr, tokenID)
			if confirmedBalance.Value.Cmp(spent.Value) < 0 {
				return nil, fmt.Errorf("the balance of addr %s is not enough", addr)
			}
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

	cb := &ledger.ConfirmBatch{}
	cb.Seq = seq
	cb.Txs = cTxs
	return cb, nil
}

// solveConflicts remove elders from txpool and reprocess
// all txs in the pool in order to make sure all txs are
// correct after seq confirmation.
func (pool *TxPool) solveConflicts(batch *ledger.ConfirmBatch) {
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

	var txsInPool []types.Txi
	// remove elders from pool
	for _, tx := range batch.Txs {
		elderHash := tx.GetHash()
		pool.txLookup.removeTxFromMapOnly(elderHash)
	}

	for _, hash := range pool.txLookup.getorder() {
		tx := pool.get(hash)
		if tx == nil {
			continue
		}
		// TODO
		// sequencer is removed from txpool later when calling
		// pool.clearall() but not added back. Try figure out if this
		// will cause any problem.
		if tx.GetType() == types.TxBaseTypeSequencer {
			continue
		}
		txsInPool = append(txsInPool, tx)
	}
	pool.clearAll()
	for _, tx := range txsInPool {
		// TODO
		// Throw away the txs that been rejudged for more than 5 times. In order
		// to clear up the memory and the process pressure of tx pool.
		pool.Logger.WithField("tx", tx).Tracef("start rejudge")
		txEnv := &txEnvelope{
			tx:     tx,
			txType: TxTypeRejudge,
			status: TxStatusQueue,
		}
		pool.txLookup.Add(txEnv)
		e := pool.commit(tx)
		if e != nil {
			pool.Logger.WithField("tx ", tx).WithError(e).Debug("rejudge error")
		}
	}
}

func (pool *TxPool) verifyNonce(addr common.Address, noncesP *nonceHeap, seq *types.Sequencer) error {
	sort.Sort(noncesP)
	nonces := *noncesP

	latestNonce, nErr := pool.Dag.GetLatestNonce(addr)
	if nErr != nil {
		return fmt.Errorf("get latest nonce err: %v", nErr)
	}
	if nonces[0] != latestNonce+1 {
		return fmt.Errorf("nonce %d is not the next one of latest nonce %d, addr: %s", nonces[0], latestNonce, addr.String())
	}

	for i := 1; i < nonces.Len(); i++ {
		if nonces[i] != nonces[i-1]+1 {
			return fmt.Errorf("nonce order mismatch, addr: %s, preNonce: %d, curNonce: %d", addr.String(), nonces[i-1], nonces[i])
		}
	}

	if seq.Sender().Hex() == addr.Hex() {
		if seq.GetNonce() != nonces[len(nonces)-1]+1 {
			return fmt.Errorf("seq's nonce is not the next nonce of Confirm list, seq nonce: %d, latest nonce in Confirm list: %d", seq.GetNonce(), nonces[len(nonces)-1])
		}
	}

	return nil
}

// reset clears the txs that conflicts with sequencer
func (pool *TxPool) reset() {
	// TODO
}

// BatchDetail describes all the details of a specific address within a
// sequencer confirmation term.
// - TxList - represents the txs sent by this addrs, ordered by nonce.
// - Neg    - means the amount this address should spent out.
type BatchDetail struct {
	TxList *TxList
	Neg    map[int32]*math.BigInt
}

func (bd *BatchDetail) AddNeg(tokenID int32, amount *math.BigInt) {
	v, ok := bd.Neg[tokenID]
	if !ok {
		v = math.NewBigInt(0)
	}
	v.Value.Add(v.Value, amount.Value)
	bd.Neg[tokenID] = v
}

type cachedConfirm struct {
	seqHash types2.Hash
	seq     *types.Sequencer
	root    types2.Hash
	txs     types.Txis
	elders  map[types2.Hash]types.Txi
}

func newCachedConfirm(seq *types.Sequencer, root types2.Hash, txs types.Txis, elders map[types2.Hash]types.Txi) *cachedConfirm {
	return &cachedConfirm{
		seqHash: seq.GetHash(),
		seq:     seq,
		root:    root,
		txs:     txs,
		elders:  elders,
	}
}

func (c *cachedConfirm) SeqHash() types2.Hash              { return c.seqHash }
func (c *cachedConfirm) Seq() *types.Sequencer             { return c.seq }
func (c *cachedConfirm) Root() types2.Hash                 { return c.root }
func (c *cachedConfirm) Txs() types.Txis                   { return c.txs }
func (c *cachedConfirm) Elders() map[types2.Hash]types.Txi { return c.elders }

type TxMap struct {
	txs map[types2.Hash]types.Txi
	mu  sync.RWMutex
}

func NewTxMap() *TxMap {
	tm := &TxMap{
		txs: make(map[types2.Hash]types.Txi),
	}
	return tm
}

func (tm *TxMap) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return len(tm.txs)
}

func (tm *TxMap) Get(hash types2.Hash) types.Txi {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.txs[hash]
}

func (tm *TxMap) GetAllKeys() types2.Hashes {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var keys types2.Hashes
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

	if _, ok := tm.txs[tx.GetHash()]; !ok {
		return false
	}
	return true
}
func (tm *TxMap) Remove(hash types2.Hash) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.txs, hash)
}
func (tm *TxMap) Add(tx types.Txi) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, ok := tm.txs[tx.GetHash()]; !ok {
		tm.txs[tx.GetHash()] = tx
	}
}

type txLookUp struct {
	order types2.Hashes
	txs   map[types2.Hash]*txEnvelope
	mu    sync.RWMutex
}

func newTxLookUp() *txLookUp {
	return &txLookUp{
		order: types2.Hashes{},
		txs:   make(map[types2.Hash]*txEnvelope),
	}
}

// Get tx from txLookUp by hash
func (t *txLookUp) Get(h types2.Hash) types.Txi {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.get(h)
}
func (t *txLookUp) get(h types2.Hash) types.Txi {
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
	if _, ok := t.txs[txEnv.tx.GetHash()]; ok {
		return
	}

	t.order = append(t.order, txEnv.tx.GetHash())
	t.txs[txEnv.tx.GetHash()] = txEnv
}

// Remove tx from txLookUp
func (t *txLookUp) Remove(h types2.Hash, removeType hashOrderRemoveType) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.remove(h, removeType)
}

func (t *txLookUp) remove(h types2.Hash, removeType hashOrderRemoveType) {
	switch removeType {
	case noRemove:
		break
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
func (t *txLookUp) RemoveTxFromMapOnly(h types2.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.removeTxFromMapOnly(h)
}
func (t *txLookUp) removeTxFromMapOnly(h types2.Hash) {
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
func (t *txLookUp) GetOrder() types2.Hashes {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.getorder()
}

func (t *txLookUp) getorder() types2.Hashes {
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
func (t *txLookUp) Status(h types2.Hash) TxStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.status(h)
}

func (t *txLookUp) status(h types2.Hash) TxStatus {
	if txEnv := t.txs[h]; txEnv != nil {
		return txEnv.status
	}
	return TxStatusNotExist
}

// SwitchStatus switches the tx status
func (t *txLookUp) SwitchStatus(h types2.Hash, status TxStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.switchstatus(h, status)
}

func (t *txLookUp) switchstatus(h types2.Hash, status TxStatus) {
	if txEnv := t.txs[h]; txEnv != nil {
		txEnv.status = status
	}
}

//func (pool *TxPool) GetConfirmStatus() *ConfirmInfo {
//	return pool.confirmStatus.GetInfo()
//}

func (pool *TxPool) GetOrder() types2.Hashes {
	return pool.txLookup.GetOrder()
}
