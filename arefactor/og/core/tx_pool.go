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
package core

import (
	"fmt"
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/annchain/OG/arefactor/types"
	"github.com/annchain/commongo/math"
	log "github.com/sirupsen/logrus"
)

type TxType uint8

const (
	TxTypeGenesis TxType = iota
	TxTypeLocal
	TxTypeRemote
	TxTypeRejudge
)

const (
	PoolRejudgeThreshold int = 10
)

type TxPool struct {
	conf TxPoolConfig

	queue chan *txEvent // queue stores txs that need to validate later

	storage    *txPoolStorage
	cachedSets *cachedAccDetailSets
	dag        *Dag

	closed bool
	close  chan struct{}

	mu sync.RWMutex
	//wg sync.WaitGroup // for TxPool Stop()

	onNewTxReceived      map[channelName]chan types.Txi       // for notifications of new txs.
	OnBatchConfirmed     []chan map[ogTypes.HashKey]types.Txi // for notifications of confirmation.
	OnNewLatestSequencer []chan bool                          // for broadcasting new latest sequencer to record height

	maxWeight uint64
	//confirmStatus *ConfirmStatus
}

func (pool *TxPool) GetBenchmarks() map[string]interface{} {
	tipsNum, badNum, pendingNum := pool.storage.txLookup.Stats()
	return map[string]interface{}{
		"queue":      len(pool.queue),
		"event":      len(pool.onNewTxReceived),
		"txlookup":   len(pool.storage.txLookup.txs),
		"tips":       tipsNum,
		"badtxs":     badNum,
		"pendings":   pendingNum,
		"latest_seq": int(pool.dag.latestSequencer.Number()),
		"flows":      len(pool.storage.flows.afs),
		"hashordr":   len(pool.storage.getTxHashesInOrder()),
	}
}

func NewTxPool(conf TxPoolConfig, dag *Dag) *TxPool {
	if conf.ConfirmStatusRefreshTime == 0 {
		conf.ConfirmStatusRefreshTime = 30
	}

	pool := &TxPool{}

	pool.conf = conf
	pool.dag = dag
	pool.queue = make(chan *txEvent, conf.QueueSize)
	pool.storage = newTxPoolStorage(dag)
	pool.cachedSets = newCachedAccDetailSets()
	pool.closed = false
	pool.close = make(chan struct{})

	pool.onNewTxReceived = make(map[channelName]chan types.Txi)
	pool.OnBatchConfirmed = []chan map[ogTypes.HashKey]types.Txi{}

	return pool
}

type TxPoolConfig struct {
	QueueSize                int `mapstructure:"queue_size"`
	TipsSize                 int `mapstructure:"tips_size"`
	ResetDuration            int `mapstructure:"reset_duration"`
	TxVerifyTime             int `mapstructure:"tx_verify_time"`
	TxValidTime              int `mapstructure:"tx_valid_time"`
	TimeOutPoolQueue         int `mapstructure:"timeout_pool_queue_ms"`
	TimeoutSubscriber        int `mapstructure:"timeout_subscriber_ms"`
	TimeoutConfirmation      int `mapstructure:"timeout_confirmation_ms"`
	TimeoutLatestSequencer   int `mapstructure:"timeout_latest_seq_ms"`
	ConfirmStatusRefreshTime int //minute
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
	tx       types.Txi
	txType   TxType
	status   TxStatus
	judgeNum int
}

func newTxEnvelope(t TxType, status TxStatus, tx types.Txi, judgeNum int) *txEnvelope {
	te := &txEnvelope{}
	te.txType = t
	te.status = status
	te.tx = tx
	te.judgeNum = judgeNum

	return te
}

func (t *txEnvelope) String() string {
	return fmt.Sprintf("{ judgeNum: %d, tx: %s }", t.judgeNum, t.tx.String())
}

// Start begin the txpool sevices
func (pool *TxPool) Start() {
	log.Infof("TxPool Start")
	//go pool.loop()
}

// Stop stops all the txpool sevices
func (pool *TxPool) Stop() {
	close(pool.close)
	pool.closed = true
	//pool.wg.Wait()

	log.Infof("TxPool Stopped")
}

func (pool *TxPool) Init(genesis *types.Sequencer) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.storage.init(genesis)

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
	return pool.storage.stats()
}

// Get get a transaction or sequencer according to input hash,
// if tx not exists return nil
func (pool *TxPool) Get(hash ogTypes.Hash) types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.get(hash)
}

func (pool *TxPool) GetTxNum() int {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.storage.getTxNum()
}

func (pool *TxPool) GetMaxWeight() uint64 {
	return atomic.LoadUint64(&pool.maxWeight)
}

