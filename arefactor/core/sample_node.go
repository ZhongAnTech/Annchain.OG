package core

import (
	"github.com/annchain/OG/arefactor/bouncer"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/og"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/OG/common/io"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"path"
)

// OgNode is the basic entry point for all modules to start.
type SampleNode struct {
	components             []Component
	transportAccountHolder og.TransportAccountHolder
	cpTransport            *transport.PhysicalCommunicator
	cpBouncer              *bouncer.Bouncer
}

// InitDefault only set necessary data structures.
// to Init a node with components, use Setup
func (n *SampleNode) InitDefault() {
	n.components = []Component{}
}

func (n *SampleNode) Setup() {
	// load private info
	// check if file exists
	n.transportAccountHolder = &og.LocalTransportAccountHolder{
		PrivateGenerator:   &og.DefaultPrivateGenerator{},
		NetworkIdConverter: &og.OgNetworkIdConverter{},
		BackFilePath:       io.FixPrefixPath(viper.GetString("rootdir"), path.Join(PrivateDir, "transport.key")),
		CryptoType:         transport_interface.CryptoTypeSecp256k1,
		account:            nil,
	}

	// low level transport (libp2p)
	cpTransport := getTransport(n.transportAccountHolder)

	cpPerformanceMonitor := getPerformanceMonitor()

	// bouncer

	cpBouncer := &bouncer.Bouncer{
		Id:    viper.GetInt("id"),
		Peers: []string{},
	}
	cpBouncer.InitDefault()

	n.components = append(n.components, cpTransport)
	n.components = append(n.components, cpPerformanceMonitor)
	n.components = append(n.components, cpBouncer)

	// event registration
	// bouncer io
	cpBouncer.RegisterSubscriberNewOutgoingMessageEvent(cpTransport)
	cpTransport.AddSubscriberNewIncomingMessageEvent(cpBouncer)

	// performance monitor registration
	cpPerformanceMonitor.Register(cpBouncer)

	n.cpTransport = cpTransport
	n.cpBouncer = cpBouncer
}

func (n *SampleNode) Start() {
	for _, component := range n.components {
		logrus.Infof("Starting %s", component.Name())
		component.Start()
		logrus.Infof("Started: %s", component.Name())

	}
	logrus.Info("OgNode Started")
	n.AfterStart()
}
func (n *SampleNode) AfterStart() {
	knownPeersAddress, err := transport_interface.LoadKnownPeers(
		io.FixPrefixPath(viper.GetString("rootdir"), path.Join(ConfigDir, "peers.lst")))
	if err != nil {
		logrus.WithError(err).Fatal("you need provide at least one known address to connect to the address network. Place them in config/peers.lst")
	}

	// let bouncer knows first. pretend that the suggest is given by bouncer
	for _, address := range knownPeersAddress {
		nodeId, err := n.cpTransport.GetPeerId(address)
		utilfuncs.PanicIfError(err, "parse node address")
		n.cpBouncer.Peers = append(n.cpBouncer.Peers, nodeId)
	}

	for _, peer := range knownPeersAddress {
		n.cpTransport.SuggestConnection(peer)
	}
}
func (n *SampleNode) Stop() {
	for i := len(n.components) - 1; i >= 0; i-- {
		comp := n.components[i]
		logrus.Infof("Stopping %s", comp.Name())
		comp.Stop()
		logrus.Infof("Stopped: %s", comp.Name())
	}
	logrus.Info("OgNode Stopped")
}
