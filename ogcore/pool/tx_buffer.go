package pool

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/protocol"
	"github.com/annchain/gcache"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type TxBufferConfig struct {
	//Dag                              ILedger
	//Verifiers                        []protocol.Verifier
	//Syncer                           Syncer
	//TxAnnouncer                      Announcer
	//TxPool                           ITxPool
	DependencyCacheMaxSize           int
	DependencyCacheExpirationSeconds int
	NewTxQueueSize                   int
	KnownCacheMaxSize                int
	KnownCacheExpirationSeconds      int
	AddedToPoolQueueSize             int
	TestNoVerify                     bool
}

// TxBuffer rebuild graph by buffering newly incoming txs and find their parents.
// Txi will be buffered here until parents are got.
// Once the parents are got, Txi will be send to TxPool for further processing.
type TxBuffer struct {
	debuglog.NodeLogger
	Verifiers              []protocol.Verifier
	PoolHashLocator        PoolHashLocator
	LedgerHashLocator      LedgerHashLocator
	LocalGraphInfoProvider LocalGraphInfoProvider
	EventBus               eventbus.EventBus
	knownCache             gcache.Cache // txs that are already fulfilled and pushed to txpool
	dependencyCache        gcache.Cache // list of hashes that are pending on the parent. map[common.Hash]map[common.Hash]types.Txi
	affmu                  sync.RWMutex
	newTxChan              chan types.Txi
	quit                   chan bool
}

func (t *TxBuffer) InitDefault(config TxBufferConfig) {
	t.quit = make(chan bool)
	t.newTxChan = make(chan types.Txi)
	t.knownCache = gcache.New(config.KnownCacheMaxSize).Simple().
		Expiration(time.Second * time.Duration(config.KnownCacheExpirationSeconds)).Build()
	t.dependencyCache = gcache.New(config.DependencyCacheMaxSize).Simple().
		Expiration(time.Second * time.Duration(config.DependencyCacheExpirationSeconds)).Build()
}

func (t *TxBuffer) Start() {
	go t.loop()
}

func (t *TxBuffer) Stop() {
	close(t.quit)
}

func (t *TxBuffer) Name() string {
	return "TxBuffer"
}

func (t *TxBuffer) HandleEvent(ev eventbus.Event) {
	switch ev.GetEventType() {
	case events.TxReceivedEventType:
		tx := ev.(*events.TxReceivedEvent).Tx
		<-ffchan.NewTimeoutSenderShort(t.newTxChan, tx, "txbufferchan").C
		//t.newTxChan <- tx // send to buffer
	case events.SequencerReceivedEventType:
		seq := ev.(*events.SequencerReceivedEvent).Sequencer
		t.newTxChan <- seq
	case events.NewTxLocallyGeneratedEventType:
		evt := ev.(*events.NewTxLocallyGeneratedEvent)
		t.newTxChan <- evt.Tx // send to buffer
	default:
		t.Logger.WithField("type", ev.GetEventType()).Warn("event type not supported by txbuffer")
		return
	}
}

func (o *TxBuffer) HandlerDescription(ev eventbus.EventType) string {
	switch ev {
	case events.TxReceivedEventType:
		return "SendTxToBuffer"
	case events.SequencerReceivedEventType:
		return "SendSeqToBuffer"
	case events.NewTxLocallyGeneratedEventType:
		return "SendLocalTxToBuffer"
	default:
		return "N/A"
	}
}

func (b *TxBuffer) DumpUnsolved() {
	ever := false
	for k, v := range b.dependencyCache.GetALL(true) {
		for k1 := range v.(map[common.Hash]types.Txi) {
			b.Logger.Warnf("not fulfilled: %s <- %s", k.(common.Hash), k1)
			ever = true
		}
	}
	if !ever {
		b.Logger.Debug("Dependency all clear")
	}
}

// PendingLen returns the txs that is pending sync in the cache
func (b *TxBuffer) PendingLen() int {
	return b.dependencyCache.Len(false)
}

func (b *TxBuffer) loop() {
	for {
		b.Logger.Info("looping")
		select {
		case <-b.quit:
			b.Logger.Info("tx buffer received quit Message. Quitting...")
			return
		case v := <-b.newTxChan:
			b.handleTx(v)
		}
	}
}

