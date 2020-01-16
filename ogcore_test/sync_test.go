package ogcore_test

import (
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/events"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func setupSync(total int) []*ogcore.OgPartner {
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
		communicator := NewDummyOgPeerCommunicator(i, peerChans[i], peerChans)
		communicator.Run()

		bus := &eventbus.DefaultEventBus{ID: i}
		bus.InitDefault()

		dag := &dummyDag{}
		dag.InitDefault()

		ogCore := &ogcore.OgCore{
			EventBus:         bus,
			LedgerTxProvider: dag,
		}

		partner := &ogcore.OgPartner{
			Config: ogcore.OgProcessorConfig{
				MaxTxCountInResponse: 100,
			},
			PeerOutgoing:   communicator,
			PeerIncoming:   communicator,
			EventBus:       bus,
			StatusProvider: nil,
			OgCore:         ogCore,
		}

		processors[i] = partner
		processors[i].Start()

		// setup bus
		bus.ListenTo(eventbus.EventHandlerRegisterInfo{
			Type:    events.HeightSyncRequestReceivedEventType,
			Name:    "HeightSyncRequestReceivedEventType",
			Handler: ogCore,
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
		//bus.ListenTo(eventbus.EventHandlerRegisterInfo{
		//	Type:    events.TxReceivedEventType,
		//	Name:    "TxReceivedEventType",
		//	Handler: ogCore,
		//})

		bus.Build()
	}
	return processors
}

func TestBroadcast(t *testing.T) {
	setupLog()
	total := 2
	processors := setupSync(total)

	// send sync request
	logrus.Debug("Sending sync request on height 0")
	processors[0].SendMessageHeightSyncRequest(&communication.OgPeer{Id: 1})
	time.Sleep(time.Second * 5)
}

func TestIncremental(t *testing.T) {
	setupLog()
	total := 2
	processors := setupSync(total)

	// one is generating new txs constantly
	logrus.Debug("generating txs")

	// event should be generated outside the processor
	processors[0].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
		Tx: sampleTx("0x01", []string{"0x00"}, 1),
	})
	//processors[1].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
	//	Tx: sampleTx("0x02", []string{"0x01"}),
	//})
	//processors[2].EventBus.Route(&events.NewTxLocallyGeneratedEvent{
	//	Tx: sampleTx("0x03", []string{"0x02"}),
	//})
	time.Sleep(time.Second * 5)
}
