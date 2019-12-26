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

func TestSync(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})

	total := 2
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

		bus := &eventbus.DefaultEventBus{}
		bus.InitDefault()

		dag := &dummyDag{}
		dag.InitDefault()

		ogCore := &ogcore.OgCore{
			OgCoreConfig: ogcore.OgCoreConfig{
				MaxTxCountInResponse: 100,
			},
			EventBus:         bus,
			LedgerTxProvider: dag,
		}

		partner := &ogcore.OgPartner{
			Config:         ogcore.OgProcessorConfig{},
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
		bus.Build()
	}

	// send sync request
	logrus.Debug("Sending sync request on height 0")
	processors[0].SendMessageHeightSyncRequest(communication.OgPeer{Id: 1})
	time.Sleep(time.Second * 5)
}