// in parallel
func (b *TxBuffer) handleTx(tx types.Txi) {
	b.Logger.WithField("tx", tx).WithField("parents", tx.GetParents()).Debugf("buffer is handling tx")
	start := time.Now()
	defer func() {
		b.Logger.WithField("ts", time.Now().Sub(start)).WithField("tx", tx).WithField("parents", tx.GetParents()).Debugf("buffer handled tx")
		// b.Logger.WithField("tx", tx).Debugf("buffer handled tx")
	}()
	// already in the Dag or tx_pool or buffer itself.
	if b.IsKnownHash(tx.GetHash()) {
		return
	}
	//if err := b.verifyTxFormat(tx); err != nil {
	//	b.Logger.WithError(err).WithField("tx", tx).Debugf("buffer received invalid tx")
	//	return
	//}
	for _, verifier := range b.Verifiers {
		if !verifier.Independent() {
			continue
		}
		if !verifier.Verify(tx) {
			b.Logger.WithField("tx", tx).WithField("verifier", verifier).Warn("bad tx")
			return
		}
	}

	b.knownCache.Set(tx.GetHash(), tx)

	if b.buildDependencies(tx) {
		// directly fulfilled, insert into txpool
		// needs to resolve itself first
		b.Logger.WithField("tx", tx).Debugf("new tx directly fulfilled in buffer")
		b.niceTx(tx, true)
	}
}

// isKnownHash tests if the tx is ever heard of, either in local or in buffer.
// if tx is known, do not broadcast anymore
func (b *TxBuffer) IsKnownHash(hash common.Hash) bool {
	return b.isBufferedHash(hash) || b.PoolHashLocator.IsLocalHash(hash)
}

// buildDependencies examines if all ancestors are in our local cache.
// If not, goto fetch it and record it in the map for future reference
// Returns true if all ancestors are local now.
func (b *TxBuffer) buildDependencies(tx types.Txi) bool {
	allFetched := true
	// not in the pool, check its parents
	//var sendBloom bool
	for _, parentHash := range tx.GetParents() {
		if !b.isLocalHash(parentHash) {
			b.Logger.WithField("parent", parentHash).WithField("tx", tx).Debugf("parent not known by pool or dag tx")
			allFetched = false
			//this line is for test , may be can fix
			b.updateDependencyMap(parentHash, tx)
			// TODO: identify if this tx is already synced
			if !b.isBufferedHash(parentHash) {
				// not in cache, never synced before.
				// sync.
				//b.Logger.WithField("parent", parentHash).WithField("tx", tx).Debugf("enqueue parent to syncer")
				pHash := parentHash
				//b.updateDependencyMap(parentHash, tx)
				//maxWeight := b.LocalGraphInfoProvider.GetMaxWeight()
				//b.EventBus.Route(&events.NeedSyncTxEvent{
				//	ParentHash:      pHash,
				//	ChildHash:       tx.GetHash(),
				//	SendBloomfilter: false,
				//})
				b.EventBus.Route(&events.NeedSyncTxEvent{
					Hash:            pHash,
					SpecifiedSource: nil,
				})
				// Weight is unknown until parents are got. So here why check this?
				//
				//if !sendBloom && tx.GetWeight() > maxWeight && tx.GetWeight()-maxWeight > 20 {
				//	b.EventBus.Route(&events.NeedSyncTxEvent{
				//		ParentHash:      pHash,
				//		ChildHash:       tx.GetHash(),
				//		SendBloomfilter: true,
				//	})
				//	//b.Syncer.Enqueue(&pHash, tx.GetHash(), true)
				//	sendBloom = true
				//} else {
				//	b.EventBus.Route(&events.NeedSyncTxEvent{
				//		ParentHash:      pHash,
				//		ChildHash:       tx.GetHash(),
				//		SendBloomfilter: false,
				//	})
				//	//b.Syncer.Enqueue(&pHash, tx.GetHash(), false)
				//}
				//b.children.AddChildren(parentHash, tx.GetHash())
			} else {
				b.Logger.WithField("parent", parentHash).WithField("tx", tx).Debugf("cached by someone before.")
				// TODO: check consequence
				//b.Syncer.Enqueue(nil, common.Hash{}, false)
			}
		}
	}
	if !allFetched {
		//missingHashes := b.getMissingHashes(tx)
		//b.Logger.WithField("missingAncestors", missingHashes).WithField("tx", tx).Debugf("tx is pending on ancestors")

		// add myself to the dependency map
		//if tx.GetType() == types.TxBaseTypeSequencer {
		//	seq := tx.(*types.Sequencer)
		//	//b.Syncer.SyncHashList(seq.GetHash())
		//	//proposing seq
		//	//why ??
		//	if seq.Proposing {
		//		//return allFetched
		//		b.Syncer.SyncHashList(seq.GetHash())
		//		return allFetched
		//	}
		//	//seq.Hash
		//
		//}
		b.updateDependencyMap(tx.GetHash(), tx)
	}
	return allFetched
}
func (b *TxBuffer) isLocalHash(hash common.Hash) bool {
	return b.PoolHashLocator.IsLocalHash(hash) || b.LedgerHashLocator.IsLocalHash(hash)
}

