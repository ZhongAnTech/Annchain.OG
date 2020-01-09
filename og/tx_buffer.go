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
package og

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/types/tx_types"
	"sort"
	"sync"
	"time"

	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/common/math"

	"github.com/annchain/OG/types"
	"github.com/annchain/gcache"
	"github.com/sirupsen/logrus"
)

type txStatus int

const (
	txStatusNone       txStatus = iota
	txStatusFetched             // all previous ancestors got
	txStatusValidated           // ancestors are valid
	txStatusConflicted          // ancestors are conflicted, or itself is conflicted
)

type Syncer interface {
	Enqueue(hash *common.Hash, childHash common.Hash, sendBloomFilter bool)
	SyncHashList(seqHash common.Hash)
	ClearQueue()
	IsCachedHash(hash common.Hash) bool
}

type Announcer interface {
	BroadcastNewTx(txi types.Txi)
}

type ITxPool interface {
	Get(hash common.Hash) types.Txi
	AddRemoteTx(tx types.Txi, noFeedBack bool) error
	RegisterOnNewTxReceived(c chan types.Txi, name string, allTx bool)
	GetLatestNonce(addr common.Address) (uint64, error)
	IsLocalHash(hash common.Hash) bool
	GetMaxWeight() uint64
	GetByNonce(addr common.Address, nonce uint64) types.Txi
	IsBadSeq(seq *tx_types.Sequencer) error
}

type IDag interface {
	GetTx(hash common.Hash) types.Txi
	GetTxByNonce(addr common.Address, nonce uint64) types.Txi
	GetSequencerByHeight(id uint64) *tx_types.Sequencer
	GetTxisByNumber(id uint64) types.Txis
	LatestSequencer() *tx_types.Sequencer
	GetSequencer(hash common.Hash, id uint64) *tx_types.Sequencer
	Genesis() *tx_types.Sequencer
	GetHeight() uint64
	GetSequencerByHash(hash common.Hash) *tx_types.Sequencer
	GetBalance(addr common.Address, tokenID int32) *math.BigInt
	GetLatestNonce(addr common.Address) (uint64, error)
}

// TxBuffer rebuild graph by buffering newly incoming txs and find their parents.
// Tx will be buffered here until parents are got.
// Once the parents are got, Tx will be send to TxPool for further processing.
type TxBuffer struct {
	dag                    IDag
	Verifiers              []Verifier
	Syncer                 Syncer
	Announcer              Announcer
	txPool                 ITxPool
	dependencyCache        gcache.Cache // list of hashes that are pending on the parent. map[common.Hash]map[common.Hash]types.Tx
	affmu                  sync.RWMutex
	SelfGeneratedNewTxChan chan types.Txi
	ReceivedNewTxChan      chan types.Txi
	ReceivedNewTxsChan     chan []types.Txi
	quit                   chan bool
	knownCache             gcache.Cache // txs that are already fulfilled and pushed to txpool
	txAddedToPoolChan      chan types.Txi
	OnProposalSeqCh        chan common.Hash
	//children               *childrenCache //key : phash ,value :
	//HandlingQueue           txQueue
	TestNoVerify bool
	wg           *sync.WaitGroup
}

type childrenCache struct {
	cache gcache.Cache
	mu    sync.RWMutex
}

func newChilrdenCache(size int, expire time.Duration) *childrenCache {
	return &childrenCache{
		cache: gcache.New(size).Simple().Expiration(expire).Build(),
	}
}

func (c *childrenCache) AddChildren(parent common.Hash, child common.Hash) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, err := c.cache.GetIFPresent(parent)
	var children common.Hashes
	if err == nil {
		children = value.(common.Hashes)
	}
	for _, h := range children {
		if h == child {
			return
		}
	}
	children = append(children, child)
	if len(children) != 0 {
		c.cache.Set(parent, children)
	}
}

func (c *childrenCache) GetChildren(parent common.Hash) (children common.Hashes) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, err := c.cache.GetIFPresent(parent)
	if err == nil {
		children = value.(common.Hashes)
	}
	return
}

func (c *childrenCache) GetAndRemove(parent common.Hash) (children common.Hashes) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, err := c.cache.GetIFPresent(parent)
	if err == nil {
		children = value.(common.Hashes)
	} else {
		return
	}
	c.cache.Remove(parent)
	return
}

