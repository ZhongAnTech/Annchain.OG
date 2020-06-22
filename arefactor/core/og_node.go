package core

import (
	"github.com/annchain/OG/arefactor/og"
	"github.com/annchain/OG/arefactor/rpc"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/OG/common/io"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"path"
)

// OgNode is the basic entry point for all modules to start.
type OgNode struct {
	components             []Component
	transportAccountHolder og.TransportAccountHolder
}

// InitDefault only set necessary data structures.
// to Init a node with components, use Setup
func (n *OgNode) InitDefault() {
	n.components = []Component{}
}

func (n *OgNode) Setup() {
	// load private info
	// check if file exists
	n.transportAccountHolder = &og.LocalTransportAccountHolder{
		PrivateGenerator:   &og.DefaultPrivateGenerator{},
		NetworkIdConverter: &og.OgNetworkIdConverter{},
		BackFilePath:       io.FixPrefixPath(viper.GetString("rootdir"), path.Join(PrivateDir, "transport.key")),
		CryptoType:         transport_interface.CryptoTypeSecp256k1,
		Account:            nil,
	}

	// low level transport (libp2p)
	cpTransport := getTransport(n.transportAccountHolder)
	cpPerformanceMonitor := getPerformanceMonitor()

	// peer relationship management
	cpCommunityManager := &og.DefaultCommunityManager{
		PhysicalCommunicator:  cpTransport,
		KnownPeerListFilePath: io.FixPrefixPath(viper.GetString("rootdir"), path.Join(ConfigDir, "peers.lst")),
	}
	cpCommunityManager.InitDefault()
	cpCommunityManager.StaticSetup()

	// ledger implementation
	ledger := &og.IntArrayLedger{}
	ledger.InitDefault()
	ledger.StaticSetup()

	cpController := &rpc.RpcController{
		Ledger:                    ledger,
		CpDefaultCommunityManager: cpCommunityManager,
	}

	// rpc
	cpRpc := &rpc.RpcServer{
		Controller: cpController,
		Port:       viper.GetInt("rpc.port"),
	}

	cpRpc.InitDefault()

	cpSyncer := &og.BlockByBlockSyncer{
		Ledger: ledger,
	}
	cpSyncer.InitDefault()

	// OG engine
	cpOgEngine := &og.OgEngine{
		Ledger:           ledger,
		CommunityManager: cpCommunityManager,
	}
	cpCommunityManager.NodeInfoProvider = cpOgEngine

	cpOgEngine.InitDefault()
	cpOgEngine.StaticSetup()

	n.components = append(n.components, cpTransport)
	n.components = append(n.components, cpPerformanceMonitor)
	n.components = append(n.components, cpCommunityManager)
	n.components = append(n.components, cpOgEngine)
	n.components = append(n.components, cpRpc)
	n.components = append(n.components, cpSyncer)

	// event registration

	// message sender
	cpOgEngine.AddSubscriberNewOutgoingMessageEvent(cpTransport)
	cpCommunityManager.AddSubscriberNewOutgoingMessageEvent(cpTransport)
	cpSyncer.AddSubscriberNewOutgoingMessageEvent(cpTransport)

	// message receivers
	cpTransport.AddSubscriberNewIncomingMessageEvent(cpOgEngine)
	cpTransport.AddSubscriberNewIncomingMessageEvent(cpCommunityManager)
	cpTransport.AddSubscriberNewIncomingMessageEvent(cpSyncer)

	// peer connected
	cpTransport.AddSubscriberPeerConnectedEvent(cpCommunityManager)

	// peer joined and left to the network cluster (protocol verified)
	cpCommunityManager.AddSubscriberPeerJoinedEvent(cpOgEngine)
	cpCommunityManager.AddSubscriberPeerLeftEvent(cpSyncer)

	// peer height provided
	cpOgEngine.AddSubscriberNewHeightDetectedEvent(cpSyncer)

	// performance monitor registration
	cpPerformanceMonitor.Register(cpOgEngine)
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