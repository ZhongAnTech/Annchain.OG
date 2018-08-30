package core

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"sync"
	"errors"
)

type txStatus int

const (
	txStatusNone       txStatus = iota
	txStatusFetched     // all previous ancestors got
	txStatusValidated   // ancestors are valid
	txStatusConflicted  // ancestors are conflicted, or itself is conflicted
)

type TxBuffer struct {
	dag      IDag
	verifier *og.Verifier
	syncer   *og.Syncer

	dependencyMap map[types.Hash]map[types.Hash]struct{} // list of hashes that are pending on the parent.
	affmu         sync.RWMutex
}

func NewTxBuffer(d IDag, verifier *og.Verifier, syncer *og.Syncer) *TxBuffer {
	return &TxBuffer{
		dag:           d,
		verifier:      verifier,
		syncer:        syncer,
		dependencyMap: make(map[types.Hash]map[types.Hash]struct{}),
	}
}

func (pool *TxBuffer) addToDependencyMap(parent types.Hash, self types.Hash) {
	pool.affmu.Lock()
	defer pool.affmu.Unlock()

	if v, ok := pool.dependencyMap[parent]; ok {
		v[self] = struct{}{}
	} else {
		pool.dependencyMap[parent] = map[types.Hash]struct{}{self: {}}
	}
}

func (pool *TxBuffer) deleteFromDependencyMap(parent types.Hash, self types.Hash) {
	pool.affmu.Lock()
	defer pool.affmu.Unlock()
	if v, ok := pool.dependencyMap[parent]; ok {
		delete(v, self)
		if len(v) == 0 {
			delete(pool.dependencyMap, parent)
		}
	} else {
		return
	}
}

// verifyTx checks if the signatures and hashes are correct in tx
func (pool *TxBuffer) verifyTx(tx types.Txi) error {
	if !pool.verifier.VerifyHash(&tx) {
		return errors.New("hash is not valid")
	}
	if !pool.verifier.VerifySignature(&tx) {
		return errors.New("signature is not valid")
	}
	return nil
}
