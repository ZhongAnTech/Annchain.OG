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
			logrus.Info("TxBuffer received quit message. Quitting...")
			return
		case v := <-b.newTxChan:
			b.handleTx(v)
		}
	}
}

// AddTx is called once there are new tx coming in.
func (b *TxBuffer) AddTx(tx types.Txi) {
	b.newTxChan <- tx
}

func (b *TxBuffer) handleTx(tx types.Txi) {
	// already in the dag or tx_pool.
	if b.isKnownHash(tx.GetTxHash()) {
		return
	}
	if err := b.verifyTxFormat(tx); err != nil {
		logrus.WithError(err).Debugf("Received invalid tx %s: %s", tx.GetTxHash().Hex())
		return
	}

	if b.fetchAllAncestors(tx) {
		// already fulfilled, insert into txpool
		// needs to resolve itself first
		logrus.Debugf("New tx fulfilled: %s", tx.GetTxHash().Hex())
		b.resolve(tx)
		b.txPool.AddRemoteTx(tx)
	}
}

// updateDependencyMap will update dependency relationship currently known.
// e.g., If there is already (c <- b), adding (c <- a) will result in (c <- [a,b]).
func (b *TxBuffer) updateDependencyMap(parentHash types.Hash, self types.Txi) {
	logrus.Infof("Updating dependency map: %s <- %s", parentHash.Hex()[:4], self.GetTxHash().Hex())
	v, err := b.dependencyCache.GetIFPresent(parentHash)
	if err != nil {
		// key not present, need to build a inner map
		b.dependencyCache.Set(parentHash, map[types.Hash]types.Txi{self.GetBase().Hash: self})
	} else {
		v.(map[types.Hash]types.Txi)[self.GetBase().Hash] = self
	}
}

// resolve is called when all ancestors of the tx is got.
func (b *TxBuffer) resolve(tx types.Txi) {
	logrus.Debugf("Resolve %s", tx.GetTxHash().Hex())
	vs, err := b.dependencyCache.GetIFPresent(tx.GetTxHash())
	if err != nil {
		// key not present, already resolved.
		return
	}

	b.dependencyCache.Remove(tx.GetTxHash())
	// try resolve the remainings
	for _, v := range vs.(map[types.Hash]types.Txi) {
		logrus.Debugf("Resolving %s because %s is resolved", v.GetTxHash().Hex(), tx.GetTxHash().Hex())
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
	return b.txPool.Get(hash) != nil || b.dag.GetTx(hash) != nil
}

// tryResolve triggered when a Tx is added or resolved by other Tx
// It will check if the given hash has no more dependencies in the cache.
// If so, resolve this hash and try resolve its children
func (b *TxBuffer) tryResolve(tx types.Txi) bool {
	logrus.Debugf("Try to resolve %s", tx.GetTxHash().Hex())
	for _, parent := range tx.Parents() {
		_, err := b.dependencyCache.GetIFPresent(parent)
		if err == nil {
			// dependency presents.
			logrus.Debugf("Cannot be resolved because %s is still a dependency of %s", parent.Hex(), tx.GetTxHash().Hex())
			return false
		}
	}
	// no more dependencies
	b.resolve(tx)
	return true
}

// fetchAllAncestors examine if all ancestors are in our local cache.
// If not, go fetch it and record it in the map for future reference
// Returns true if all ancestors are local now.
func (b *TxBuffer) fetchAllAncestors(tx types.Txi) bool {
	allFetched := true
	// not in the pool, check its parents
	for _, parentHash := range tx.Parents() {
		if !b.isKnownHash(parentHash) {
			logrus.Infof("Hash not known by buffer tx: %s", parentHash.Hex())
			allFetched = false

			b.updateDependencyMap(parentHash, tx)
			logrus.Infof("Enqueue tx to syncer: %s", parentHash.Hex())
			b.syncer.Enqueue(parentHash)
		}
	}
	return allFetched
}
