package ogcore_test

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/utilfuncs"
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/ledger"
	"github.com/annchain/OG/ogcore/pool"
	syncer2 "github.com/annchain/OG/ogcore/syncer"
	"github.com/annchain/OG/protocol"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func setupSyncBuffer(total int) []*ogcore.OgPartner {
	// init two OG peers's In channel
	peerChans := make([]chan *communication.OgMessageEvent, total)
	peerInfos := make([]communication.OgPeer, total)

	// build communication channels
	for i := 0; i < total; i++ {
		peerInfos[i] = communication.OgPeer{Id: i}
		peerChans[i] = make(chan *communication.OgMessageEvent, 10)
	}

	processors := make([]*ogcore.OgPartner, total)

	// build peer communicator
	for i := 0; i < total; i++ {

		logger := debuglog.SetupOrderedLog(i)

		communicator := &DummyOgPeerCommunicator{
			NodeLogger: debuglog.NodeLogger{
				Logger: logger,
			},
			Myid:        i,
			PeerPipeIns: peerChans,
			PipeIn:      peerChans[i],
		}
		communicator.InitDefault()
		communicator.Run()

		bus := &eventbus.DefaultEventBus{
			NodeLogger: debuglog.NodeLogger{
				Logger: logger,
			},
			ID: i,
		}
		bus.InitDefault()

		ver := new(dummyVerifier)
		dag := &dummyDag{}
		dag.InitDefault()

		txPool := &pool.TxPool{
			NodeLogger: debuglog.NodeLogger{
				Logger: logger,
			},
			EventBus: bus,
			Config:   pool.DefaultTxPoolConfig(),
			Dag:      dag,
		}
		txPool.InitDefault()

		genesis := &types.Sequencer{
			Hash:         common.HexToHashNoError("0x00"),
			ParentsHash:  nil,
			Height:       1,
			MineNonce:    0,
			AccountNonce: 0,
			Issuer:       common.HexToAddressNoError("0x00"),
			Signature:    nil,
			PublicKey:    nil,
			StateRoot:    common.Hash{},
			Weight:       1,
		}

		err := txPool.PushBatch(&ledger.ConfirmBatch{
			Seq: genesis,
			Txs: nil,
		})
		utilfuncs.PanicIfError(err, "writing genesis")

		txBuffer := &pool.TxBuffer{
			NodeLogger: debuglog.NodeLogger{
				Logger: logger,
			},
			Verifiers:              []protocol.Verifier{ver},
			PoolHashLocator:        txPool,
			LedgerHashLocator:      dag,
			LocalGraphInfoProvider: txPool,
			EventBus:               bus,
		}
		txBuffer.InitDefault(pool.TxBufferConfig{
			DependencyCacheMaxSize:           10,
			DependencyCacheExpirationSeconds: 30,
			NewTxQueueSize:                   10,
			KnownCacheMaxSize:                10,
			KnownCacheExpirationSeconds:      30,
			AddedToPoolQueueSize:             10,
			TestNoVerify:                     false,
		})
		txBuffer.Start()

		knownTxiCache := &pool.KnownTxiCache{
			AdditionalHashLocators: []pool.LedgerHashLocator{
				dag,
			},
			Config: pool.KnownTxiCacheConfig{
				MaxSize:           100,
				ExpirationSeconds: 100,
			},
		}
		knownTxiCache.InitDefault()

		ogCore := &ogcore.OgCore{
			NodeLogger: debuglog.NodeLogger{
				Logger: logger,
			},
			EventBus:         bus,
			LedgerTxProvider: dag,
			TxBuffer:         txBuffer,
			TxPool:           txPool,
			KnownTxiCache:    knownTxiCache,
		}
		syncer := &syncer2.Syncer2{
			NodeLogger: debuglog.NodeLogger{
				Logger: logger,
			},
			Config: &syncer2.SyncerConfig{
				AcquireTxDedupCacheMaxSize:           10,
				AcquireTxDedupCacheExpirationSeconds: 10,
			},
			PeerOutgoing: communicator,
		}
		syncer.InitDefault()
		syncer.Start()

		partner := &ogcore.OgPartner{
			NodeLogger: debuglog.NodeLogger{
				Logger: logger,
			},
			Config:         ogcore.OgProcessorConfig{},
			PeerOutgoing:   communicator,
			PeerIncoming:   communicator,
			EventBus:       bus,
			StatusProvider: nil,
			OgCore:         ogCore,
			Syncer:         syncer,
		}
		partner.InitDefault()
		processors[i] = partner
		processors[i].Start()

		// setup bus
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.HeightSyncRequestReceivedEventType,
			Name:    "HeightSyncRequestReceivedEventType",
			Handler: partner,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.BatchSyncRequestReceivedEventType,
			Name:    "BatchSyncRequestReceivedEventType",
			Handler: partner,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.TxsFetchedForResponseEventType,
			Name:    "TxsFetchedForResponseEventType",
			Handler: partner,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.NewTxLocallyGeneratedEventType,
			Name:    "NewTxLocallyGeneratedEventType",
			Handler: partner,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.NewTxLocallyGeneratedEventType,
			Name:    "NewTxLocallyGeneratedEventType",
			Handler: txBuffer,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.NewTxLocallyGeneratedEventType,
			Name:    "NewTxLocallyGeneratedEventType",
			Handler: ogCore,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.NewSequencerLocallyGeneratedEventType,
			Name:    "NewSequencerLocallyGeneratedEventType",
			Handler: ogCore,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.TxReceivedEventType,
			Name:    "TxReceivedEventType",
			Handler: ogCore,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.SequencerReceivedEventType,
			Name:    "SequencerReceivedEventType",
			Handler: ogCore,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.TxReceivedEventType,
			Name:    "TxReceivedEventType",
			Handler: txBuffer,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.NeedSyncEventType,
			Name:    "BufferLackTxSyncerHelps",
			Handler: syncer,
		})
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.NewTxiDependencyFulfilledEventType,
			Name:    "BufferGotAllDependencies",
			Handler: txPool,
		})

		bus.Build()
	}
	return processors
}