func (c *childrenCache) Remove(parent common.Hash) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache.Remove(parent)
}

func (c *childrenCache) Len() int {
	return c.cache.Len(true)
}

func (b *TxBuffer) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"selfGeneratedNewTxChan": len(b.SelfGeneratedNewTxChan),
		"receivedNewTxChan":      len(b.ReceivedNewTxChan),
		"receivedNewTxsChan":     len(b.ReceivedNewTxsChan),
		"dependencyCache":        b.dependencyCache.Len(true),
		"knownCache":             b.knownCache.Len(true),
		"txAddedToPoolChan":      len(b.txAddedToPoolChan),
		//"childrenCache": b.children.Len(),
	}
}

type TxBufferConfig struct {
	Dag                              IDag
	Verifiers                        []Verifier
	Syncer                           Syncer
	TxAnnouncer                      Announcer
	TxPool                           ITxPool
	DependencyCacheMaxSize           int
	DependencyCacheExpirationSeconds int
	NewTxQueueSize                   int
	KnownCacheMaxSize                int
	KnownCacheExpirationSeconds      int
	AddedToPoolQueueSize             int
	TestNoVerify                     bool
}

func NewTxBuffer(config TxBufferConfig) *TxBuffer {
	return &TxBuffer{
		dag:       config.Dag,
		Verifiers: config.Verifiers,
		Syncer:    config.Syncer,
		Announcer: config.TxAnnouncer,
		txPool:    config.TxPool,
		dependencyCache: gcache.New(config.DependencyCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.DependencyCacheExpirationSeconds)).Build(),
		//todo maybe we need big size channels
		SelfGeneratedNewTxChan: make(chan types.Txi, config.NewTxQueueSize),
		ReceivedNewTxChan:      make(chan types.Txi, config.NewTxQueueSize),
		ReceivedNewTxsChan:     make(chan []types.Txi, config.NewTxQueueSize),
		txAddedToPoolChan:      make(chan types.Txi, config.AddedToPoolQueueSize),
		quit:                   make(chan bool),
		knownCache: gcache.New(config.KnownCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.KnownCacheExpirationSeconds)).Build(),
		//children: newChilrdenCache(config.DependencyCacheMaxSize, time.Second*time.Duration(config.DependencyCacheExpirationSeconds)),
		TestNoVerify: config.TestNoVerify,
		wg:           &sync.WaitGroup{},
	}
}

func (b *TxBuffer) Start() {
	b.txPool.RegisterOnNewTxReceived(b.txAddedToPoolChan, "b.txAddedToPoolChan", false)
	goroutine.New(b.loop)
	goroutine.New(b.releasedTxCacheLoop)
}

func (b *TxBuffer) Stop() {
	logrus.Info("tx buffer will stop.")
	close(b.quit)
}

func (b *TxBuffer) Name() string {
	return "TxBuffer"
}

func (b *TxBuffer) loop() {
	for {
		select {
		case <-b.quit:
			logrus.Info("tx buffer received quit message. Quitting...")
			return
		case v := <-b.ReceivedNewTxChan:
			b.handleTx(v)
		case txs := <-b.ReceivedNewTxsChan:
			b.handleTxs(txs)
		case v := <-b.SelfGeneratedNewTxChan:
			b.handleTx(v)
		}
	}
}

// niceTx is the logic triggered when tx's ancestors are all fetched to local
func (b *TxBuffer) niceTx(tx types.Txi, firstTime bool) {
	// Check if the tx is valid based on graph structure rules
	// Only txs that are obeying rules will be added to the graph.
	b.knownCache.Remove(tx.GetTxHash())

	// added verifier for specific tx types. e.g. Campaign, TermChange.
	if !b.TestNoVerify {
		for _, verifier := range b.Verifiers {
			if verifier.Independent() {
				continue
			}
			if !verifier.Verify(tx) {
				logrus.WithField("tx", tx).WithField("verifier", verifier).Warn("bad tx")
				return
			}
		}
	}
	logrus.WithField("tx", tx).Debugf("nice tx")
	// resolve other dependencies
	b.resolve(tx, firstTime)
}

