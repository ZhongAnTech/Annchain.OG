package node

import (
	"github.com/annchain/OG/consensus/dpos"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/rpc"
	"github.com/sirupsen/logrus"

	"github.com/annchain/OG/common/crypto"

	"fmt"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og/downloader"
	miner2 "github.com/annchain/OG/og/miner"
	"github.com/annchain/OG/og/syncer"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/wserver"
	"github.com/spf13/viper"
	"strconv"
)

// Node is the basic entrypoint for all modules to start.
type Node struct {
	Components []Component
}

func InitLoggers(logger *logrus.Logger, logdir string) {
	downloader.InitLoggers(logger, logdir)
	p2p.InitLoggers(logger, logdir)
	og.InitLoggers(logger, logdir)
}

func NewNode() *Node {
	n := new(Node)
	// Order matters.
	// Start myself first and then provide service and do p2p
	pm := &PerformanceMonitor{}

	var rpcServer *rpc.RpcServer

	maxPeers := viper.GetInt("p2p.max_peers")
	if maxPeers == 0 {
		maxPeers = defaultMaxPeers
	}
	if viper.GetBool("rpc.enabled") {
		rpcServer = rpc.NewRpcServer(viper.GetString("rpc.port"))
		n.Components = append(n.Components, rpcServer)
	}
	bootNode := viper.GetBool("p2p.bootstrap_node")

	networkId := viper.GetInt64("p2p.network_id")
	if networkId == 0 {
		networkId = defaultNetworkId
	}

	org, err := og.NewOg(
		og.OGConfig{
			BootstrapNode: bootNode,
			NetworkId:     uint64(networkId),
		},
	)
	org.NewLatestSequencerCh = org.TxPool.OnNewLatestSequencer

	if err != nil {
		logrus.WithError(err).Fatalf("Error occurred while initializing OG")
		panic("Error occurred while initializing OG")
	}

	hub := og.NewHub(&og.HubConfig{
		OutgoingBufferSize:            viper.GetInt("hub.outgoing_buffer_size"),
		IncomingBufferSize:            viper.GetInt("hub.incoming_buffer_size"),
		MessageCacheExpirationSeconds: viper.GetInt("hub.message_cache_expiration_seconds"),
		MessageCacheMaxSize:           viper.GetInt("hub.message_cache_max_size"),
		MaxPeers:                      maxPeers,
	}, org.Dag, org.TxPool)

	hub.StatusDataProvider = org

	n.Components = append(n.Components, org)
	n.Components = append(n.Components, hub)

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

	graphVerifier := &og.GraphVerifier{
		Dag:    org.Dag,
		TxPool: org.TxPool,
		//Buffer: txBuffer,
	}

	txFormatVerifier := &og.TxFormatVerifier{
		Signer:       signer,
		CryptoType:   signer.GetCryptoType(),
		MaxTxHash:    types.HexToHash(viper.GetString("max_tx_hash")),
		MaxMinedHash: types.HexToHash(viper.GetString("max_mined_hash")),
	}

	verifiers := []og.Verifier{graphVerifier, txFormatVerifier}

	txBuffer := og.NewTxBuffer(og.TxBufferConfig{
		Verifiers:                        verifiers,
		Dag:                              org.Dag,
		TxPool:                           org.TxPool,
		DependencyCacheExpirationSeconds: 10 * 60,
		DependencyCacheMaxSize:           5000,
		NewTxQueueSize:                   10000,
	})
	syncBuffer := syncer.NewSyncBuffer(syncer.SyncBufferConfig{
		TxPool:         org.TxPool,
		Dag:            org.Dag,
		FormatVerifier: txFormatVerifier,
		GraphVerifier:  graphVerifier,
	})
	n.Components = append(n.Components, syncBuffer)

	org.TxBuffer = txBuffer
	n.Components = append(n.Components, txBuffer)

	syncManager := syncer.NewSyncManager(syncer.SyncManagerConfig{
		Mode:           downloader.FullSync,
		ForceSyncCycle: uint(viper.GetInt("hub.sync_cycle_ms")),
		BootstrapNode:  bootNode,
	}, hub, org)

	downloaderInstance := downloader.New(downloader.FullSync, org.Dag, hub.RemovePeer, syncBuffer.AddTxs)

	syncManager.CatchupSyncer = &syncer.CatchupSyncer{
		PeerProvider:           hub,
		NodeStatusDataProvider: org,
		Hub:                    hub,
		Downloader:             downloaderInstance,
		SyncMode:               downloader.FullSync,
	}
	syncManager.CatchupSyncer.Init()
	hub.Downloader = downloaderInstance

	messageHandler := &og.IncomingMessageHandler{
		Hub: hub,
		Og:  org,
	}

	m := &og.MessageRouter{
		FetchByHashResponseHandler: messageHandler,
		PongHandler:                messageHandler,
		PingHandler:                messageHandler,
		BodiesRequestHandler:       messageHandler,
		BodiesResponseHandler:      messageHandler,
		HeaderRequestHandler:       messageHandler,
		SequencerHeaderHandler:     messageHandler,
		TxsRequestHandler:          messageHandler,
		TxsResponseHandler:         messageHandler,
		HeaderResponseHandler:      messageHandler,
		FetchByHashRequestHandler:  messageHandler,
		Hub:                        hub,
	}

	syncManager.IncrementalSyncer = syncer.NewIncrementalSyncer(
		&syncer.SyncerConfig{
			BatchTimeoutMilliSecond:              100,
			AcquireTxQueueSize:                   1000,
			MaxBatchSize:                         100,
			AcquireTxDedupCacheMaxSize:           10000,
			AcquireTxDedupCacheExpirationSeconds: 60,
		}, m)

	m.NewSequencerHandler = syncManager.IncrementalSyncer
	m.NewTxsHandler = syncManager.IncrementalSyncer
	m.NewTxHandler = syncManager.IncrementalSyncer

	//syncManager.OnUpToDate = append(syncManager.OnUpToDate, syncer.UpToDateEventListener)
	//org.OnNodeSyncStatusChanged = append(org.OnNodeSyncStatusChanged, syncer.UpToDateEventListener)

	syncManager.IncrementalSyncer.OnNewTxiReceived = append(syncManager.IncrementalSyncer.OnNewTxiReceived, txBuffer.ReceivedNewTxChan)

	txBuffer.Syncer = syncManager.IncrementalSyncer
	announcer := syncer.NewAnnouncer(m)
	txBuffer.Announcer = announcer

	n.Components = append(n.Components, syncManager)

	messageHandler32 := &og.IncomingMessageHandlerOG32{
		Hub: hub,
		Og:  org,
	}

	mr32 := &og.MessageRouterOG32{
		GetNodeDataMsgHandler: messageHandler32,
		GetReceiptsMsgHandler: messageHandler32,
		NodeDataMsgHandler:    messageHandler32,
	}
	// Setup Hub
	SetupCallbacks(m, hub)
	SetupCallbacksOG32(mr32, hub)

	miner := &miner2.PoWMiner{}

	txCreator := &og.TxCreator{
		Signer:             signer,
		Miner:              miner,
		TipGenerator:       org.TxPool,
		MaxConnectingTries: 100,
		MaxTxHash:          types.HexToHash(viper.GetString("max_tx_hash")),
		MaxMinedHash:       types.HexToHash(viper.GetString("max_mined_hash")),
		DebugNodeId:        viper.GetInt("debug.node_id"),
		GraphVerifier:      graphVerifier,
	}

	// TODO: move to (embeded) client. It is not part of OG
	//var privateKey crypto.PrivateKey
	//if viper.IsSet("account.private_key") {
	//	privateKey, err = crypto.PrivateKeyFromString(viper.GetString("account.private_key"))
	//	logrus.Info("Loaded private key from configuration")
	//	if err != nil {
	//		panic(err)
	//	}
	//} else {
	//	_, privateKey, err = signer.RandomKeyPair()
	//	logrus.Warnf("Generated public/private key pair")
	//	if err != nil {
	//		panic(err)
	//	}
	//}

	delegate := &Delegate{
		TxPool:    org.TxPool,
		TxBuffer:  txBuffer,
		Dag:       org.Dag,
		TxCreator: txCreator,
	}

	autoClientManager := &AutoClientManager{
		SampleAccounts: core.GetSampleAccounts(),
	}
	autoClientManager.Init(
		StringArrayToIntArray(viper.GetStringSlice("auto_client.tx.account_ids")),
		delegate,
	)
	n.Components = append(n.Components, autoClientManager)
	syncManager.OnUpToDate = append(syncManager.OnUpToDate, autoClientManager.UpToDateEventListener)
	hub.OnNewPeerConnected = append(hub.OnNewPeerConnected, syncManager.CatchupSyncer.NewPeerConnectedEventListener)

	if org.BootstrapNode {
		go func() {
			autoClientManager.UpToDateEventListener <- true
		}()
	}

	switch viper.GetString("consensus") {
	case "dpos":
		//todo
		consensus := dpos.NewDpos(org.Dag, &types.Address{})
		n.Components = append(n.Components, consensus)
	case "pos":
		//todo
	case "pow":
		//todo
	default:
		panic("Unknown consensus algorithm: " + viper.GetString("consensus"))
	}

	// DataLoader
	dataLoader := &og.DataLoader{
		Dag:    org.Dag,
		TxPool: org.TxPool,
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
		// just for debugging, ignoring index OOR
		rpcServer.C.NewRequestChan = autoClientManager.Clients[0].ManualChan
		rpcServer.C.SyncerManager = syncManager
	}
	if viper.GetBool("websocket.enabled") {
		wsServer := wserver.NewServer(fmt.Sprintf(":%d", viper.GetInt("websocket.port")))
		n.Components = append(n.Components, wsServer)
		org.TxPool.OnNewTxReceived = append(org.TxPool.OnNewTxReceived, wsServer.NewTxReceivedChan)
		org.TxPool.OnBatchConfirmed = append(org.TxPool.OnBatchConfirmed, wsServer.BatchConfirmedChan)
		pm.Register(wsServer)
	}

	pm.Register(org.TxPool)
	pm.Register(syncManager)
	pm.Register(txBuffer)
	pm.Register(hub)
	n.Components = append(n.Components, pm)

	return n
}