func (pool *TxPool) get(hash ogTypes.Hash) types.Txi {
	return pool.storage.getTxByHash(hash)
}

func (pool *TxPool) Has(hash ogTypes.Hash) bool {
	return pool.Get(hash) != nil
}

func (pool *TxPool) IsLocalHash(hash ogTypes.Hash) bool {
	if pool.Has(hash) {
		return true
	}
	if pool.dag.ExistTx(pool.dag.LatestSequencerHash(), hash) {
		return true
	}
	return false
}

// GetHashOrder returns a hash list of txs in pool, ordered by the
// time that txs added into pool.
func (pool *TxPool) GetHashOrder() []ogTypes.Hash {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getHashOrder()
}

func (pool *TxPool) getHashOrder() []ogTypes.Hash {
	return pool.storage.getTxHashesInOrder()
}

//func (pool *TxPool) GetOrder() []ogTypes.Hash {
//	return pool.GetHashOrder()
//}

// GetByNonce get a tx or sequencer from account flows by sender's address and tx's nonce.
func (pool *TxPool) GetByNonce(addr ogTypes.Address, nonce uint64) types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getByNonce(addr, nonce)
}
func (pool *TxPool) getByNonce(addr ogTypes.Address, nonce uint64) types.Txi {
	return pool.storage.getTxByNonce(addr, nonce)
}

// GetLatestNonce get the latest nonce of an address
func (pool *TxPool) GetLatestNonce(addr ogTypes.Address) (uint64, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.storage.getLatestNonce(addr)
}

// GetStatus gets the current status of a tx
func (pool *TxPool) GetStatus(hash ogTypes.Hash) TxStatus {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.getStatus(hash)
}

func (pool *TxPool) getStatus(hash ogTypes.Hash) TxStatus {
	return pool.storage.getTxStatusInPool(hash)
}

type channelName struct {
	name   string
	allMsg bool
}

func (c channelName) String() string {
	return c.name
}

func (pool *TxPool) RegisterOnNewTxReceived(c chan types.Txi, chanName string, allTx bool) {
	log.Tracef("RegisterOnNewTxReceived with chan: %s ,all %v", chanName, allTx)
	chName := channelName{chanName, allTx}
	pool.onNewTxReceived[chName] = c
}

// GetRandomTips returns n tips randomly.
func (pool *TxPool) GetRandomTips(n int) (v []types.Txi) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// select n random hashes
	values := pool.storage.getTipsInList()
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
func (pool *TxPool) GetAllTips() map[ogTypes.HashKey]types.Txi {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.storage.getTipsInMap()
}

//// AddLocalTx adds a tx to txpool if it is valid, note that if success it returns nil.
//// AddLocalTx only process tx that sent by local node.
//func (pool *TxPool) AddLocalTx(tx types.Txi, noFeedBack bool) error {
//	return pool.addTx(tx, TxTypeLocal, noFeedBack)
//}
//
//// AddLocalTxs adds a list of txs to txpool if they are valid. It returns
//// the process result of each tx with an error list. AddLocalTxs only process
//// txs that sent by local node.
//func (pool *TxPool) AddLocalTxs(txs []types.Txi, noFeedBack bool) []error {
//	result := make([]error, len(txs))
//	for _, tx := range txs {
//		result = append(result, pool.addTx(tx, TxTypeLocal, noFeedBack))
//	}
//	return result
//}
//
//// AddRemoteTx adds a tx to txpool if it is valid. AddRemoteTx only process tx
//// sent by remote nodes, and will hold extra functions to prevent from ddos
//// (large amount of invalid tx sent from one node in a short time) attack.
//func (pool *TxPool) AddRemoteTx(tx types.Txi, noFeedBack bool) error {
//	return pool.addTx(tx, TxTypeRemote, noFeedBack)
//}
//
//// AddRemoteTxs works as same as AddRemoteTx but processes a list of txs
//func (pool *TxPool) AddRemoteTxs(txs []types.Txi, noFeedBack bool) []error {
//	result := make([]error, len(txs))
//	for _, tx := range txs {
//		result = append(result, pool.addTx(tx, TxTypeRemote, noFeedBack))
//	}
//	return result
//}

// Remove totally removes a tx from pool, it checks badtxs, tips,
// pendings and txlookup.
func (pool *TxPool) Remove(tx types.Txi, removeType hashOrderRemoveType) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.remove(tx, removeType)
}

func (pool *TxPool) remove(tx types.Txi, removeType hashOrderRemoveType) {
	pool.storage.remove(tx, removeType)
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
	pool.storage.removeAll()
}