func TestBroadcastAndBuffer(t *testing.T) {
	setupLog()
	total := 5
	processors := setupSyncBuffer(total)

	// one is generating new txs constantly
	logrus.Debug("generating txs")

	// event should be generated outside the processor
	processors[0].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x01", []string{"0x00"}, 1),
		RequireBroadcast: true,
	})
	processors[1].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x02", []string{"0x01"}, 2),
		RequireBroadcast: true,
	})
	processors[2].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x03", []string{"0x02"}, 3),
		RequireBroadcast: true,
	})
	processors[3].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x04", []string{"0x03"}, 4),
		RequireBroadcast: true,
	})
	processors[4].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x05", []string{"0x04"}, 5),
		RequireBroadcast: true,
	})
	time.Sleep(time.Second * 5)
	for _, processor := range processors {
		processor.OgCore.TxBuffer.DumpUnsolved()
		assert.Equal(t, processor.OgCore.TxBuffer.PendingLen(), 0)
		assert.Equal(t, int(processor.OgCore.TxPool.GetTxNum()), total)
	}

}

func TestSyncAndBuffer(t *testing.T) {
	setupLog()
	total := 2
	processors := setupSyncBuffer(total)

	// one is generating new txs constantly
	logrus.Debug("generating txs")

	// event should be generated outside the processor
	processors[0].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x01", []string{"0x00"}, 1),
		RequireBroadcast: false,
	})
	// sleep 2 seconds for node 0 handling
	time.Sleep(time.Second * 2)
	processors[1].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x02", []string{"0x01"}, 2),
		RequireBroadcast: false,
	})
	//processors[2].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
	//	Tx: sampleTx("0x03", []string{"0x02"}, 1),
	//	RequireBroadcast: false,
	//})
	//processors[3].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
	//	Tx: sampleTx("0x04", []string{"0x03"}, 1),
	//	RequireBroadcast: false,
	//})
	//processors[4].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
	//	Tx: sampleTx("0x05", []string{"0x04"}, 1),
	//	RequireBroadcast: false,
	//})
	time.Sleep(time.Second * 5)
	for i, processor := range processors {
		processor.OgCore.TxBuffer.DumpUnsolved()
		assert.Equal(t, processor.OgCore.TxBuffer.PendingLen(), 0)
		assert.Equal(t, int(processor.OgCore.TxPool.GetTxNum()), i+1)
	}
}

func TestBufferSingleNode(t *testing.T) {
	setupLog()
	total := 1
	processors := setupSyncBuffer(total)

	// one is generating new txs constantly
	logrus.Debug("generating txs")

	// event should be generated outside the processor
	processors[0].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x01", []string{"0x00"}, 1),
		RequireBroadcast: false,
	})
	// nonce error
	processors[0].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx:               sampleTx("0x02", []string{"0x01"}, 3),
		RequireBroadcast: false,
	})
	time.Sleep(time.Second)

	for i, processor := range processors {
		processor.OgCore.TxBuffer.DumpUnsolved()
		assert.Equal(t, processor.OgCore.TxBuffer.PendingLen(), 0)
		assert.Equal(t, int(processor.OgCore.TxPool.GetTxNum()), i+1)
	}
}

