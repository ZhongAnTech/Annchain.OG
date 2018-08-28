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

	hub := og.NewHub(&og.HubConfig{
		OutgoingBufferSize: viper.GetInt("hub.outgoing_buffer_size"),
		IncomingBufferSize: viper.GetInt("hub.incoming_buffer_size"),
	})

	syncer := og.NewSyncer(&og.SyncerConfig{
		BatchTimeoutMilliSecond: 1000,
		AcquireTxQueueSize:      1000,
		MaxBatchSize:            100,
	}, hub)

	n.Components = append(n.Components, new(og.Og))
	n.Components = append(n.Components, hub)
	n.Components = append(n.Components, syncer)

	//var txPool core.TxPool
	n.Components = append(n.Components, og.NewManager(og.ManagerConfig{AcquireTxQueueSize: 10, BatchAcquireSize: 10}, hub, syncer, nil))
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