// in parallel
func (b *TxBuffer) handleTx(tx types.Txi) {
	logrus.WithField("tx", tx).WithField("parents", tx.Parents()).Debugf("buffer is handling tx")
	start := time.Now()
	defer func() {
		logrus.WithField("ts", time.Now().Sub(start)).WithField("tx", tx).WithField("parents", tx.Parents()).Debugf("buffer handled tx")
		// logrus.WithField("tx", tx).Debugf("buffer handled tx")
	}()
	// already in the dag or tx_pool or buffer itself.
	if b.IsKnownHash(tx.GetTxHash()) {
		var overWriteProposalSeq bool
		//if tx.GetType() ==types.TxBaseTypeSequencer {
		//	txi := b.GetFromBuffer(tx.GetTxHash())
		//	if txi.GetType() ==types.TxBaseTypeSequencer && txi.GetTxHash() == tx.GetTxHash(){
		//		seq :=txi.(*tx_types.Sequencer)
		//		if seq.Proposing {
		//			overWriteProposalSeq = true
		//		}
		//	}
		//}
		if !overWriteProposalSeq {
			return
		}

	}
	//if err := b.verifyTxFormat(tx); err != nil {
	//	logrus.WithError(err).WithField("tx", tx).Debugf("buffer received invalid tx")
	//	return
	//}
	if !b.TestNoVerify {
		for _, verifier := range b.Verifiers {
			if !verifier.Independent() {
				continue
			}
			if !verifier.Verify(tx) {
				logrus.WithField("tx", tx).WithField("verifier", verifier).Warn("bad tx")
				return
			}
		}
	}

	b.knownCache.Set(tx.GetTxHash(), tx)

	if b.buildDependencies(tx) {
		// directly fulfilled, insert into txpool
		// needs to resolve itself first
		logrus.WithField("tx", tx).Debugf("new tx directly fulfilled in buffer")
		b.niceTx(tx, true)
	}
}

func (b *TxBuffer) verify(tx types.Txi, validTxs *types.Txis) {
	if !b.TestNoVerify {
		for _, verifier := range b.Verifiers {
			if !verifier.Independent() {
				continue
			}
			if !verifier.Verify(tx) {
				logrus.WithField("tx", tx).WithField("verifier", verifier).Warn("bad tx")
				return
			}
		}
	}
	b.affmu.Lock()
	b.knownCache.Set(tx.GetTxHash(), tx)
	*validTxs = append(*validTxs, tx)
	b.affmu.Unlock()
	b.wg.Done()
}

// in parallel
func (b *TxBuffer) handleTxs(txs types.Txis) {
	logrus.WithField("tx", txs).Debug("buffer is handling txs")
	start := time.Now()
	defer func() {
		logrus.WithField("txs", time.Now().Sub(start)).WithField("txs", txs).Debug("buffer handled txs")
		// logrus.WithField("tx", tx).Debugf("buffer handled tx")
	}()
	var validTxs types.Txis
	for i := range txs {
		// already in the dag or tx_pool or buffer itself.
		tx := txs[i]
		// already in the dag or tx_pool or buffer itself.
		if b.IsKnownHash(tx.GetTxHash()) {
			var overWriteProposalSeq bool
			//if tx.GetType() ==types.TxBaseTypeSequencer {
			//	txi := b.GetFromBuffer(tx.GetTxHash())
			//	if txi.GetType() ==types.TxBaseTypeSequencer && txi.GetTxHash() == tx.GetTxHash(){
			//		seq :=txi.(*tx_types.Sequencer)
			//		if seq.Proposing {
			//			overWriteProposalSeq = true
			//		}
			//	}
			//}
			if !overWriteProposalSeq {
				continue
			}

		}

		//b.knownCache.Set(txs[i].GetTxHash(), txs[i])
		//validTxs = append(validTxs,txs[i])

		//logrus.Debug("i ", i, tx)
		f := func() {
			//logrus.Debug(tx)
			b.verify(tx, &validTxs)
		}
		b.wg.Add(1)
		goroutine.New(f)

	}
	b.wg.Wait()
	sort.Sort(validTxs)
	//logrus.Debug(validTxs,"hahah")
	for _, tx := range validTxs {
		logrus.WithField("tx", tx).WithField("parents", tx.Parents()).Debugf("buffer is handling tx")
		if b.buildDependencies(tx) {
			// directly fulfilled, insert into txpool
			// needs to resolve itself first
			logrus.WithField("tx", tx).Debugf("new tx directly fulfilled in buffer")
			b.niceTx(tx, true)
		}
	}
}