//func (pool *TxPool) loop() {
//	defer log.Tracef("TxPool.loop() terminates")
//
//	//pool.wg.Add(1)
//	//defer pool.wg.Done()
//
//	//resetTimer := time.NewTicker(time.Duration(pool.conf.ResetDuration) * time.Second)
//
//	for {
//		select {
//		case <-pool.close:
//			log.Info("pool got quit signal quiting...")
//			return
//
//		case txEvent := <-pool.queue:
//			log.WithField("tx", txEvent.txEnv.tx).Trace("get tx from queue")
//
//			var err error
//			tx := txEvent.txEnv.tx
//			// check if tx is duplicate
//			txStatus := pool.getStatus(tx.GetTxHash())
//			if txStatus == TxStatusNotExist {
//				log.WithField("tx", tx).Warn("tx not exists")
//				txEvent.callbackChan <- types.ErrTxNotExist
//				continue
//			}
//			if txStatus != TxStatusQueue {
//				log.WithField("tx", tx).Warn("Duplicate tx found in txlookup")
//				txEvent.callbackChan <- types.ErrDuplicateTx
//				continue
//			}
//
//			pool.mu.Lock()
//
//			switch tx := tx.(type) {
//			case *types.Sequencer:
//				err = pool.preConfirm(tx)
//				if err != nil {
//					break
//				}
//				maxWeight := atomic.LoadUint64(&pool.maxWeight)
//				if maxWeight < tx.GetWeight() {
//					atomic.StoreUint64(&pool.maxWeight, tx.GetWeight())
//				}
//			default:
//				err = pool.commit(tx)
//				if err != nil {
//					break
//				}
//				maxWeight := atomic.LoadUint64(&pool.maxWeight)
//				if maxWeight < tx.GetWeight() {
//					atomic.StoreUint64(&pool.maxWeight, tx.GetWeight())
//				}
//				tx.GetBase().Height = pool.dag.LatestSequencer().Height + 1 //temporary height ,will be re write after confirm
//			}
//
//			pool.mu.Unlock()
//
//			txEvent.callbackChan <- err
//		}
//	}
//}
//
//// addMember adds tx to the pool queue and wait to become tip after validation.
//func (pool *TxPool) addTx(tx types.Txi, senderType TxType, noFeedBack bool) error {
//	log.WithField("noFeedBack", noFeedBack).WithField("tx", tx).Tracef("start addMember, tx parents: %v", tx.Parents())
//
//	// check if tx is duplicate
//	if pool.get(tx.GetTxHash()) != nil {
//		log.WithField("tx", tx).Warn("Duplicate tx found in txpool")
//		return fmt.Errorf("duplicate tx found in txpool: %s", tx.GetTxHash().Hex())
//	}
//
//	//if normalTx, ok := tx.(*tx_types.Tx); ok {
//	//	normalTx.Setconfirm()
//	//	pool.confirmStatus.AddTxNum()
//	//}
//
//	te := &txEvent{
//		callbackChan: make(chan error),
//		txEnv:        newTxEnvelope(senderType, TxStatusQueue, tx, 1),
//	}
//	pool.storage.addTxEnv(te.txEnv)
//	pool.queue <- te
//
//	// waiting for callback
//	select {
//	case err := <-te.callbackChan:
//		if err != nil {
//			pool.remove(tx, removeFromEnd)
//			return err
//		}
//
//	}
//	// notify all subscribers of newTxEvent
//	for name, subscriber := range pool.onNewTxReceived {
//		log.WithField("tx", tx).Trace("notify subscriber: ", name)
//		if !noFeedBack || name.allMsg {
//			subscriber <- tx
//		}
//	}
//	log.WithField("tx", tx).Trace("successfully added tx to txPool")
//	return nil
//}

// commit commits tx to tips pool. commit() checks if this tx is bad tx and moves
// bad tx to badtx list other than tips list. If this tx proves any txs in the
// tip pool, those tips will be removed from tips but stored in pending.
func (pool *TxPool) commit(tx types.Txi) error {
	log.WithField("tx", tx).Trace("start commit tx")

	// check tx's quality.
	txquality := pool.isBadTx(tx)
	if txquality == TxQualityIsFatal {
		pool.remove(tx, removeFromEnd)
		tx.SetInValid(true)
		log.WithField("tx ", tx).Debug("set invalid ")
		return fmt.Errorf("tx is surely incorrect to commit, hash: %s", tx.GetTxHash())
	}
	if txquality == TxQualityIsBad {
		log.Tracef("bad tx: %s", tx)
		pool.storage.switchTxStatus(tx.GetTxHash(), TxStatusBadTx)
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
		parent := pool.get(pHash)
		if parent == nil {
			log.WithField("parent", pHash).WithField("tx", tx).
				Warn("parent status is tip but can not find in tips")
			continue
		}

		// move parent to pending
		pool.storage.switchTxStatus(parent.GetTxHash(), TxStatusPending)
	}

	pool.storage.flowProcess(tx)
	pool.storage.switchTxStatus(tx.GetTxHash(), TxStatusTip)

	// TODO delete this line later.
	if log.GetLevel() >= log.TraceLevel {
		log.WithField("tx", tx).WithField("status", pool.getStatus(tx.GetTxHash())).Tracef("finished commit tx")
	}
	return nil
}

