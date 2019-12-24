package engine

import (
	"github.com/annchain/OG/communication"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/message"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	"sync"
)

type EngineConfig struct {
}

type Engine struct {
	Config        EngineConfig
	plugins       []communication.GeneralMessageHandlerPlugin
	messageRouter map[message.GeneralMessageType]communication.GeneralMessageEventHandler
	eventBus      *eventbus.DefaultEventBus
	PeerOutgoing  communication.GeneralPeerCommunicatorOutgoing
	PeerIncoming  communication.GeneralPeerCommunicatorIncoming

	quit        chan bool
	quitWg      sync.WaitGroup
	performance map[string]interface{}
}

func (a *Engine) Name() string {
	return "Engine"
}

func (a *Engine) GetBenchmarks() map[string]interface{} {
	return a.performance
}

func (a *Engine) InitDefault() {
	a.messageRouter = make(map[message.GeneralMessageType]communication.GeneralMessageEventHandler)
	a.eventBus = &eventbus.DefaultEventBus{}
	a.eventBus.InitDefault()
	a.quitWg = sync.WaitGroup{}
	a.quit = make(chan bool)
	a.performance = make(map[string]interface{})
	a.performance["mps"] = uint(0)

}

func (a *Engine) RegisterPlugin(plugin communication.GeneralMessageHandlerPlugin) {
	a.plugins = append(a.plugins, plugin)
	handler := plugin.GetMessageEventHandler()
	// message handlers
	for _, msgType := range plugin.SupportedMessageTypes() {
		a.messageRouter[msgType] = handler
	}
	// event handlers
	for _, handler := range plugin.SupportedEventHandlers() {
		a.eventBus.ListenTo(handler)
	}
}

func (ap *Engine) Start() {
	// build eventbus
	ap.eventBus.Build()
	// start the plugins
	for _, plugin := range ap.plugins {
		plugin.Start()
	}
	// start the receiver
	go ap.loop()
	log.Info("Engine Started")
}

func (ap *Engine) Stop() {
	ap.quit <- true
	ap.quitWg.Wait()
	logrus.Debug("Engine stopped")
}

func (ap *Engine) loop() {
	for {
		select {
		case <-ap.quit:
			ap.quitWg.Done()
			logrus.Debug("AnnsensusPartner quit")
			return
		case msgEvent := <-ap.PeerIncoming.GetPipeOut():
			ap.HandleGeneralMessage(msgEvent)
			ap.performance["mps"] = ap.performance["mps"].(uint) + 1
		}
	}
}

func (ap *Engine) HandleGeneralMessage(msgEvent *message.GeneralMessageEvent) {
	generalMessage := msgEvent.Message
	handler, ok := ap.messageRouter[generalMessage.GetType()]
	if !ok {
		logrus.WithField("type", generalMessage.GetType()).Warn("message type not recognized by engine. not registered?")
		return
	}
	handler.Handle(msgEvent)
}
