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
package syncer

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"sync"
	"testing"
	"time"
)

func newTestIncrementalSyncer() *IncrementalSyncer {

	isKnownHash := func(h common.Hash) bool {
		return false
	}
	newTxEnable := func() bool {
		return true
	}
	hashOrder := func() common.Hashes {
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
		tx := tx_types.RandomTx()
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
	pubKey, _ := signer.RandomKeyPair()
	types.Signer = signer
	for i := 0; i < 60000; i++ {
		tx := tx_types.RandomTx()
		tx.PublicKey = pubKey.Bytes
		msg := &p2p_message.MessageNewTx{tx.RawTx()}
		wg.Add(1)
		go func() {
			syncer.HandleNewTx(msg, "123")
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println("used", time.Now().Sub(start).String(), "len", syncer.bufferedIncomingTxCache.Len())
}

func TestSyncBuffer_AddTxs(t *testing.T) {
	signer := crypto.NewSigner(crypto.CryptoTypeEd25519)
	syncer := newTestIncrementalSyncer()
	pubKey, _ := signer.RandomKeyPair()
	types.Signer = signer
	var wg sync.WaitGroup
	for i := 0; i < 60000; i++ {
		tx := tx_types.RandomTx()
		tx.PublicKey = pubKey.Bytes
		msg := &p2p_message.MessageNewTx{tx.RawTx()}
		wg.Add(1)
		go func() {
			syncer.HandleNewTx(msg, "123")
			wg.Done()
		}()
	}
}