type TxQuality uint8

const (
	TxQualityIsBad TxQuality = iota
	TxQualityIsGood
	TxQualityIsFatal
	TxQualityIgnore
)

func (pool *TxPool) isBadTx(tx types.Txi) TxQuality {
	// check if the tx's parents exists and if is badtx
	for _, parentHash := range tx.Parents() {
		// check if tx in pool
		if pool.get(parentHash) != nil {
			if pool.getStatus(parentHash) == TxStatusBadTx {
				log.WithField("tx", tx).Tracef("bad tx, parent %s is bad tx", parentHash)
				return TxQualityIsBad
			}
			continue
		}
		// check if tx in dag
		if pool.dag.GetConfirmedTx(parentHash) == nil {
			log.WithField("tx", tx).Tracef("fatal tx, parent %s is not exist", parentHash)
			return TxQualityIsFatal
		}
	}

	if tx.GetType() == types.TxBaseTypeArchive {
		return TxQualityIsGood
	}

	// check if nonce is duplicate
	txinpool := pool.storage.getTxByNonce(tx.Sender(), tx.GetNonce())
	if txinpool != nil {
		if txinpool.GetTxHash() == tx.GetTxHash() {
			log.WithField("tx", tx).Error("duplicated tx in pool. Why received many times")
			return TxQualityIsFatal
		}
		log.WithField("tx", tx).WithField("existing", txinpool).Trace("bad tx, duplicate nonce found in pool")
		return TxQualityIsBad
	}
	txindag := pool.dag.GetTxByNonce(pool.dag.ConfirmedSequencerHash(), tx.Sender(), tx.GetNonce())
	if txindag != nil {
		if txindag.GetTxHash() == tx.GetTxHash() {
			log.WithField("tx", tx).Error("duplicated tx in dag. Why received many times")
		}
		log.WithField("tx", tx).WithField("existing", txindag).Trace("bad tx, duplicate nonce found in dag")
		return TxQualityIsFatal
	}

	latestNonce, err := pool.storage.getLatestNonce(tx.Sender())
	if err != nil {
		latestNonce, err = pool.dag.GetLatestNonce(pool.dag.ConfirmedSequencerHash(), tx.Sender())
		if err != nil {
			log.Errorf("get latest nonce err: %v", err)
			return TxQualityIsFatal
		}
	}
	if tx.GetNonce() != latestNonce+1 {
		log.WithField("should be ", latestNonce+1).WithField("tx", tx).Error("nonce err")
		return TxQualityIsFatal
	}

	switch tx := tx.(type) {
	case *types.Tx:
		quality := pool.storage.tryProcessTx(tx)
		if quality != TxQualityIsGood {
			return quality
		}
	case *types.ActionTx:
		if tx.Action == types.ActionTxActionIPO {
			//actionData := tx.ActionData.(*tx_types.InitialOffering)
			//actionData.TokenId = pool.dag.GetLatestTokenId()
		}
		if tx.Action == types.ActionTxActionSPO {
			actionData := tx.ActionData.(*types.SecondaryOffering)
			if actionData.TokenId == 0 {
				log.WithField("tx ", tx).Warn("og token is disabled for publishing")
				return TxQualityIsFatal
			}
			token := pool.dag.GetToken(pool.dag.LatestSequencerHash(), actionData.TokenId)
			if token == nil {
				log.WithField("tx ", tx).Warn("token not found")
				return TxQualityIsFatal
			}
			if !token.ReIssuable {
				log.WithField("tx ", tx).Warn("token is disabled for second publishing")
				return TxQualityIsFatal
			}
			if token.Destroyed {
				log.WithField("tx ", tx).Warn("token is destroyed already")
				return TxQualityIsFatal
			}
			if token.Issuer != tx.Sender() {
				log.WithField("token ", token).WithField("you address", tx.Sender()).Warn("you have no authority to second publishing for this token")
				return TxQualityIsFatal
			}
		}
		if tx.Action == types.ActionTxActionDestroy {
			actionData := tx.ActionData.(*types.DestroyOffering)
			if actionData.TokenId == 0 {
				log.WithField("tx ", tx).Warn("og token is disabled for withdraw")
				return TxQualityIsFatal
			}
			token := pool.dag.GetToken(pool.dag.LatestSequencerHash(), actionData.TokenId)
			if token == nil {
				log.WithField("tx ", tx).Warn("token not found")
				return TxQualityIsFatal
			}
			if token.Destroyed {
				log.WithField("tx ", tx).Warn("token is destroyed already")
				return TxQualityIsFatal
			}
			if token.Issuer != tx.Sender() {
				log.WithField("tx ", tx).WithField("token ", token).WithField("you address", tx.Sender()).Warn("you have no authority to second publishing for this token")
				return TxQualityIsFatal
			}
		}
	default:
		// TODO
	}

	return TxQualityIsGood
}