func (b *TxBuffer) GetFromBuffer(hash common.Hash) types.Txi {
	a, err := b.knownCache.GetIFPresent(hash)
	if err == nil {
		return a.(types.Txi)
	}
	return nil
}

func (b *TxBuffer) GetFromAllKnownSource(hash common.Hash) types.Txi {
	if tx := b.GetFromBuffer(hash); tx != nil {
		return tx
	} else if tx := b.GetFromProviders(hash); tx != nil {
		return tx
	}
	return nil
}

func (b *TxBuffer) GetFromProviders(hash common.Hash) types.Txi {
	if poolTx := b.txPool.Get(hash); poolTx != nil {
		return poolTx
	} else if dagTx := b.dag.GetTx(hash); dagTx != nil {
		return dagTx
	}
	return nil
}

// updateDependencyMap will update dependency relationship currently known.
// e.g., If there is already (c <- b), adding (c <- a) will result in (c <- [a,b]).

func (b *TxBuffer) updateDependencyMap(parentHash common.Hash, self types.Txi) {
	logrus.WithFields(logrus.Fields{
		"parent": parentHash,
		"child":  self,
	}).Debugf("updating dependency map")

	b.affmu.Lock()
	v, err := b.dependencyCache.GetIFPresent(parentHash)

	if err != nil {
		// key not present, need to build an inner map
		v = map[common.Hash]types.Txi{self.GetBase().Hash: self}
	}
	v.(map[common.Hash]types.Txi)[self.GetBase().Hash] = self
	b.dependencyCache.Set(parentHash, v)

	b.affmu.Unlock()
}

func (b *TxBuffer) addToTxPool(tx types.Txi) error {
	//no need to receive added tx by buffer
	return b.txPool.AddRemoteTx(tx, true)
}

// resolve is called when all ancestors of the tx is got.
// Once resolved, add it to the pool
func (b *TxBuffer) resolve(tx types.Txi, firstTime bool) {
	var proposingSeq bool
	if tx.GetType() == types.TxBaseTypeSequencer {
		seq := tx.(*tx_types.Sequencer)
		if seq.Proposing {
			function := func() {
				b.OnProposalSeqCh <- seq.GetTxHash()
			}
			goroutine.New(function)
			logrus.WithField("seq ", seq).Debug("is a proposiong seq ")
			//
			proposingSeq = true
		}
	}

	vs, err := b.dependencyCache.GetIFPresent(tx.GetTxHash())
	//children := b.children.GetAndRemove(tx.GetTxHash())
	logrus.WithField("tx", tx).Trace("after cache GetIFPresent")
	if !proposingSeq {
		addErr := b.addToTxPool(tx)
		if addErr != nil {
			logrus.WithField("txi", tx).WithError(addErr).Warn("add tx to txpool err")
		} else {
			b.Announcer.BroadcastNewTx(tx)
		}
	}
	b.dependencyCache.Remove(tx.GetTxHash())

	logrus.WithField("tx", tx).Debugf("tx resolved")

	if err != nil {
		//if len(children) == 0 {
		// key not present, already resolved.
		if firstTime {
			logrus.WithField("tx", tx).Debug("new local tx")
		} else {
			logrus.WithField("tx", tx).Warn("already been resolved before")
		}
		return
	}
	// try resolve the remainings
	for _, v := range vs.(map[common.Hash]types.Txi) {
		if v.GetTxHash() == tx.GetTxHash() {
			// self already resolved
			continue
		}
		//for _, h := range children {
		//	v, err := b.knownCache.GetIFPresent(h)
		//if err != nil {
		//continue
		//}
		//txi := v.(types.Txi)
		txi := v
		logrus.WithField("resolved", tx).WithField("resolving", txi).Debugf("cascade resolving")
		b.tryResolve(txi)
	}
}

