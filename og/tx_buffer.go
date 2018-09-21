package og

import (
	"errors"
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type txStatus int

const (
	txStatusNone       txStatus = iota
	txStatusFetched     // all previous ancestors got
	txStatusValidated   // ancestors are valid
	txStatusConflicted  // ancestors are conflicted, or itself is conflicted
)

type ISyncer interface {
	Enqueue(hash types.Hash)
}
type ITxPool interface {
	Get(hash types.Hash) types.Txi
	AddRemoteTx(tx types.Txi) error
}
type IDag interface {
	GetTx(hash types.Hash) types.Txi
}
type IVerifier interface {
	VerifyHash(t types.Txi) bool
	VerifySignature(t types.Txi) bool
	VerifySourceAddress(t types.Txi) bool
}

// TxBuffer rebuild graph by buffering newly incoming txs and find their parents.
// Tx will be buffered here until parents are got.
// Once the parents are got, Tx will be send to TxPool for further processing.
type TxBuffer struct {
	dag             IDag
	verifier        IVerifier
	syncer          ISyncer
	txPool          ITxPool
	dependencyCache gcache.Cache // list of hashes that are pending on the parent. map[types.Hash]map[types.Hash]types.Tx
	affmu           sync.RWMutex
	newTxChan       chan types.Txi
	quit            chan bool
	Hub             *Hub
}

func (b *TxBuffer) GetBenchmarks() map[string]int {
	return map[string]int{
		"newTxChan": len(b.newTxChan),
	}
}

type TxBufferConfig struct {
	Dag                              IDag
	Verifier                         IVerifier
	Syncer                           ISyncer
	TxPool                           ITxPool
	DependencyCacheMaxSize           int
	DependencyCacheExpirationSeconds int
	NewTxQueueSize                   int
}

func NewTxBuffer(config TxBufferConfig) *TxBuffer {
	return &TxBuffer{
		dag:      config.Dag,
		verifier: config.Verifier,
		syncer:   config.Syncer,
		txPool:   config.TxPool,
		dependencyCache: gcache.New(config.DependencyCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.DependencyCacheExpirationSeconds)).Build(),
		newTxChan: make(chan types.Txi, config.NewTxQueueSize),
		quit:      make(chan bool),
	}
}

func (b *TxBuffer) Start() {
	go b.loop()
}

func (b *TxBuffer) Stop() {
	b.quit <- true
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
		case v := <-b.newTxChan:
			go b.handleTx(v)
		}
	}
}

// AddTx is called once there are new tx coming in.
func (b *TxBuffer) AddTx(tx types.Txi) {
	b.newTxChan <- tx
}

// niceTx
func (b *TxBuffer) niceTx(tx types.Txi, firstTime bool) {
	// Check if the tx is valid based on graph structure rules
	// Only txs that are obeying rules will be added to the graph.
	logrus.WithField("tx", tx).Info("nice tx")
	if !b.VerifyGraphStructure(tx) {
		logrus.WithField("tx", tx).Info("bad graph tx")
		return
	}
	// resolve other dependencies
	b.resolve(tx, firstTime)
}

// in parallel
func (b *TxBuffer) handleTx(tx types.Txi) {
	logrus.WithField("tx", tx).Debugf("buffer is handling tx")
	var shoudBrodcast bool
	// already in the dag or tx_pool.
	if b.isKnownHash(tx.GetTxHash()) {
		return
	} else {
		// not in tx buffer , a new tx , shoud brodcast
		shoudBrodcast = true
	}
	// TODO: Temporarily comment it out to test performance.
	//if err := b.verifyTxFormat(tx); err != nil {
	//	logrus.WithError(err).WithField("tx", tx).Debugf("buffer received invalid tx")
	//	return
	//}

	if b.buildDependencies(tx) {
		// directly fulfilled, insert into txpool
		// needs to resolve itself first
		logrus.WithField("tx", tx).Debugf("new tx directly fulfilled in buffer")
		b.niceTx(tx, true)
	}

	if shoudBrodcast {
		b.sendMessage(tx)
	}

}