//func (pool *TxPool) processSequencer(seq *types.Sequencer) error {
//	if err := pool.isBadSeq(seq); err != nil {
//		return err
//	}
//
//	confirmSeqHash := seq.GetConfirmSeqHash()
//	cBatch := pool.cachedBatches.getConfirmBatch(confirmSeqHash)
//	if cBatch == nil {
//		return fmt.Errorf("can't find ")
//	}
//
//	return nil
//}

func (pool *TxPool) PreConfirm(seq *types.Sequencer) ([]types.Txi, error) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	return pool.preConfirm(seq)
}

// preConfirm try to confirm sequencer and store the related data into pool.cachedSets.
// Once a sequencer is hardly confirmed after 3 later preConfirms, the function confirm()
// will be dialed.
func (pool *TxPool) preConfirm(seq *types.Sequencer) ([]types.Txi, error) {
	if seq.GetHeight() <= pool.dag.latestSequencer.GetHeight() {
		return nil, fmt.Errorf("the height of seq to pre-confirm is lower than "+
			"the latest seq in dag. get height: %d, latest: %d", seq.GetHeight(), pool.dag.latestSequencer.GetHeight())
	}

	err := pool.isBadSeq(seq)
	if err != nil {
		return nil, err
	}
	_, eldersMap, err := pool.seekElders(seq)
	if err != nil {
		return nil, err
	}
	accountDetailSet := newAccountDetailSet(seq, pool.dag)
	err = accountDetailSet.construct(eldersMap)
	if err != nil {
		return nil, err
	}
	err = accountDetailSet.isValid(pool.dag)
	if err != nil {
		return nil, err
	}
	err = pool.cachedSets.addSet(accountDetailSet)
	if err != nil {
		return nil, err
	}

	pool.storage.switchTxStatus(seq.GetTxHash(), TxStatusSeqPreConfirm)
	return accountDetailSet.eldersList, nil
}

func (pool *TxPool) Confirm(seqHash ogTypes.Hash) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	return pool.confirm(seqHash)
}

// confirm pushes a batch of txs that confirmed by a sequencer to the dag.
func (pool *TxPool) confirm(seqHash ogTypes.Hash) error {

	accSet := pool.cachedSets.getSet(seqHash)

	// push pushBatch to dag
	pushBatch := &PushBatch{
		Seq: accSet.seq,
		Txs: accSet.eldersList,
	}
	if err := pool.dag.Push(pushBatch); err != nil {
		log.WithField("error", err).Errorf("dag Push error: %v", err)
		return err
	}

	// solve conflicts of txs in pool
	pool.solveConflicts(accSet)

	// notification
	for _, c := range pool.OnBatchConfirmed {
		if pool.closed {
			break
		}
		c <- accSet.eldersMap
	}
	for _, c := range pool.OnNewLatestSequencer {
		if pool.closed {
			break
		}
		c <- true
	}

	log.WithField("seq height", accSet.seq.Height).WithField("seq", accSet.seq).Trace("finished confirm seq")
	return nil
}

// isBadSeq checks if a sequencer is correct.
func (pool *TxPool) isBadSeq(seq *types.Sequencer) error {
	// check if the nonce is duplicate
	seqindag := pool.dag.GetTxByNonce(seq.GetConfirmSeqHash(), seq.Sender(), seq.GetNonce())
	if seqindag != nil {
		return fmt.Errorf("bad seq,duplicate nonce %d found in dag, existing %s ", seq.GetNonce(), seqindag)
	}

	if pool.dag.LatestSequencer().Height+1 != seq.Height {
		return fmt.Errorf("bad seq hieght mismatch  height %d old_height %d", seq.Height, pool.dag.latestSequencer.Height)
	}
	return nil
}

