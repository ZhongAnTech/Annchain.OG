package ogcore_test

import (
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestPingPong(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	total := 2
	// init two OG peers's In channel
	peerChans := make([]chan *communication.OgMessageEvent, total)
	peerInfos := make([]*communication.OgPeer, total)

	// build communication channels
	for i := 0; i < total; i++ {
		peerInfos[i] = &communication.OgPeer{Id: i}
		peerChans[i] = make(chan *communication.OgMessageEvent, 10)
	}

	processors := make([]*ogcore.OgPartner, total)

	// build peer communicator
	for i := 0; i < total; i++ {
		communicator := &DummyOgPeerCommunicator{
			NodeLogger: debuglog.NodeLogger{
				Logger: debuglog.SetupOrderedLog(i),
			},
			Myid:        i,
			PeerPipeIns: peerChans,
			PipeIn:      peerChans[i],
		}
		communicator.InitDefault()
		communicator.Run()

		processor := &ogcore.OgPartner{
			NodeLogger: debuglog.NodeLogger{
				Logger: debuglog.SetupOrderedLog(i),
			},
			Config:         ogcore.OgProcessorConfig{},
			PeerOutgoing:   communicator,
			PeerIncoming:   communicator,
			EventBus:       nil,
			StatusProvider: nil,
			OgCore:         nil,
		}
		processor.InitDefault()

		processors[i] = processor
		processors[i].Start()
	}

	// send ping
	logrus.Debug("Sending ping")
	processors[0].SendMessagePing(peerInfos[1])
	time.Sleep(time.Minute * 4)
}
