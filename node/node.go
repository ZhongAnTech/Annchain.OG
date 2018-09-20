package node

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/rpc"
	"github.com/sirupsen/logrus"

	"github.com/annchain/OG/common/crypto"

	miner2 "github.com/annchain/OG/og/miner"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	"github.com/spf13/viper"
	"github.com/annchain/OG/wserver"
	"fmt"
)

// Node is the basic entrypoint for all modules to start.
type Node struct {
	Components []Component
}

func NewNode() *Node {
	n := new(Node)
	// Order matters.
	// Start myself first and then provide service and do p2p
	pm := &PerformanceMonitor{}

	var rpcServer *rpc.RpcServer


	maxPerr := viper.GetInt("p2p.max_peers")
	if maxPerr == 0 {
		maxPerr = defaultMaxPeers
	}
	if viper.GetBool("rpc.enabled") {
		rpcServer = rpc.NewRpcServer(viper.GetString("rpc.port"))
		n.Components = append(n.Components, rpcServer)
	}

	hub := og.NewHub(&og.HubConfig{
		OutgoingBufferSize:            viper.GetInt("hub.outgoing_buffer_size"),
		IncomingBufferSize:            viper.GetInt("hub.incoming_buffer_size"),
		MessageCacheExpirationSeconds: viper.GetInt("hub.message_cache_expiration_seconds"),
		MessageCacheMaxSize:           viper.GetInt("hub.message_cache_max_size"),
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

	hub.Dag = org.Dag

	// Setup crypto algorithm
	var signer crypto.Signer
	switch viper.GetString("crypto.algorithm") {
	case "ed25519":
		signer = &crypto.SignerEd25519{}
	case "secp256k1":
		signer = &crypto.SignerSecp256k1{}
	default:
		panic("Unknown crypto algorithm: " + viper.GetString("crypto.algorithm"))
	}

	verifier := og.NewVerifier(signer,
		types.HexToHash(viper.GetString("max_tx_hash")),
		types.HexToHash(viper.GetString("max_mined_hash")),
	)

	txBuffer := og.NewTxBuffer(og.TxBufferConfig{
		Syncer:                           syncer,
		Verifier:                         verifier,
		Dag:                              org.Dag,
		TxPool:                           org.Txpool,
		DependencyCacheExpirationSeconds: 10 * 60,
		DependencyCacheMaxSize:           5000,
		NewTxQueueSize:                   10000,
	})
	txBuffer.Hub = hub
	n.Components = append(n.Components, txBuffer)

	m := &og.Manager{
		TxPool:   org.Txpool,
		TxBuffer: txBuffer,
		Verifier: verifier,
		Syncer:   syncer,
		Hub:      hub,
		Dag:      org.Dag,
		Config:   &og.ManagerConfig{AcquireTxQueueSize: 10, BatchAcquireSize: 10},
	}
	// Setup Hub
	SetupCallbacks(m, hub)

	miner := &miner2.PoWMiner{}

	txCreator := &og.TxCreator{
		Signer:             signer,
		Miner:              miner,
		TipGenerator:       org.Txpool,
		MaxConnectingTries: 100,
		MaxTxHash:          types.HexToHash(viper.GetString("max_tx_hash")),
		MaxMinedHash:       types.HexToHash(viper.GetString("max_mined_hash")),
		DebugNodeId:        viper.GetInt("debug.node_id"),
	}

	var privateKey crypto.PrivateKey
	if viper.IsSet("account.private_key") {
		privateKey, err = crypto.PrivateKeyFromString(viper.GetString("account.private_key"))
		logrus.Info("Loaded private key from configuration")
		if err != nil {
			panic(err)
		}
	} else {
		_, privateKey, err = signer.RandomKeyPair()
		logrus.Warnf("Generated public/private key pair")
		if err != nil {
			panic(err)
		}
	}

	autoSequencer := &ClientAutoSequencer{
		TxCreator:             txCreator,
		TxBuffer:              m.TxBuffer,
		PrivateKey:            privateKey,
		BlockTimeMilliSeconds: viper.GetInt("auto_sequencer.interval_ms"),
		Dag:                   org.Dag,
	}
	autoSequencer.Init()
	if viper.GetBool("auto_sequencer.enabled") {
		n.Components = append(n.Components, autoSequencer)
	}

	autoTx := &ClientAutoTx{
		TxCreator:              txCreator,
		TxBuffer:               m.TxBuffer,
		PrivateKey:             privateKey,
		TxIntervalMilliSeconds: viper.GetInt("auto_tx.interval_ms"),
		Dag:                    org.Dag,
		InstanceCount:          viper.GetInt("auto_tx.count"),
	}
	autoTx.Init()
	if viper.GetBool("auto_tx.enabled") {
		n.Components = append(n.Components, autoTx)
	}

	// DataLoader
	dataLoader := &og.DataLoader{
		Dag:    org.Dag,
		TxPool: org.Txpool,
	}
	n.Components = append(n.Components, dataLoader)

	n.Components = append(n.Components, m)
	org.Manager = m

	var p2pServer *p2p.Server
	if viper.GetBool("p2p.enabled") {
		// TODO: Merge private key loading. All keys should be stored at just one place.
		privKey := getNodePrivKey()
		p2pServer = NewP2PServer(privKey)
		p2pServer.Protocols = append(p2pServer.Protocols, hub.SubProtocols...)

		n.Components = append(n.Components, p2pServer)
	}

	if rpcServer != nil {
		rpcServer.C.P2pServer = p2pServer
		rpcServer.C.Og = org
		rpcServer.C.AutoSequencer = autoSequencer
		rpcServer.C.AutoTx = autoTx
	}
	if viper.GetBool("websocket.enabled") {
		wsServer := wserver.NewServer(fmt.Sprintf(":%d", viper.GetInt("websocket.port")))
		n.Components = append(n.Components, wsServer)
		org.Txpool.OnNewTxReceived = append(org.Txpool.OnNewTxReceived, wsServer.NewTxReceivedChan)
		pm.Register(wsServer)
	}

	pm.Register(org.Txpool)
	pm.Register(syncer)
	pm.Register(txBuffer)
	pm.Register(hub)
	n.Components = append(n.Components, pm)

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
