package core

import (
	"github.com/annchain/OG/arefactor/og"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_event"
	"github.com/annchain/OG/common/io"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"path"
)

// SoloNode is the basic entrypoint for all modules to start.
type Node struct {
	components              []Component
	transportIdentityHolder *transport.DefaultTransportIdentityHolder
	cpTransport             *transport.PhysicalCommunicator
	cpBouncer               *og.Bouncer
}

// InitDefault only set necessary data structures.
// to Init a node with components, use Setup
func (n *Node) InitDefault() {
	n.components = []Component{}
}

func (n *Node) Setup() {
	// load private info
	// check if file exists
	n.transportIdentityHolder = &transport.DefaultTransportIdentityHolder{
		KeyFile: io.FixPrefixPath(viper.GetString("rootdir"), path.Join(PrivateDir, "network.key")),
	}

	cpTransport := getTransport(n.transportIdentityHolder)
	cpPerformanceMonitor := getPerformanceMonitor()

	// bouncer

	cpBouncer := &og.Bouncer{
		Id:    viper.GetInt("id"),
		Peers: []string{},
	}

	n.components = append(n.components, cpTransport)
	n.components = append(n.components, cpPerformanceMonitor)
	n.components = append(n.components, cpBouncer)

	// event registration
	cpBouncer.RegisterSubscriberNewOutgoingMessageEvent(cpTransport)
	cpTransport.RegisterSubscriberNewIncomingMessageEventSubscriber(cpBouncer)

	n.cpTransport = cpTransport
	n.cpBouncer = cpBouncer
}

func getTransport(identityHolder *transport.DefaultTransportIdentityHolder) *transport.PhysicalCommunicator {

	identity, err := identityHolder.ProvidePrivateKey(viper.GetBool("genkey"))
	if err != nil {
		logrus.WithError(err).Fatal("failed to init transport")
	}

	p2p := &transport.PhysicalCommunicator{
		Port:       viper.GetInt("p2p.port"),
		PrivateKey: identity.PrivateKey,
		ProtocolId: viper.GetString("p2p.network_id"),
	}
	p2p.InitDefault()
	// load known peers
	return p2p
}

func (n *Node) Start() {
	for _, component := range n.components {
		logrus.Infof("Starting %s", component.Name())
		component.Start()
		logrus.Infof("Started: %s", component.Name())

	}
	logrus.Info("Node Started")
	go n.AfterStart()
}
func (n *Node) AfterStart() {
	knownPeers, err := transport_event.LoadKnownPeers(
		io.FixPrefixPath(viper.GetString("rootdir"), path.Join(ConfigDir, "peers.lst")))
	if err != nil {
		logrus.WithError(err).Fatal("you need provide at least one known peer to connect to the peer network. Place them in config/peers.lst")
	}

	for _, peer := range knownPeers {
		peerId := n.cpTransport.SuggestConnection(peer)
		n.cpBouncer.Peers = append(n.cpBouncer.Peers, peerId)
	}
}
func (n *Node) Stop() {
	for i := len(n.components) - 1; i >= 0; i-- {
		comp := n.components[i]
		logrus.Infof("Stopping %s", comp.Name())
		comp.Stop()
		logrus.Infof("Stopped: %s", comp.Name())
	}
	logrus.Info("Node Stopped")
}
