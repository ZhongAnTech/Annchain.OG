package node

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/rpc"
	"github.com/spf13/viper"
	"github.com/sirupsen/logrus"
)

// Node is the basic entrypoint for all modules to start.
type Node struct {
	Components []Component
}

func NewNode() *Node {
	n := new(Node)
	// Order matters.
	// Start myself first and then provide service and do p2p
	if viper.GetBool("p2p.enabled") {
		n.Components = append(n.Components, p2p.NewP2PServer(viper.GetString("p2p.port")))
	}
	if viper.GetBool("rpc.enabled") {
		n.Components = append(n.Components, rpc.NewRpcServer(viper.GetString("rpc.port")))
	}
	n.Components = append(n.Components, p2p.NewHub(&p2p.HubConfig{
		OutgoingBufferSize: viper.GetInt("hub.outgoing_buffer_size"),
		IncomingBufferSize: viper.GetInt("hub.incoming_buffer_size"),
	}))
	n.Components = append(n.Components, new(og.Og))
	return n
}

func (n *Node) Start() {
	for _, component := range n.Components {
		logrus.Infof("Starting %s", component.Name())
		component.Start()
		logrus.Infof("Started: %s", component.Name())
	}
	logrus.Info("Node Started")
}
func (n *Node) Stop() {
	for i := len(n.Components) - 1; i >= 0; i -- {
		comp := n.Components[i]
		logrus.Infof("Stopping %s", comp.Name())
		comp.Stop()
		logrus.Infof("Stopped: %s", comp.Name())
	}
	logrus.Info("Node Stopped")
}
