package node

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/rpc"
	"github.com/sirupsen/logrus"

	"github.com/annchain/OG/common/crypto"

	"github.com/annchain/OG/p2p"
	"github.com/spf13/viper"
)

// Node is the basic entrypoint for all modules to start.
type Node struct {
	Components []Component
}

func NewNode() *Node {
	n := new(Node)
	// Order matters.
	// Start myself first and then provide service and do p2p

	var rpcServer *rpc.RpcServer
	var p2pServer *p2p.Server
	 maxPerr := viper.GetInt("p2p.max_peers")
	 if maxPerr ==0 {
	 	maxPerr = defaultMaxPeers
	 }
	if viper.GetBool("rpc.enabled") {
		rpcServer = rpc.NewRpcServer(viper.GetString("rpc.port"))
		n.Components = append(n.Components, rpcServer)
	}
	if viper.GetBool("p2p.enabled") {
		privKey := getNodePrivKey()
		p2pServer = NewP2PServer(privKey)
		n.Components = append(n.Components, p2pServer)
	}

	hub := og.NewHub(&og.HubConfig{
		OutgoingBufferSize: viper.GetInt("hub.outgoing_buffer_size"),
		IncomingBufferSize: viper.GetInt("hub.incoming_buffer_size"),
	}, maxPerr)

	syncer := og.NewSyncer(&og.SyncerConfig{
		BatchTimeoutMilliSecond:              1000,
		AcquireTxQueueSize:                   1000,
		MaxBatchSize:                         100,
		AcquireTxDedupCacheMaxSize:           10000,
		AcquireTxDedupCacheExpirationSeconds: 60,
	}, hub)

	org, err := og.NewOg()
	if err != nil {
		logrus.WithError(err).Fatalf("Error occurred while initializing OG")
		panic("Error occurred while initializing OG")
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

	m.TxBuffer = og.NewTxBuffer(og.TxBufferConfig{
		Syncer:   syncer,
		Verifier: m.Verifier,
		Dag:      org.Dag,
		TxPool:   org.Txpool,
		DependencyCacheExpirationSeconds: 10 * 60,
		DependencyCacheMaxSize:           5000,
		NewTxQueueSize:                   1000,
	})
	n.Components = append(n.Components, m.TxBuffer)
	n.Components = append(n.Components, m)
	org.Manager = m
	p2pServer.Protocols = append(p2pServer.Protocols, hub.SubProtocols...)
	if rpcServer != nil {
		rpcServer.C.P2pServer = p2pServer
		rpcServer.C.Og = org
	}
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
	for i := len(n.Components) - 1; i >= 0; i-- {
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