func (pool *TxPool) IsBadSeq(seq *types.Sequencer) error {
	// check if the nonce is duplicate
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.isBadSeq(seq)
}

func (pool *TxPool) SeekElders(seq *types.Sequencer) ([]types.Txi, map[ogTypes.HashKey]types.Txi, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.seekElders(seq)
}

// seekElders finds all the unconfirmed elders of baseTx.
func (pool *TxPool) seekElders(seq *types.Sequencer) ([]types.Txi, map[ogTypes.HashKey]types.Txi, error) {
	eldersList := make([]types.Txi, 0)
	eldersMap := make(map[ogTypes.HashKey]types.Txi)

	inSeekingPool := map[ogTypes.HashKey]int{}
	var seekingPool []ogTypes.Hash
	for _, parentHash := range seq.Parents() {
		seekingPool = append(seekingPool, parentHash)
	}
	for len(seekingPool) > 0 {
		elderHash := seekingPool[0]
		seekingPool = seekingPool[1:]

		elder := pool.get(elderHash)
		if elder == nil {
			// check if elder has been preConfirmed or confirmed.
			if pool.dag.ExistTx(seq.GetParentSeqHash(), elderHash) {
				continue
			}
			return nil, nil, fmt.Errorf("can't find elder %s", elderHash)
		}

		if eldersMap[elder.GetTxHash().HashKey()] == nil {
			eldersList = append(eldersList, elder)
			eldersMap[elder.GetTxHash().HashKey()] = elder
		}
		for _, elderParentHash := range elder.Parents() {
			if _, in := inSeekingPool[elderParentHash.HashKey()]; !in {
				seekingPool = append(seekingPool, elderParentHash)
				inSeekingPool[elderParentHash.HashKey()] = 0
			}
		}
	}
	return eldersList, eldersMap, nil
}

// solveConflicts remove elders from txpool and reprocess
// all txs in the pool in order to make sure all txs are
// correct after seq confirmation.
func (pool *TxPool) solveConflicts(accSet *accDetailSet) {
	// TODO check if there is any bugs here
	txToRejudge := pool.storage.switchToConfirmBatch(accSet)
	for _, txEnv := range txToRejudge {
		if txEnv.judgeNum >= PoolRejudgeThreshold {
			log.WithField("txEnvelope", txEnv.String()).Infof("exceed rejudge time, throw away")
			continue
		}

		log.WithField("txEnvelope", txEnv).Tracef("start rejudge")
		txEnv.txType = TxTypeRejudge
		txEnv.status = TxStatusQueue
		txEnv.judgeNum += 1
		pool.storage.addTxEnv(txEnv)
		e := pool.commit(txEnv.tx)
		if e != nil {
			log.WithField("txEnvelope ", txEnv).WithError(e).Debug("rejudge error")
		}
	}
}

type Ledger interface {
	LatestSequencerHash() ogTypes.Hash
	ConfirmedSequencerHash() ogTypes.Hash
	GetBalance(baseSeqHash ogTypes.Hash, addr ogTypes.Address, tokenID int32) *math.BigInt
	GetLatestNonce(baseSeqHash ogTypes.Hash, addr ogTypes.Address) (uint64, error)
}

type cachedAccDetailSets struct {
	heights []uint64
	sets    map[uint64][]*accDetailSet
	setsMap map[ogTypes.HashKey]*accDetailSet
}

func newCachedAccDetailSets() *cachedAccDetailSets {
	return &cachedAccDetailSets{
		heights: make([]uint64, 0),
		sets:    make(map[uint64][]*accDetailSet),
		setsMap: make(map[ogTypes.HashKey]*accDetailSet),
	}
}

func (ca *cachedAccDetailSets) getSet(seqHash ogTypes.Hash) *accDetailSet {
	return ca.setsMap[seqHash.HashKey()]
}

func (ca *cachedAccDetailSets) addSet(as *accDetailSet) error {
	height := as.seq.GetHeight()
	if len(ca.heights) == 0 {
		return ca.addSetHelper(as)
	}
	if height < ca.heights[0] {
		return fmt.Errorf("height incorrect, should no less than minimun %d", ca.heights[0])
	}
	return ca.addSetHelper(as)
}

func (ca *cachedAccDetailSets) addSetHelper(as *accDetailSet) error {
	height := as.seq.GetHeight()
	if len(ca.heights) == 0 || height == ca.heights[len(ca.heights)-1]+1 {
		ca.heights = append(ca.heights, height)
		ca.sets[height] = make([]*accDetailSet, 0)
	}

	arr, exists := ca.sets[height]
	if !exists {
		return fmt.Errorf("height not exists: %d, current highest: %d", height, ca.heights[len(ca.heights)-1])
	}
	arr = append(arr, as)

	ca.sets[height] = arr
	ca.setsMap[as.seq.GetTxHash().HashKey()] = as
	return nil
}

