package main

import (
	"fmt"
	"github.com/annchain/OG/debug/dep"
	"github.com/annchain/OG/engine"
	"github.com/annchain/OG/message"
	"github.com/annchain/OG/mylog"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/plugin/og"
	"github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

func something() {
	mylog.LogInit(logrus.InfoLevel)
	len := 2
	plugins := make([]*og.OgPlugin, len)
	chans := make([]chan *message.GeneralMessageEvent, len)
	communicators := make([]*dep.LocalGeneralPeerCommunicator, len)

	engines := make([]*engine.Engine, len)

	for i := 0; i < len; i++ {
		chans[i] = make(chan *message.GeneralMessageEvent)
	}

	for i := 0; i < len; i++ {
		communicators[i] = dep.NewLocalGeneralPeerCommunicator(i, chans[i], chans)
	}

	for i := 0; i < len; i++ {
		plugins[i] = og.NewOgPlugin()
		plugins[i].SetOutgoing(communicators[i])
	}

	// init general processor
	for i := 0; i < len; i++ {
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

func main() {
	// we need a webserver to get the pprof webserver
	go func() {
		logrus.Info(http.ListenAndServe("localhost:6060", nil))
	}()
	fmt.Println("hello world")
	var wg sync.WaitGroup
	wg.Add(1)
	something()
	wg.Wait()
}