//// isLocalHash tests if the tx has already been in the txpool or dag.
//func (b *TxBuffer) isLocalHash(hash common.Hash) bool {
//	//just get once
//	var poolTx, dagTx types.Txi
//	ok := false
//	if poolTx = b.txPool.Get(hash); poolTx != nil {
//		ok = true
//	} else if dagTx = b.dag.GetTx(hash); dagTx != nil {
//		ok = true
//	}
//	//logrus.WithFields(logrus.Fields{
//	//	"TxPool": poolTx,
//	//	"DAG":    dagTx,
//	//	"Hash":   hash,
//	//	//"Buffer": b.GetFromBuffer(hash),
//	//}).Trace("transaction location")
//	return ok
//}

func (b *TxBuffer) isLocalHash(hash common.Hash) bool {
	return b.txPool.IsLocalHash(hash)
}

// isKnownHash tests if the tx is ever heard of, either in local or in buffer.
// if tx is known, do not broadcast anymore
func (b *TxBuffer) IsKnownHash(hash common.Hash) bool {
	return b.isBufferedHash(hash) || b.txPool.IsLocalHash(hash)
}

// isCachedHash tests if the tx is in the buffer
func (b *TxBuffer) isBufferedHash(hash common.Hash) bool {
	return b.GetFromBuffer(hash) != nil
}

func (b *TxBuffer) IsCachedHash(hash common.Hash) bool {
	return b.Syncer.IsCachedHash(hash)
}

func (b *TxBuffer) IsReceivedHash(hash common.Hash) bool {
	return b.IsCachedHash(hash) || b.IsKnownHash(hash)
}

// tryResolve triggered when a Tx is added or resolved by other Tx
// It will check if the given hash has no more dependencies in the cache.
// If so, resolve this hash and try resolve its children
func (b *TxBuffer) tryResolve(tx types.Txi) {
	logrus.Debugf("try to resolve %s", tx)
	for _, parent := range tx.Parents() {
		_, err := b.dependencyCache.GetIFPresent(parent)
		//children := b.children.GetChildren(parent)
		//if len(children) != 0 {
		if err == nil {
			// dependency presents.
			logrus.WithField("parent", parent).WithField("tx", tx).Debugf("cascade resolving is still ongoing")
			return
		}
	}
	// no more dependencies, further check graph structure
	b.niceTx(tx, false)
}

// buildDependencies examines if all ancestors are in our local cache.
// If not, goto fetch it and record it in the map for future reference
// Returns true if all ancestors are local now.
func (b *TxBuffer) buildDependencies(tx types.Txi) bool {
	allFetched := true
	// not in the pool, check its parents
	var sendBloom bool
	for _, parentHash := range tx.Parents() {
		// check if parent is in pool or dag.
		if b.isLocalHash(parentHash) {
			continue
		}

		logrus.WithField("parent", parentHash).WithField("tx", tx).Debugf("parent not known by pool or dag tx")
		allFetched = false
		b.updateDependencyMap(parentHash, tx)
		if b.isBufferedHash(parentHash) {
			logrus.WithField("parent", parentHash).WithField("tx", tx).Debugf("cached by someone before.")
			b.Syncer.Enqueue(nil, common.Hash{}, false)
			continue
		}
		// not in cache, never synced before.
		// sync.
		logrus.WithField("parent", parentHash).WithField("tx", tx).Debugf("enqueue parent to syncer")
		pHash := parentHash
		//b.updateDependencyMap(parentHash, tx)
		if !sendBloom && tx.GetWeight()-b.txPool.GetMaxWeight() > 20 {
			b.Syncer.Enqueue(&pHash, tx.GetTxHash(), true)
			sendBloom = true
		} else {
			b.Syncer.Enqueue(&pHash, tx.GetTxHash(), false)
		}
		//b.children.AddChildren(parentHash, tx.GetTxHash())
	}
	if !allFetched {
		//missingHashes := b.getMissingHashes(tx)
		//logrus.WithField("missingAncestors", missingHashes).WithField("tx", tx).Debugf("tx is pending on ancestors")

		// add myself to the dependency map
		if tx.GetType() == types.TxBaseTypeSequencer {
			seq := tx.(*tx_types.Sequencer)
			//b.Syncer.SyncHashList(seq.GetTxHash())
			//proposing seq
			//why ??
			if seq.Proposing {
				//return allFetched
				b.Syncer.SyncHashList(seq.GetTxHash())
				return allFetched
			}
			//seq.Hash

		}
		b.updateDependencyMap(tx.GetTxHash(), tx)
	}
	return allFetched
}

