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
package pool

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/og/types/archive"
	"github.com/annchain/OG/og/verifier"
	"github.com/annchain/OG/protocol"
	"testing"
	"time"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/gcache"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
)

func setup() *TxBuffer {
	ver := new(og.dummyVerifier)
	buffer := NewTxBuffer(TxBufferConfig{
		Verifiers:                        []protocol.Verifier{ver},
		DependencyCacheMaxSize:           20,
		TxPool:                           new(og.dummyTxPool),
		Dag:                              new(og.dummyDag),
		Syncer:                           new(og.dummySyncer),
		DependencyCacheExpirationSeconds: 60,
		NewTxQueueSize:                   100,
		KnownCacheMaxSize:                10000,
		KnownCacheExpirationSeconds:      30,
	})
	buffer.Syncer.(*og.dummySyncer).dmap = make(map[common.Hash]types.Txi)
	buffer.Syncer.(*og.dummySyncer).buffer = buffer
	buffer.Syncer.(*og.dummySyncer).acquireTxDedupCache = gcache.New(100).Simple().
		Expiration(time.Second * 10).Build()
	buffer.dag.(*og.dummyDag).init()
	buffer.txPool.(*og.dummyTxPool).init()
	return buffer
}

func doTest(buffer *TxBuffer) {
	buffer.Start()

	if buffer.dependencyCache.Len(true) != 0 {
		for k, v := range buffer.dependencyCache.GetALL(true) {
			for k1 := range v.(map[common.Hash]types.Txi) {
				logrus.Warnf("not fulfilled: %s <- %s", k.(common.Hash), k1)
			}
		}
	}
}

func TestBuffer(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.Syncer.(*og.dummySyncer)
	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(og.sampleTx("0x02", []string{"0x00"}))
	m.Know(og.sampleTx("0x03", []string{"0x00"}))
	m.Know(og.sampleTx("0x04", []string{"0x02"}))
	m.Know(og.sampleTx("0x05", []string{"0x01", "0x02", "0x03"}))
	m.Know(og.sampleTx("0x06", []string{"0x02"}))
	m.Know(og.sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(og.sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(og.sampleTx("0x09", []string{"0x07", "0x08"}))
	tx := og.sampleTx("0x0A", []string{"0x09"})
	<-ffchan.NewTimeoutSenderShort(buffer.ReceivedNewTxChan, tx, "test").C
	//buffer.AddTx(sampleTx("0x09", []string{"0x04"}))

	doTest(buffer)
	time.Sleep(time.Second * 3)
	buffer.Stop()
	assert.Equal(t, buffer.dependencyCache.Len(true), 0)
}

func TestBufferMissing3(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.Syncer.(*og.dummySyncer)
	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(og.sampleTx("0x02", []string{"0x00"}))
	//m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(og.sampleTx("0x04", []string{"0x02"}))
	m.Know(og.sampleTx("0x05", []string{"0x01", "0x02", "0x03"}))
	m.Know(og.sampleTx("0x06", []string{"0x02"}))
	m.Know(og.sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(og.sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(og.sampleTx("0x09", []string{"0x07", "0x08"}))
	tx := og.sampleTx("0x0A", []string{"0x09"})
	<-ffchan.NewTimeoutSenderShort(buffer.ReceivedNewTxChan, tx, "test").C
	//buffer.AddTx(sampleTx("0x09", []string{"0x04"}))

	doTest(buffer)
	time.Sleep(time.Second * 3)
	buffer.Stop()
	// missing 5,7,8,9
	assert.Equal(t, buffer.dependencyCache.Len(true), 5)
}

func TestBufferCache(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.Syncer.(*og.dummySyncer)
	tx := og.sampleTx("0x0A", []string{"0x09"})
	<-ffchan.NewTimeoutSenderShort(buffer.ReceivedNewTxChan, tx, "test").C
	buffer.Start()
	success := false
	for i := 0; i < 8; i++ {
		time.Sleep(time.Second * 2)
		// query request cache
		_, err := m.acquireTxDedupCache.Get(common.HexToHash("0x09"))
		if err != nil {
			// not found
			logrus.Debug("not in cache")
			success = true
			break
		} else {
			logrus.Debug("in cache")
		}
	}
	buffer.Stop()
	assert.Equal(t, success, true)
}

func TestLocalHash(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})
	buffer := setup()
	tx2 := og.sampleTx("0x02", []string{"0x00"})
	tx3 := og.sampleTx("0x03", []string{"0x00"})
	buffer.txPool.AddRemoteTx(tx2, true)
	if !buffer.isLocalHash(tx2.GetTxHash()) {
		t.Fatal("is localhash")
	}
	if buffer.isLocalHash(tx3.GetTxHash()) {
		t.Fatal("is not localhash")
	}
}

func TestTxBuffer_Handle(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.TraceLevel)
	ver := &verifier.TxFormatVerifier{
		NoVerifyMaxTxHash: true,
		NoVerifyMindHash:  true,
	}
	buffer := NewTxBuffer(TxBufferConfig{
		Verifiers:                        []protocol.Verifier{ver},
		DependencyCacheMaxSize:           20,
		TxPool:                           new(og.dummyTxPool),
		DependencyCacheExpirationSeconds: 60,
		NewTxQueueSize:                   100,
		KnownCacheMaxSize:                10000,
		KnownCacheExpirationSeconds:      30,
	})
	pub, priv := crypto.Signer.RandomKeyPair()
	from := pub.Address()
	N := 20
	var txs types.Txis
	for i := 0; i < N; i++ {
		tx := archive.RandomTx()
		tx.Height = 1
		tx.Weight = tx.Weight % uint64(N)
		tx.SetSender(from)
		tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).SignatureBytes
		tx.PublicKey = pub.KeyBytes
		tx.Hash = tx.CalcTxHash()
		txs = append(txs, tx)
	}
	logrus.Debug("handle txis start", txs)
	buffer.handleTxs(txs)
	for i := 0; i < N; i++ {

	}
}
