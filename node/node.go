package node

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/rpc"
	"github.com/sirupsen/logrus"

	"github.com/annchain/OG/common/crypto"

	"github.com/spf13/viper"

	"github.com/annchain/OG/core"
)

// Node is the basic entrypoint for all modules to start.
type Node struct {
	Components []Component
}

func NewNode() *Node {
	n := new(Node)
	// Order matters.
	// Start myself first and then provide service and do p2p
	/*
	if viper.GetBool("p2p.enabled") {
		n.Components = append(n.Components, p2p.NewP2PServer(viper.GetString("p2p.port")))
	}
	*/
	if viper.GetBool("rpc.enabled") {
		n.Components = append(n.Components, rpc.NewRpcServer(viper.GetString("rpc.port")))
	}
	if viper.GetBool("p2p.enabled") {
		privKey := getNodePrivKey()
		n.Components = append(n.Components, NewP2PServer(privKey))
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

	org, err := og.NewOg()
	if err != nil {
		logrus.WithError(err).Fatalf("Error occurred while initializing Node")
		panic("Error occurred while initializing Node")
	}

	n.Components = append(n.Components, org)
	n.Components = append(n.Components, hub)
	n.Components = append(n.Components, syncer)

	m := og.NewManager(&og.ManagerConfig{AcquireTxQueueSize: 10, BatchAcquireSize: 10})

	// Setup Hub
	m.Hub = hub
	SetupCallbacks(m, hub)

	m.Syncer = syncer
	m.TxPool = nil

	// Setup crypto algorithm
	switch viper.GetString("crypto.algorithm") {
	case "ed25519":
		m.Verifier = og.NewVerifier(&crypto.SignerEd25519{})
	default:
		panic("Unknown crypto algorithm: " + viper.GetString("crypto.algorithm"))
	}

	m.TxBuffer = core.NewTxBuffer(core.TxBufferConfig{
		Syncer:                           syncer,
		Verifier:                         m.Verifier,
		Dag:                              org.Dag,
		TxPool:                           org.Txpool,
		DependencyCacheExpirationSeconds: 10 * 60,
		DependencyCacheMaxSize:           5000,
	})

	n.Components = append(n.Components, m)
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

// SetupCallbacks Regist callbacks to handle different messages
func SetupCallbacks(m *og.Manager, hub *og.Hub) {
	hub.CallbackRegistry[og.MessageTypePing] = m.HandlePing
	hub.CallbackRegistry[og.MessageTypePong] = m.HandlePong
	hub.CallbackRegistry[og.MessageTypeFetchByHash] = m.HandleFetchByHash
	hub.CallbackRegistry[og.MessageTypeFetchByHashResponse] = m.HandleFetchByHashResponse
	hub.CallbackRegistry[og.MessageTypeNewTx] = m.HandleNewTx
	hub.CallbackRegistry[og.MessageTypeNewSequence] = m.HandleNewSequence
}