func (ca *cachedAccDetailSets) confirmSet(seqHash ogTypes.Hash) error {
	as, exists := ca.setsMap[seqHash.HashKey()]
	if !exists {
		return fmt.Errorf("can't find cached detailSet: %s", seqHash.HashKey())
	}

	//height := as.seq.GetHeight()
	var breakIndex int
	for i, height := range ca.heights {
		if height >= as.seq.GetHeight() {
			breakIndex = i
			break
		}
		arr := ca.sets[height]
		for _, accSet := range arr {
			delete(ca.setsMap, accSet.seq.GetTxHash().HashKey())
		}
		delete(ca.sets, height)
	}
	ca.heights = ca.heights[breakIndex:]
	return nil
}

type accDetailSet struct {
	seq        *types.Sequencer
	eldersList []types.Txi
	eldersMap  map[ogTypes.HashKey]types.Txi
	ledger     Ledger
	details    map[ogTypes.AddressKey]*accountDetail
}

func newAccountDetailSet(seq *types.Sequencer, ledger Ledger) *accDetailSet {
	return &accDetailSet{
		seq:        seq,
		eldersList: make([]types.Txi, 0),
		eldersMap:  make(map[ogTypes.HashKey]types.Txi),
		ledger:     ledger,
		details:    make(map[ogTypes.AddressKey]*accountDetail),
	}
}

func (as *accDetailSet) construct(elders map[ogTypes.HashKey]types.Txi) error {
	for _, txi := range elders {
		// return error if a sequencer confirm a tx that has same nonce as itself.
		if txi.Sender() == as.seq.Sender() && txi.GetNonce() == as.seq.GetNonce() {
			return fmt.Errorf("seq's nonce is the same as a tx it confirmed, nonce: %d, tx hash: %s",
				as.seq.GetNonce(), txi.GetTxHash())
		}

		switch tx := txi.(type) {
		case *types.Sequencer:
			break
		//case *types.Tx:
		default:
			as.processTx(tx)
		}
	}
	return nil
}

func (as *accDetailSet) processTx(tx types.Txi) {
	switch tx := tx.(type) {
	case *types.Tx:
		detailCost := as.getOrCreateDetail(tx.From)
		detailCost.addCost(as.ledger, as.seq.GetParentSeqHash(), tx.TokenId, tx.Value)
		detailCost.addTx(tx)
		as.setDetail(detailCost)

		detailEarn := as.getOrCreateDetail(tx.To)
		detailEarn.addEarn(as.ledger, as.seq.GetParentSeqHash(), tx.TokenId, tx.Value)
		as.setDetail(detailEarn)
	default:
		detailSender := as.getOrCreateDetail(tx.Sender())
		detailSender.addTx(tx)
	}
	as.eldersList = append(as.eldersList, tx)
	as.eldersMap[tx.GetTxHash().HashKey()] = tx
}

func (as *accDetailSet) isValid(ledger Ledger) error {
	// verify balance and nonce
	for _, accountDetail := range as.getDetails() {
		err := accountDetail.isValid(ledger, as.seq.GetParentSeqHash(), as.seq)
		if err != nil {
			return err
		}
	}
	return nil
}

func (as *accDetailSet) getDetail(addr ogTypes.Address) *accountDetail {
	return as.details[addr.AddressKey()]
}

func (as *accDetailSet) getDetails() map[ogTypes.AddressKey]*accountDetail {
	return as.details
}

func (as *accDetailSet) createDetail(addr ogTypes.Address) *accountDetail {
	as.details[addr.AddressKey()] = newAccountDetail(addr)
	return as.details[addr.AddressKey()]
}

func (as *accDetailSet) setDetail(detail *accountDetail) {
	as.details[detail.address.AddressKey()] = detail
}

func (as *accDetailSet) getOrCreateDetail(addr ogTypes.Address) *accountDetail {
	detailCost := as.getDetail(addr)
	if detailCost == nil {
		detailCost = as.createDetail(addr)
	}
	return detailCost
}

func (as *accDetailSet) existTx(hash ogTypes.Hash) bool {
	_, exist := as.eldersMap[hash.HashKey()]
	return exist
}

func (as *accDetailSet) existSeq(hash ogTypes.Hash) bool {
	txi, exist := as.eldersMap[hash.HashKey()]
	if !exist {
		return false
	}
	if txi.GetType() != types.TxBaseTypeSequencer {
		return false
	}
	return true
}

