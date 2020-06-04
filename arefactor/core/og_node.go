package core

import (
	"github.com/annchain/OG/arefactor/og"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/common/io"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"path"
)

// OgNode is the basic entry point for all modules to start.
type OgNode struct {
	components              []Component
	transportIdentityHolder *transport.DefaultTransportIdentityHolder
	cpTransport             *transport.PhysicalCommunicator
	cpOgEngine              *og.OgEngine
}

// InitDefault only set necessary data structures.
// to Init a node with components, use Setup
func (n *OgNode) InitDefault() {
	n.components = []Component{}
}

func (n *OgNode) Setup() {
	// load private info
	// check if file exists
	n.transportIdentityHolder = &transport.DefaultTransportIdentityHolder{
		KeyFile: io.FixPrefixPath(viper.GetString("rootdir"), path.Join(PrivateDir, "network.key")),
	}

	cpTransport := getTransport(n.transportIdentityHolder)
	cpPerformanceMonitor := getPerformanceMonitor()

	// og engine

	ledger := &og.DefaultLedger{}
	ledger.StaticSetup()

	cpCommunityManager := &og.DefaultCommunityManager{
		PhysicalCommunicator:  cpTransport,
		KnownPeerListFilePath: io.FixPrefixPath(viper.GetString("rootdir"), path.Join(ConfigDir, "peers.lst")),
	}
	cpCommunityManager.StaticSetup()

	cpOgEngine := &og.OgEngine{
		Ledger:           ledger,
		CommunityManager: cpCommunityManager,
	}

	cpOgEngine.InitDefault()
	cpOgEngine.StaticSetup()

	n.components = append(n.components, cpTransport)
	n.components = append(n.components, cpPerformanceMonitor)
	n.components = append(n.components, cpCommunityManager)
	n.components = append(n.components, cpOgEngine)

	// event registration
	// bouncer io
	cpOgEngine.RegisterSubscriberNewOutgoingMessageEvent(cpTransport)
	cpCommunityManager.RegisterSubscriberNewOutgoingMessageEvent(cpTransport)
	cpTransport.RegisterSubscriberNewIncomingMessageEventSubscriber(cpOgEngine)

	// performance monitor registration
	cpPerformanceMonitor.Register(cpOgEngine)

	n.cpTransport = cpTransport
	n.cpOgEngine = cpOgEngine
}

func (n *OgNode) Start() {
	for _, component := range n.components {
		logrus.Infof("Starting %s", component.Name())
		component.Start()
		logrus.Infof("Started: %s", component.Name())

	}
	logrus.Info("OgNode Started")
}

func (n *OgNode) Stop() {
	for i := len(n.components) - 1; i >= 0; i-- {
		comp := n.components[i]
		logrus.Infof("Stopping %s", comp.Name())
		comp.Stop()
		logrus.Infof("Stopped: %s", comp.Name())
	}
	logrus.Info("OgNode Stopped")
}
