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
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/interfaces"
	"github.com/annchain/OG/ogcore/pool"
	"github.com/annchain/OG/protocol"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
)

func setupTxBuffer() (*pool.TxBuffer, interfaces.Syncer) {
	logrus.StandardLogger().Formatter = &logrus.TextFormatter{
		ForceColors: true,
	}

	bus := &eventbus.DefaultEventBus{
		NodeLogger: debuglog.NodeLogger{
			Logger: logrus.StandardLogger(),
		},
	}
	bus.InitDefault()

	ver := new(dummyVerifier)
	txPool := new(dummyTxPool)
	txPool.InitDefault()
	dag := new(dummyDag)
	dag.InitDefault()
	syncer := &dummySyncer{EventBus: bus}
	syncer.InitDefault()

	buffer := &pool.TxBuffer{
		NodeLogger: debuglog.NodeLogger{
			Logger: logrus.StandardLogger(),
		},
		Verifiers:              []protocol.Verifier{ver},
		PoolHashLocator:        txPool,
		LedgerHashLocator:      dag,
		LocalGraphInfoProvider: txPool,
		EventBus:               bus,
	}
	buffer.InitDefault(pool.TxBufferConfig{
		DependencyCacheMaxSize:           10,
		DependencyCacheExpirationSeconds: 30,
		NewTxQueueSize:                   10,
		KnownCacheMaxSize:                10,
		KnownCacheExpirationSeconds:      30,
		AddedToPoolQueueSize:             10,
		TestNoVerify:                     false,
	})
	//buffer.Syncer.(*dummySyncer).dmap = make(map[common.Hash]types.Txi)
	//buffer.Syncer.(*dummySyncer).buffer = buffer
	//buffer.Syncer.(*dummySyncer).acquireTxDedupCache = gcache.New(100).Simple().
	//	Expiration(time.Second * 10).Build()

	// event registration
	bus.ListenTo(eventbus.EventHandlerRegisterInfo{
		Type:    events.TxReceivedEventType,
		Name:    "TxReceivedEventType",
		Handler: buffer,
	})
	bus.ListenTo(eventbus.EventHandlerRegisterInfo{
		Type:    events.SequencerReceivedEventType,
		Name:    "SequencerReceivedEventType",
		Handler: buffer,
	})
	bus.ListenTo(eventbus.EventHandlerRegisterInfo{
		Type:    events.NeedSyncTxEventType,
		Name:    "NeedSyncTxEventType",
		Handler: syncer,
	})
	bus.Build()
	return buffer, syncer
}

func doTest(buffer *pool.TxBuffer) {
	buffer.Start()
	time.Sleep(time.Second * 3)
	buffer.DumpUnsolved()
}

func TestBufferAllOK(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer, syncerInst := setupTxBuffer()
	m := syncerInst.(*dummySyncer)
	m.Know(sampleTx("0x00", nil, 0))

	m.Know(sampleTx("0x01", []string{"0x00"}, 1))
	m.Know(sampleTx("0x02", []string{"0x00"}, 2))
	m.Know(sampleTx("0x03", []string{"0x00"}, 3))
	m.Know(sampleTx("0x04", []string{"0x02"}, 4))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02", "0x03"}, 5))
	m.Know(sampleTx("0x06", []string{"0x02"}, 6))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}, 7))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}, 8))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}, 9))
	tx := sampleTx("0x0A", []string{"0x09"}, 10)
	buffer.Start()

	buffer.EventBus.Route(&events.NeedSyncTxEvent{
		Hash:            tx.ParentsHash[0],
		SpecifiedSource: nil,
	})

	doTest(buffer)
	time.Sleep(time.Second * 4)
	buffer.Stop()
	assert.Equal(t, buffer.PendingLen(), 0)
}

func TestBufferMissing3(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer, syncer := setupTxBuffer()
	m := syncer.(*dummySyncer)
	m.Know(sampleTx("0x00", nil, 0))

	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(sampleTx("0x02", []string{"0x00"}, 1))
	//m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(sampleTx("0x04", []string{"0x02"}, 2))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02", "0x03"}, 3))
	m.Know(sampleTx("0x06", []string{"0x02"}, 4))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}, 5))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}, 6))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}, 7))
	tx := sampleTx("0x0A", []string{"0x09"}, 8)

	buffer.Start()
	buffer.EventBus.Route(&events.TxReceivedEvent{
		Tx: tx,
	})

	doTest(buffer)
	time.Sleep(time.Second * 3)
	buffer.Stop()
	// missing 1,3,5,7,8,9,a
	assert.Equal(t, buffer.PendingLen(), 7)
}
