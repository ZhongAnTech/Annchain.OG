package syncer

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"sync"
	"testing"
	"time"
)

func newTestIncrementalSyncer() *IncrementalSyncer {

	isKnownHash := func(h types.Hash) bool {
		return false
	}
	newTxEnable := func() bool {
		return true
	}
	hashOrder := func() []types.Hash {
		return nil
	}
	heighter := func() uint64 {
		return 0
	}
	log = logrus.StandardLogger()
	log.SetLevel(logrus.InfoLevel)
	syncer := NewIncrementalSyncer(
		&SyncerConfig{
			BatchTimeoutMilliSecond:                  100,
			AcquireTxQueueSize:                       1000,
			MaxBatchSize:                             100,
			AcquireTxDedupCacheMaxSize:               10000,
			AcquireTxDedupCacheExpirationSeconds:     60,
			BufferedIncomingTxCacheExpirationSeconds: 600,
			BufferedIncomingTxCacheMaxSize:           80000,
			FiredTxCacheExpirationSeconds:            600,
			FiredTxCacheMaxSize:                      10000,
		}, nil, hashOrder, isKnownHash,
		heighter, newTxEnable)
	syncer.Enabled = true
	for i := 0; i < 5000; i++ {
		tx := types.RandomTx()
		err := syncer.bufferedIncomingTxCache.EnQueue(tx)
		if err != nil {
			panic(err)
		}
	}
	go syncer.txNotifyLoop()
	return syncer

}

func stopIncSyncer(syncer *IncrementalSyncer) {
	syncer.quitNotifyEvent <- true
}

func TestIncrementalSyncer_AddTxs(t *testing.T) {
	syncer := newTestIncrementalSyncer()
	defer stopIncSyncer(syncer)
	var wg sync.WaitGroup
	start := time.Now()
	signer := crypto.NewSigner(crypto.CryptoTypeEd25519)
	pubKey, _, _ := signer.RandomKeyPair()
	types.Signer = signer
	for i := 0; i < 60000; i++ {
		tx := types.RandomTx()
		tx.PublicKey = pubKey.Bytes
		msg := &types.MessageNewTx{tx.RawTx()}
		wg.Add(1)
		go func() {
			syncer.HandleNewTx(msg)
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println("used", time.Now().Sub(start).String(), "len", syncer.bufferedIncomingTxCache.Len())
}


func TestSyncBuffer_AddTxs(t *testing.T) {
	signer := crypto.NewSigner(crypto.CryptoTypeEd25519)
	pubKey, _, _ := signer.RandomKeyPair()
	types.Signer = signer
	for i := 0; i < 60000; i++ {
		tx := types.RandomTx()
		tx.PublicKey = pubKey.Bytes
		msg := &types.MessageNewTx{tx.RawTx()}
		wg.Add(1)
		go func() {
			syncer.HandleNewTx(msg)
			wg.Done()
		}()
	}
}