// updateDependencyMap will update dependency community currently known.
// e.g., If there is already (c <- b), adding (c <- a) will result in (c <- [a,b]).
func (b *TxBuffer) updateDependencyMap(parentHash common.Hash, self types.Txi) {
	if self == nil {
		b.Logger.WithFields(logrus.Fields{
			"parent": parentHash,
			"child":  nil,
		}).Debugf("updating dependency map")
	} else {
		b.Logger.WithFields(logrus.Fields{
			"parent": parentHash,
			"child":  self,
		}).Debugf("updating dependency map")
	}

	b.affmu.Lock()
	v, err := b.dependencyCache.GetIFPresent(parentHash)

	if err != nil {
		// key not present, need to build an inner map
		v = map[common.Hash]types.Txi{self.GetHash(): self}
	}
	v.(map[common.Hash]types.Txi)[self.GetHash()] = self
	b.dependencyCache.Set(parentHash, v)

	b.affmu.Unlock()
}

// isCachedHash tests if the tx is in the buffer
func (b *TxBuffer) isBufferedHash(hash common.Hash) bool {
	return b.GetFromBuffer(hash) != nil
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
	if poolTx := b.PoolHashLocator.Get(hash); poolTx != nil {
		return poolTx
	} else if dagTx := b.LedgerHashLocator.GetTx(hash); dagTx != nil {
		return dagTx
	}
	return nil
}

// niceTx is the logic triggered when tx's ancestors are all fetched to local
func (b *TxBuffer) niceTx(tx types.Txi, firstTime bool) {
	// Check if the tx is valid based on graph structure rules
	// Only txs that are obeying rules will be added to the graph.
	b.knownCache.Remove(tx.GetHash())

	// added verifier for specific tx types. e.g. Campaign, TermChange.
	for _, verifier := range b.Verifiers {
		if verifier.Independent() {
			continue
		}
		if !verifier.Verify(tx) {
			b.Logger.WithField("tx", tx).WithField("verifier", verifier).Warn("bad tx")
			return
		}
	}
	b.Logger.WithField("tx", tx).Debugf("nice tx")
	// resolve other dependencies
	b.resolve(tx, firstTime)
}

// resolve is called when all ancestors of the tx is got.
// Once resolved, add it to the pool
func (b *TxBuffer) resolve(tx types.Txi, firstTime bool) {
	vs, err := b.dependencyCache.GetIFPresent(tx.GetHash())
	//children := b.children.GetAndRemove(tx.GetHash())
	b.Logger.WithField("tx", tx).Trace("after cache GetIFPresent")

	// announcer and txpool both listen to this event.
	b.EventBus.Route(&events.NewTxiDependencyFulfilledEvent{Txi: tx})
	//addErr := b.addToTxPool(tx)
	//if addErr != nil {
	//	b.Logger.WithField("txi", tx).WithError(addErr).Warn("add tx to txpool err")
	//} else {
	//
	//	//b.Announcer.BroadcastNewTx(tx)
	//}
	b.dependencyCache.Remove(tx.GetHash())

	b.Logger.WithField("tx", tx).Debugf("tx resolved")

	if err != nil {
		// if len(children) == 0 {
		// key not present, already resolved.
		if firstTime {
			b.Logger.WithField("tx", tx).Debug("new local tx")
		} else {
			b.Logger.WithField("tx", tx).Warn("already been resolved before")
		}
		return
	}
	// try resolve the remainings
	for _, v := range vs.(map[common.Hash]types.Txi) {
		if v.GetHash() == tx.GetHash() {
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
		b.Logger.WithField("resolved", tx).WithField("resolving", txi).Debugf("cascade resolving")
		b.tryResolve(txi)
	}
}

//func (b *TxBuffer) addToTxPool(tx types.Txi) error {
//	//no need to receive added tx by buffer
//	return b.txPool.AddRemoteTx(tx, true)
//}

// tryResolve triggered when a Txi is added or resolved by other Txi
// It will check if the given Hash has no more dependencies in the cache.
// If so, resolve this Hash and try resolve its children
func (b *TxBuffer) tryResolve(tx types.Txi) {
	b.Logger.Debugf("try to resolve %s", tx)
	for _, parent := range tx.GetParents() {
		_, err := b.dependencyCache.GetIFPresent(parent)
		//children := b.children.GetChildren(parent)
		//if len(children) != 0 {
		if err == nil {
			// dependency presents.
			b.Logger.WithField("parent", parent).WithField("tx", tx).Debugf("cascade resolving is still ongoing")
			return
		}
	}
	// no more dependencies, further check graph structure
	b.niceTx(tx, false)
}