// accountDetail describes all the db of a specific address within a
// sequencer confirmation term.
//
// - txList        - represents the txs sent by this addrs, ordered by nonce.
// - earn          - the amount this address have earned.
// - cost          - the amount this address should spent out.
// - resultBalance - balance of this address after the batch is confirmed.
type accountDetail struct {
	//ledger LedgerEngine

	address       ogTypes.Address
	txList        *TxList
	earn          map[int32]*math.BigInt
	cost          map[int32]*math.BigInt
	resultBalance map[int32]*math.BigInt
}

func newAccountDetail(addr ogTypes.Address) *accountDetail {
	bd := &accountDetail{
		//ledger:        ledger,
		address:       addr,
		txList:        NewTxList(),
		earn:          make(map[int32]*math.BigInt),
		cost:          make(map[int32]*math.BigInt),
		resultBalance: make(map[int32]*math.BigInt),
	}
	return bd
}

func (bd *accountDetail) getBalance(tokenID int32) *math.BigInt {
	return bd.resultBalance[tokenID]
}

func (bd *accountDetail) getNonce() (uint64, error) {
	if bd.txList == nil {
		return 0, fmt.Errorf("txlist is nil")
	}
	if !(bd.txList.Len() > 0) {
		return 0, fmt.Errorf("txlist is empty")
	}
	return bd.txList.keys.Tail(), nil
}

func (bd *accountDetail) existTx(tx types.Txi) bool {
	return bd.txList.Get(tx.GetNonce()) != nil
}

func (bd *accountDetail) addTx(tx types.Txi) {
	bd.txList.Put(tx)
}

func (bd *accountDetail) addCost(ledger Ledger, baseHash ogTypes.Hash, tokenID int32, amount *math.BigInt) {
	v, ok := bd.cost[tokenID]
	if !ok {
		v = math.NewBigInt(0)
	}
	bd.cost[tokenID] = v.Add(amount)

	blc, ok := bd.resultBalance[tokenID]
	if !ok {
		blc = ledger.GetBalance(baseHash, bd.address, tokenID)
	}
	bd.resultBalance[tokenID] = blc.Sub(amount)
}

func (bd *accountDetail) addEarn(ledger Ledger, baseHash ogTypes.Hash, tokenID int32, amount *math.BigInt) {
	v, ok := bd.earn[tokenID]
	if !ok {
		v = math.NewBigInt(0)
	}
	bd.earn[tokenID] = v.Add(amount)

	blc, ok := bd.resultBalance[tokenID]
	if !ok {
		blc = ledger.GetBalance(baseHash, bd.address, tokenID)
	}
	bd.resultBalance[tokenID] = blc.Add(amount)
}

func (bd *accountDetail) isValid(ledger Ledger, baseHash ogTypes.Hash, seq *types.Sequencer) error {
	// check balance
	// for every token, if balance < cost, verify failed
	for tokenID, cost := range bd.cost {
		confirmedBalance := ledger.GetBalance(baseHash, bd.address, tokenID)
		if confirmedBalance.Value.Cmp(cost.Value) < 0 {
			return fmt.Errorf("the balance of addr %s is not enough", bd.address.AddressString())
		}
	}

	// check nonce order
	nonces := bd.txList.keys
	if !(nonces.Len() > 0) {
		return nil
	}
	if nErr := bd.verifyNonce(ledger, baseHash, seq, *nonces); nErr != nil {
		return nErr
	}
	return nil
}

func (bd *accountDetail) verifyNonce(ledger Ledger, baseHash ogTypes.Hash, seq *types.Sequencer, nonces nonceHeap) error {
	sort.Sort(nonces)
	addr := bd.address

	latestNonce, err := ledger.GetLatestNonce(baseHash, bd.address)
	if err != nil {
		return fmt.Errorf("get latest nonce err: %v", err)
	}
	if nonces[0] != latestNonce+1 {
		return fmt.Errorf("nonce %d is not the next one of latest nonce %d, addr: %s", nonces[0], latestNonce, addr.AddressString())
	}
	for i := 1; i < nonces.Len(); i++ {
		if nonces[i] != nonces[i-1]+1 {
			return fmt.Errorf("nonce order mismatch, addr: %s, preNonce: %d, curNonce: %d", addr.AddressString(), nonces[i-1], nonces[i])
		}
	}
	if seq.Sender().Hex() == addr.Hex() {
		if seq.GetNonce() != nonces[len(nonces)-1]+1 {
			return fmt.Errorf("seq's nonce is not the next nonce of confirm list, seq nonce: %d, latest nonce in confirm list: %d", seq.GetNonce(), nonces[len(nonces)-1])
		}
	}
	return nil
}