func (b *TxBuffer) GetFromBuffer(hash types.Hash) types.Txi {
	a, err := b.dependencyCache.GetIFPresent(hash)
	if err == nil {
		return a.(map[types.Hash]types.Txi)[hash]
	}
	return nil
}

func (b *TxBuffer) InBuffer(tx types.Txi) bool {
	_, err := b.dependencyCache.GetIFPresent(tx.GetTxHash())
	if err == nil {
		return true
	}
	return false
}

//sendMessage  brodcase txi message
func (b *TxBuffer) sendMessage(txi types.Txi) {
	txType := txi.GetType()
	if txType == types.TxBaseTypeNormal {
		tx := txi.(*types.Tx)
		msgTx := types.MessageNewTx{Tx: tx}
		data, _ := msgTx.MarshalMsg(nil)
		b.Hub.SendMessage(MessageTypeNewTx, data)
	} else if txType == types.TxBaseTypeSequencer {
		seq := txi.(*types.Sequencer)
		msgTx := types.MessageNewSequence{seq}
		data, _ := msgTx.MarshalMsg(nil)
		b.Hub.SendMessage(MessageTypeNewSequence, data)
	} else {
		logrus.Warn("never come here ,unkown tx type", txType)
	}

}

// updateDependencyMap will update dependency relationship currently known.
// e.g., If there is already (c <- b), adding (c <- a) will result in (c <- [a,b]).
func (b *TxBuffer) updateDependencyMap(parentHash types.Hash, self types.Txi) {
	if self == nil {
		logrus.WithFields(logrus.Fields{
			"parent": parentHash.String(),
			"child":  nil,
		}).Infof("updating dependency map")
	} else {
		logrus.WithFields(logrus.Fields{
			"parent": parentHash.String(),
			"child":  self.String(),
		}).Infof("updating dependency map")
	}

	b.affmu.Lock()
	_, err := b.dependencyCache.GetIFPresent(parentHash)
	if err != nil {
		// key not present, need to build an inner map
		b.dependencyCache.Set(parentHash, map[types.Hash]types.Txi{self.GetBase().Hash: self})
	}
	b.affmu.Unlock()
}

// resolve is called when all ancestors of the tx is got.
// Once resolved, add it to the pool
func (b *TxBuffer) resolve(tx types.Txi, firstTime bool) {
	vs, err := b.dependencyCache.GetIFPresent(tx.GetTxHash())
	b.txPool.AddRemoteTx(tx)
	b.dependencyCache.Remove(tx.GetTxHash())
	logrus.WithField("tx", tx).Debugf("tx resolved")

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

// verifyTxFormat checks if the signatures and hashes are correct in tx
func (b *TxBuffer) verifyTxFormat(tx types.Txi) error {
	if !b.verifier.VerifyHash(tx) {
		return errors.New("hash is not valid")
	}
	if !b.verifier.VerifySignature(tx) {
		return errors.New("signature is not valid")
	}
	// TODO: Nonce
	return nil
}

func (b *TxBuffer) isKnownHash(hash types.Hash) bool {
	logrus.WithFields(logrus.Fields{
		"Txpool": b.txPool.Get(hash),
		"DAG":    b.dag.GetTx(hash),
		"Hash":   hash,
		"Buffer": b.GetFromBuffer(hash),
	}).Info("transaction location")

	if b.txPool.Get(hash) != nil {
		return true
	}
	if b.dag.GetTx(hash) != nil {
		return true
	}
	if b.GetFromBuffer(hash) != nil {
		return true
	}
	return false
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
		if !b.isKnownHash(parentHash) {
			logrus.WithField("hash", parentHash).Infof("parent not known by pool or dag tx")
			allFetched = false

			b.updateDependencyMap(parentHash, tx)
			logrus.Infof("enqueue parent to syncer: %s", parentHash)
			b.syncer.Enqueue(parentHash)
		}
	}
	if !allFetched {
		// add myself to the dependency map
		b.updateDependencyMap(tx.GetTxHash(), tx)
	}
	return allFetched
}
func (b *TxBuffer) VerifyGraphStructure(txi types.Txi) bool {
	return true
}
