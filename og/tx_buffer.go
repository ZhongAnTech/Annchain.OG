package og

import (
	"sync"
	"time"

	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
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
	Enqueue(hash types.Hash)
	ClearQueue()
}
type Announcer interface {
	BroadcastNewTx(txi types.Txi)
}
type ITxPool interface {
	Get(hash types.Hash) types.Txi
	AddRemoteTx(tx types.Txi) error
	RegisterOnNewTxReceived(c chan types.Txi, name string)
	GetLatestNonce(addr types.Address) (uint64, error)
	IsLocalHash(hash types.Hash) bool
}
type IDag interface {
	GetTx(hash types.Hash) types.Txi
	GetTxByNonce(addr types.Address, nonce uint64) types.Txi
	GetSequencerById(id uint64) *types.Sequencer
	GetTxsByNumber(id uint64) []*types.Tx
	LatestSequencer() *types.Sequencer
	GetSequencer(hash types.Hash, id uint64) *types.Sequencer
	Genesis() *types.Sequencer
	GetSequencerByHash(hash types.Hash) *types.Sequencer
}

// TxBuffer rebuild graph by buffering newly incoming txs and find their parents.
// Tx will be buffered here until parents are got.
// Once the parents are got, Tx will be send to TxPool for further processing.
type TxBuffer struct {
	dag                    IDag
	verifiers              []Verifier
	Syncer                 Syncer
	Announcer              Announcer
	txPool                 ITxPool
	dependencyCache        gcache.Cache // list of hashes that are pending on the parent. map[types.Hash]map[types.Hash]types.Tx
	affmu                  sync.RWMutex
	SelfGeneratedNewTxChan chan types.Txi
	ReceivedNewTxChan      chan types.Txi
	quit                   chan bool
	knownCache             gcache.Cache // txs that are already fulfilled and pushed to txpool
	txAddedToPoolChan      chan types.Txi
}

func (b *TxBuffer) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"selfGeneratedNewTxChan": len(b.SelfGeneratedNewTxChan),
		"receivedNewTxChan":      len(b.ReceivedNewTxChan),
		"dependencyCache":        b.dependencyCache.Len(),
		"knownCache":             b.knownCache.Len(),
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
}

func NewTxBuffer(config TxBufferConfig) *TxBuffer {
	return &TxBuffer{
		dag:       config.Dag,
		verifiers: config.Verifiers,
		Syncer:    config.Syncer,
		Announcer: config.TxAnnouncer,
		txPool:    config.TxPool,
		dependencyCache: gcache.New(config.DependencyCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.DependencyCacheExpirationSeconds)).Build(),
		SelfGeneratedNewTxChan: make(chan types.Txi, config.NewTxQueueSize),
		ReceivedNewTxChan:      make(chan types.Txi, config.NewTxQueueSize),
		txAddedToPoolChan:      make(chan types.Txi, config.NewTxQueueSize),
		quit:                   make(chan bool),
		knownCache: gcache.New(config.KnownCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.KnownCacheExpirationSeconds)).Build(),
	}
}

func (b *TxBuffer) Start() {
	b.txPool.RegisterOnNewTxReceived(b.txAddedToPoolChan, "b.txAddedToPoolChan")
	go b.loop()
	go b.releasedTxCacheLoop()
}

func (b *TxBuffer) Stop() {
	logrus.Info("tx bu will stop.")
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
		case v := <-b.SelfGeneratedNewTxChan:
			b.handleTx(v)
		}
	}
}

// niceTx is the logic triggered when tx's ancestors are all fetched to local
func (b *TxBuffer) niceTx(tx types.Txi, firstTime bool) {
	// Check if the tx is valid based on graph structure rules
	// Only txs that are obeying rules will be added to the graph.
	for _, verifier := range b.verifiers {
		if !verifier.Verify(tx) {
			logrus.WithField("tx", tx).WithField("verifier", verifier.Name()).Warn("bad tx")
			return
		}
	}

	logrus.WithField("tx", tx).Debugf("nice tx")
	// resolve other dependencies
	b.resolve(tx, firstTime)
}

