package core

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"sync"
	"errors"
	"time"
	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
)

type txStatus int

const (
	txStatusNone       txStatus = iota
	txStatusFetched     // all previous ancestors got
	txStatusValidated   // ancestors are valid
	txStatusConflicted  // ancestors are conflicted, or itself is conflicted
)

// TxBuffer rebuild graph by buffering newly incoming txs and find their parents.
// Tx will be buffered here until parents are got.
// Once the parents are got, Tx will be send to TxPool for further processing.
// TxBuffer will
type TxBuffer struct {
	dag             IDag
	verifier        *og.Verifier
	syncer          *og.Syncer
	txPool          *TxPool
	dependencyCache gcache.Cache // list of hashes that are pending on the parent. map[types.Hash]map[types.Hash]types.Tx
	affmu           sync.RWMutex
}

type TxBufferConfig struct {
	Dag                              IDag
	Verifier                         *og.Verifier
	Syncer                           *og.Syncer
	TxPool                           *TxPool
	DependencyCacheMaxSize           int
	DependencyCacheExpirationSeconds int
}

func NewTxBuffer(config TxBufferConfig) *TxBuffer {
	return &TxBuffer{
		dag:      config.Dag,
		verifier: config.Verifier,
		syncer:   config.Syncer,
		txPool:   config.TxPool,
		dependencyCache: gcache.New(config.DependencyCacheMaxSize).LRU().
			Expiration(time.Second * time.Duration(config.DependencyCacheExpirationSeconds)).Build(),
	}
}

// AddTx is called once there are new tx coming in.
func (buffer *TxBuffer) AddTx(tx types.Txi) {
	buffer.affmu.Lock()
	defer buffer.affmu.Unlock()

	if buffer.fetchAllAncestors(tx) {
		// already fulfilled, insert into txpool
		// needs to resolve itself first
		logrus.Debugf("New tx fulfilled: %s", tx.GetBase().Hash.Hex())
		buffer.resolve(tx)
		buffer.txPool.AddRemoteTx(tx)
	}
}

// updateDependencyMap will update dependency relationship currently known.
// e.g., If there is already (c <- b), adding (c <- a) will result in (c <- [a,b]).
func (buffer *TxBuffer) updateDependencyMap(parentHash types.Hash, self types.Txi) {
	buffer.affmu.Lock()
	defer buffer.affmu.Unlock()
	v, err := buffer.dependencyCache.GetIFPresent(parentHash)
	if err != nil {
		// key not present, need to build a inner map
		buffer.dependencyCache.Set(parentHash, map[types.Hash]types.Txi{self.GetBase().Hash: self})
	} else {
		v.(map[types.Hash]types.Txi)[self.GetBase().Hash] = self
	}
}

// resolve is called when all ancestors of the tx is got.
func (buffer *TxBuffer) resolve(tx types.Txi) {
	vs, err := buffer.dependencyCache.GetIFPresent(tx.GetBase().Hash)
	if err != nil {
		// key not present, already resolved.
		return
	}

	buffer.dependencyCache.Remove(tx.GetBase().Hash)
	// try resolve the remainings
	for k, v := range vs.(map[types.Hash]types.Txi) {
		buffer.tryResolve(v)
	}

}

// verifyTx checks if the signatures and hashes are correct in tx
func (buffer *TxBuffer) verifyTx(tx types.Txi) error {
	if !buffer.verifier.VerifyHash(&tx) {
		return errors.New("hash is not valid")
	}
	if !buffer.verifier.VerifySignature(&tx) {
		return errors.New("signature is not valid")
	}
	return nil
}

func (buffer *TxBuffer) isKnownHash(hash types.Hash) bool {
	return buffer.txPool.Get(hash) != nil || buffer.dag.GetTx(hash) != nil
}

// tryResolve triggered when a Tx is added or resolved by other Tx
// It will check if the given hash has no more dependencies in the cache.
// If so, resolve this hash and try resolve its children
func (buffer *TxBuffer) tryResolve(tx types.Txi) bool {
	for parent := range tx.Parents(){
		_, err := buffer.dependencyCache.GetIFPresent(parent)
		if err == nil{
			// dependency presents.
			return false
		}
	}
	// no more dependencies
	buffer.resolve(tx)
	return true
}


// fetchAllAncestors examine if all ancestors are in our local cache.
// If not, go fetch it and record it in the map for future reference
// Returns true if all ancestors are local now.
func (buffer *TxBuffer) fetchAllAncestors(tx types.Txi) bool {
	allFetched := true
	if buffer.isKnownHash(tx.GetBase().Hash) {
		return true
	}
	// not in the pool, check its parents
	for _, parentHash := range tx.GetBase().ParentsHash {
		if !buffer.isKnownHash(parentHash) {
			allFetched = false
			buffer.updateDependencyMap(parentHash, tx)
			buffer.syncer.Enqueue(parentHash)
		}
	}
	return allFetched
}
