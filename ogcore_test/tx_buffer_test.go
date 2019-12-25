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
package ogcore_test

import (
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/og/verifier"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/protocol"
	"testing"
	"time"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/ffchan"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
)

func setup() *ogcore.TxBuffer {
	ver := new(dummyVerifier)
	pool := new(dummyTxPool)
	pool.InitDefault()
	dag := new(dummyDag)
	dag.InitDefault()
	syncer := new(dummySyncer)

	bus := &eventbus.DefaultEventBus{}
	bus.InitDefault()

	buffer := &ogcore.TxBuffer{
		Verifiers:              []protocol.Verifier{ver},
		LocalHashLocator:       pool,
		LedgerHashLocator:      dag,
		LocalGraphInfoProvider: pool,
		EventBus:               bus,
	}
	buffer.InitDefault()
	//buffer.Syncer.(*dummySyncer).dmap = make(map[common.Hash]types.Txi)
	//buffer.Syncer.(*dummySyncer).buffer = buffer
	//buffer.Syncer.(*dummySyncer).acquireTxDedupCache = gcache.New(100).Simple().
	//	Expiration(time.Second * 10).Build()

	// event registration
	bus.ListenTo(eventbus.EventHandlerRegisterInfo{
		Type:    events.TxsReceivedEventType,
		Handler: buffer,
	})
	bus.ListenTo(eventbus.EventHandlerRegisterInfo{
		Type:    events.SequencerReceivedEventType,
		Handler: buffer,
	})
	bus.ListenTo(eventbus.EventHandlerRegisterInfo{
		Type:    events.NeedSyncEventType,
		Handler: syncer,
	})
	return buffer
}

func doTest(buffer *ogcore.TxBuffer) {
	buffer.Start()
	buffer.DumpUnsolved()
}

func TestBuffer(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.Syncer.(*dummySyncer)
	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(sampleTx("0x02", []string{"0x00"}))
	m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(sampleTx("0x04", []string{"0x02"}))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02", "0x03"}))
	m.Know(sampleTx("0x06", []string{"0x02"}))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}))
	tx := sampleTx("0x0A", []string{"0x09"})
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
	m := buffer.Syncer.(*dummySyncer)
	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(sampleTx("0x02", []string{"0x00"}))
	//m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(sampleTx("0x04", []string{"0x02"}))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02", "0x03"}))
	m.Know(sampleTx("0x06", []string{"0x02"}))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}))
	tx := sampleTx("0x0A", []string{"0x09"})
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
	m := buffer.Syncer.(*dummySyncer)
	tx := sampleTx("0x0A", []string{"0x09"})
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
	tx2 := sampleTx("0x02", []string{"0x00"})
	tx3 := sampleTx("0x03", []string{"0x00"})
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
		TxPool:                           new(dummyTxPool),
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
		tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes
		tx.PublicKey = pub.Bytes
		tx.Hash = tx.CalcTxHash()
		txs = append(txs, tx)
	}
	logrus.Debug("handle txis start", txs)
	buffer.handleTxs(txs)
	for i := 0; i < N; i++ {

	}
}