// in parallel
func (b *TxBuffer) handleTx(tx types.Txi) {
	logrus.WithField("tx", tx).WithField("parents", types.HashesToString(tx.Parents())).Debugf("buffer is handling tx")
	start := time.Now()
	defer func() {
		logrus.WithField("ts", time.Now().Sub(start)).WithField("tx", tx).WithField("parents", types.HashesToString(tx.Parents())).Debugf("buffer handled tx")
		// logrus.WithField("tx", tx).Debugf("buffer handled tx")
	}()

	// already in the dag or tx_pool or buffer itself.
	if b.isKnownHash(tx.GetTxHash()) {
		return
	}
	// not in tx buffer , a new tx , shoud broadcast

	// TODO: Temporarily comment it out to test performance.
	//if err := b.verifyTxFormat(tx); err != nil {
	//	logrus.WithError(err).WithField("tx", tx).Debugf("buffer received invalid tx")
	//	return
	//}

	b.knownCache.Set(tx.GetTxHash(), tx)

	if b.buildDependencies(tx) {
		// directly fulfilled, insert into txpool
		// needs to resolve itself first
		logrus.WithField("tx", tx).Debugf("new tx directly fulfilled in buffer")
		b.niceTx(tx, true)
	}
}

func (b *TxBuffer) GetFromBuffer(hash types.Hash) types.Txi {
	a, err := b.knownCache.GetIFPresent(hash)
	if err == nil {
		return a.(types.Txi)
	}
	return nil
}

func (b *TxBuffer) GetFromAllKnownSource(hash types.Hash) types.Txi {
	if tx := b.GetFromBuffer(hash); tx != nil {
		return tx
	} else if tx := b.GetFromProviders(hash); tx != nil {
		return tx
	}
	return nil
}

func (b *TxBuffer) GetFromProviders(hash types.Hash) types.Txi {
	if poolTx := b.txPool.Get(hash); poolTx != nil {
		return poolTx
	} else if dagTx := b.dag.GetTx(hash); dagTx != nil {
		return dagTx
	}
	return nil
}

// updateDependencyMap will update dependency relationship currently known.
// e.g., If there is already (c <- b), adding (c <- a) will result in (c <- [a,b]).
func (b *TxBuffer) updateDependencyMap(parentHash types.Hash, self types.Txi) {
	if self == nil {
		logrus.WithFields(logrus.Fields{
			"parent": parentHash.String(),
			"child":  nil,
		}).Debugf("updating dependency map")
	} else {
		logrus.WithFields(logrus.Fields{
			"parent": parentHash.String(),
			"child":  self.String(),
		}).Debugf("updating dependency map")
	}

	b.affmu.Lock()
	v, err := b.dependencyCache.GetIFPresent(parentHash)

	if err != nil {
		// key not present, need to build an inner map
		v = map[types.Hash]types.Txi{self.GetBase().Hash: self}
	}
	v.(map[types.Hash]types.Txi)[self.GetBase().Hash] = self
	b.dependencyCache.Set(parentHash, v)

	b.affmu.Unlock()
}

func (b *TxBuffer) addToTxPool(tx types.Txi) error {
	return b.txPool.AddRemoteTx(tx)
}

// resolve is called when all ancestors of the tx is got.
// Once resolved, add it to the pool
func (b *TxBuffer) resolve(tx types.Txi, firstTime bool) {
	vs, err := b.dependencyCache.GetIFPresent(tx.GetTxHash())
	addErr := b.addToTxPool(tx)
	if addErr != nil {
		logrus.WithField("txi", tx).WithError(addErr).Warn("add tx to txpool err")
	} else {
		b.Announcer.BroadcastNewTx(tx)
	}
	b.dependencyCache.Remove(tx.GetTxHash())

	if err != nil {
		// key not present, already resolved.
		if firstTime {
			logrus.WithField("tx", tx).Debug("new local tx")
		} else {
			logrus.WithField("tx", tx).Warn("already been resolved before")
		}
		return
	}
	// try resolve the remainings
	for _, v := range vs.(map[types.Hash]types.Txi) {
		if v.GetTxHash() == tx.GetTxHash() {
			// self already resolved
			continue
		}
		logrus.WithField("resolved", tx).WithField("resolving", v).Debugf("cascade resolving")
		b.tryResolve(v)
	}
}

