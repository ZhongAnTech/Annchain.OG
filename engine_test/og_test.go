package engine_test

import (
	"github.com/annchain/OG/engine"
	"github.com/annchain/OG/message"
	"github.com/annchain/OG/mylog"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/plugin/og"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

// TestPingPongBenchmark will try its best to send ping pong between two to benchmark
func TestPingPongBenchmark(t *testing.T) {
	mylog.LogInit(logrus.InfoLevel)
	nodes := 2
	plugins := make([]*og.OgPlugin, nodes)
	chans := make([]chan *message.GeneralMessageEvent, nodes)
	communicators := make([]*LocalGeneralPeerCommunicator, nodes)

	engines := make([]*engine.Engine, nodes)

	for i := 0; i < nodes; i++ {
		chans[i] = make(chan *message.GeneralMessageEvent)
	}

	for i := 0; i < nodes; i++ {
		communicators[i] = NewLocalGeneralPeerCommunicator(i, chans[i], chans)
	}

	for i := 0; i < nodes; i++ {
		plugins[i] = og.NewOgPlugin()
		plugins[i].SetOutgoing(communicators[i])
	}

	// init general processor
	for i := 0; i < nodes; i++ {
		eng := engine.Engine{
			Config:       engine.EngineConfig{},
			PeerOutgoing: communicators[i],
			PeerIncoming: communicators[i],
		}
		eng.InitDefault()
		eng.RegisterPlugin(plugins[i])
		engines[i] = &eng
		eng.Start()
	}

	logrus.Info("Started")


	plugins[0].OgPartner.SendMessagePing(communication.OgPeer{Id: 1})

	var lastValue uint = 0
	for i := 0; i < 60; i++ {
		v := engines[0].GetBenchmarks()["mps"].(uint)
		if lastValue == 0 {
			lastValue = v
		} else {
			logrus.WithField("mps", v-lastValue).Info("performance")
		}
		lastValue = v
		time.Sleep(time.Second)
	}
}


