package core

import (
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/common/io"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"path"
)

// SoloNode is the basic entrypoint for all modules to start.
type Node struct {
	components              []Component
	transportIdentityHolder *transport.DefaultTransportIdentityHolder
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

	n.components = append(n.components, getTransport(n.transportIdentityHolder))
	n.components = append(n.components, getPerformanceMonitor())
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
	knownPeers, err := transport.LoadKnownPeers(
		io.FixPrefixPath(viper.GetString("rootdir"), path.Join(ConfigDir, "peers.lst")))
	if err != nil {
		logrus.WithError(err).Fatal("you need provide at least one known peer to connect to the peer network. Place them in config/peers.lst")
	}

	for _, peer := range knownPeers {
		p2p.SuggestConnection(peer)
	}

	return p2p
}

func (n *Node) Start() {
	for _, component := range n.components {
		logrus.Infof("Starting %s", component.Name())
		component.Start()
		logrus.Infof("Started: %s", component.Name())

	}
	logrus.Info("Node Started")
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