func (b *TxBuffer) getMissingHashes(txi types.Txi) common.Hashes {
	start := time.Now()
	logrus.WithField("tx", txi).Trace("missing hashes start")
	defer func() {
		logrus.WithField("tx", txi).WithField("time", time.Now().Sub(start)).Trace("missing hashes done")
	}()
	l := common.Hashes{}
	lDedup := map[common.Hash]int{}
	s := map[common.Hash]struct{}{}
	visited := map[common.Hash]struct{}{}
	// find out who is missing
	for _, v := range txi.Parents() {
		l = append(l, v)
		// l.PushBack(v)v
	}

	for len(l) != 0 {
		hash := l[0]
		l = l[1:]
		// hash := l.Remove(l.Front()).(common.Hash)
		if _, ok := visited[hash]; ok {
			// already there, continue
			continue
		}
		visited[hash] = struct{}{}
		if parentTx := b.GetFromAllKnownSource(hash); parentTx != nil {
			for _, v := range parentTx.Parents() {
				if _, ok := lDedup[v]; !ok {
					l = append(l, v)
					// l.PushBack(v)v
					lDedup[v] = 1
				} else {
					lDedup[v] = lDedup[v] + 1
					if lDedup[v]%100 == 0 {
						logrus.WithField("tx", txi).WithField("parent", hash).WithField("pp", v).WithField("times", lDedup[v]).Trace("pushback")
					}
				}
			}
		} else {
			s[hash] = struct{}{}
		}
	}
	var missingHashes common.Hashes
	for k := range s {
		missingHashes = append(missingHashes, k)
	}
	return missingHashes
}

func (b *TxBuffer) releasedTxCacheLoop() {
	for {
		select {
		case tx := <-b.txAddedToPoolChan:
			// tx already received by pool. remove from local cache
			ok := b.knownCache.Remove(tx.GetTxHash())
			if ok {
				logrus.WithField("tx ", tx).Trace("after remove from known Cache")
			}
			// try resolve the remaining txs
			vs, err := b.dependencyCache.GetIFPresent(tx.GetTxHash())
			//children := b.children.GetAndRemove(tx.GetTxHash())
			if err == nil {
				//if len(children) != 0 {
				b.dependencyCache.Remove(tx.GetTxHash())
				for _, v := range vs.(map[common.Hash]types.Txi) {
					if v.GetTxHash() == tx.GetTxHash() {
						//self already resolved
						continue
					}
					txi := v
					if !b.isBufferedHash(v.GetTxHash()) {
						continue

					}

					//for _, hash := range children {
					//var txi types.Txi
					// v, err := b.knownCache.GetIFPresent(hash)
					//if err == nil {
					//continue
					//}
					//txi = v.(types.Txi)
					if !b.isLocalHash(txi.GetTxHash()) {
						b.tryResolve(txi)
						logrus.WithField("resolved", tx).WithField("resolving", v).Debugf("cascade resolving after remove")
					}
					logrus.WithField("resolved", tx).WithField("resolving", v).Debugf("cascade already resolved")
				}

			}
		case <-b.quit:
			logrus.Info("tx buffer releaseCacheLoop received quit message. Quitting...")
			return
		}
	}
}

func (d *TxBuffer) Dump() string {
	var str string
	d.affmu.RLock()
	defer d.affmu.RUnlock()
	vs := d.dependencyCache.GetALL(false)
	for k, v := range vs {
		pHash := k.(common.Hash)
		if v != nil {
			for hash, txi := range v.(map[common.Hash]types.Txi) {
				str += " phash " + pHash.String() + " [ " + hash.String() + txi.String() + " p " + txi.Parents().String() + " ]; "
			}
		}
	}
	return str
}

func (d *TxBuffer) DumpKnownCache() string {
	var str string
	d.affmu.RLock()
	defer d.affmu.RUnlock()
	vs := d.knownCache.GetALL(false)
	for _, v := range vs {
		//pHash := k.(common.Hash)
		if v != nil {
			txi := v.(types.Txi)
			str += txi.String()
		}
	}
	return str
}