// isLocalHash tests if the tx has already been in the txpool or dag.
func (b *TxBuffer) isLocalHash(hash types.Hash) bool {
	//just get once
	var poolTx, dagTx types.Txi
	ok := false
	if poolTx = b.txPool.Get(hash); poolTx != nil {
		ok = true
	} else if dagTx = b.dag.GetTx(hash); dagTx != nil {
		ok = true
	}
	logrus.WithFields(logrus.Fields{
		"TxPool": poolTx,
		"DAG":    dagTx,
		"Hash":   hash,
		//"Buffer": b.GetFromBuffer(hash),
	}).Trace("transaction location")
	return ok
}

// isKnownHash tests if the tx is ever heard of, either in local or in buffer.
// if tx is known, do not broadcast anymore
func (b *TxBuffer) isKnownHash(hash types.Hash) bool {
	return b.isLocalHash(hash) || b.isCachedHash(hash)
}

// isCachedHash tests if the tx is in the buffer
func (b *TxBuffer) isCachedHash(hash types.Hash) bool {
	return b.GetFromBuffer(hash) != nil
}

// tryResolve triggered when a Tx is added or resolved by other Tx
// It will check if the given hash has no more dependencies in the cache.
// If so, resolve this hash and try resolve its children
func (b *TxBuffer) tryResolve(tx types.Txi) {
	logrus.Debugf("try to resolve %s", tx.String())
	for _, parent := range tx.Parents() {
		_, err := b.dependencyCache.GetIFPresent(parent)
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
// If not, go fetch it and record it in the map for future reference
// Returns true if all ancestors are local now.
func (b *TxBuffer) buildDependencies(tx types.Txi) bool {
	allFetched := true
	// not in the pool, check its parents
	for _, parentHash := range tx.Parents() {
		if !b.isLocalHash(parentHash) {
			logrus.WithField("parent", parentHash).WithField("tx", tx).Debugf("parent not known by pool or dag tx")
			allFetched = false

			// TODO: identify if this tx is already synced
			if !b.isCachedHash(parentHash) {
				// not in cache, never synced before.
				// sync.
				logrus.WithField("parent", parentHash).WithField("tx", tx).Debugf("enqueue parent to syncer")
				b.Syncer.Enqueue(parentHash)
				b.updateDependencyMap(parentHash, tx)
			} else {
				logrus.WithField("parent", parentHash).WithField("tx", tx).Debugf("cached by someone before.")
			}
		}
	}
	if !allFetched {
		//missingHashes := b.getMissingHashes(tx)
		//logrus.WithField("missingAncestors", missingHashes).WithField("tx", tx).Debugf("tx is pending on ancestors")

		// add myself to the dependency map
		b.updateDependencyMap(tx.GetTxHash(), tx)
	}
	return allFetched
}
func (b *TxBuffer) getMissingHashes(txi types.Txi) []types.Hash {
	start := time.Now()
	logrus.WithField("tx", txi).Trace("missing hashes start")
	defer func() {
		logrus.WithField("tx", txi).WithField("time", time.Now().Sub(start)).Trace("missing hashes done")
	}()
	l := []types.Hash{}
	lDedup := map[types.Hash]int{}
	s := map[types.Hash]struct{}{}
	visited := map[types.Hash]struct{}{}
	// find out who is missing
	for _, v := range txi.Parents() {
		l = append(l, v)
		// l.PushBack(v)v
	}

	for len(l) != 0 {
		hash := l[0]
		l = l[1:]
		// hash := l.Remove(l.Front()).(types.Hash)
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
						logrus.WithField("tx", txi).WithField("parent", hash.String()).WithField("pp", v.String()).WithField("times", lDedup[v]).Trace("pushback")
					}
				}
			}
		} else {
			s[hash] = struct{}{}
		}
	}
	var missingHashes []types.Hash
	for k := range s {
		missingHashes = append(missingHashes, k)
	}
	return missingHashes
}
func (b *TxBuffer) releasedTxCacheLoop() {
	for {
		select {
		case v := <-b.txAddedToPoolChan:
			//a,err:= 	b.releasedTxCache.Get(v.GetTxHash())
			//if err!=nil {
			//	// the tx added not by tx buffer , clean dependency
			//	tx := a.(types.Txi)
			//	tx.String()
			//	//todo  resolve the tx remove dependency
			//}

			// tx already received by pool. remove from local cache
			b.knownCache.Remove(v.GetTxHash())
		case <-b.quit:
			logrus.Info("tx buffer releaseCacheLoop received quit message. Quitting...")
			return
		}
	}
}