func StringArrayToIntArray(arr []string) []int {
	var a = make([]int, len(arr))
	var err error
	for i := 0; i < len(arr); i++ {
		a[i], err = strconv.Atoi(arr[i])
		if err != nil {
			logrus.WithError(err).Fatal("bad config on string array")
		}
	}
	return a
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
func SetupCallbacks(m *og.MessageRouter, hub *og.Hub) {
	hub.CallbackRegistry[og.MessageTypePing] = m.RoutePing
	hub.CallbackRegistry[og.MessageTypePong] = m.RoutePong
	hub.CallbackRegistry[og.MessageTypeFetchByHashRequest] = m.RouteFetchByHashRequest
	hub.CallbackRegistry[og.MessageTypeFetchByHashResponse] = m.RouteFetchByHashResponse
	hub.CallbackRegistry[og.MessageTypeNewTx] = m.RouteNewTx
	hub.CallbackRegistry[og.MessageTypeNewTxs] = m.RouteNewTxs
	hub.CallbackRegistry[og.MessageTypeNewSequencer] = m.RouteNewSequencer

	hub.CallbackRegistry[og.MessageTypeSequencerHeader] = m.RouteSequencerHeader
	hub.CallbackRegistry[og.MessageTypeBodiesRequest] = m.RouteBodiesRequest
	hub.CallbackRegistry[og.MessageTypeBodiesResponse] = m.RouteBodiesResponse
	hub.CallbackRegistry[og.MessageTypeTxsRequest] = m.RouteTxsRequest
	hub.CallbackRegistry[og.MessageTypeTxsResponse] = m.RouteTxsResponse
	hub.CallbackRegistry[og.MessageTypeHeaderRequest] = m.RouteHeaderRequest
	hub.CallbackRegistry[og.MessageTypeHeaderResponse] = m.RouteHeaderResponse
}

func SetupCallbacksOG32(m *og.MessageRouterOG32, hub *og.Hub) {
	hub.CallbackRegistryOG32[og.GetNodeDataMsg] = m.RouteGetNodeDataMsg
	hub.CallbackRegistryOG32[og.NodeDataMsg] = m.RouteNodeDataMsg
	hub.CallbackRegistryOG32[og.GetReceiptsMsg] = m.RouteGetReceiptsMsg
}
