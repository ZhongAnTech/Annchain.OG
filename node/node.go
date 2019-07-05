// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package node

import (
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/encryption"
	"github.com/annchain/OG/common/io"
	"github.com/annchain/OG/p2p/ioperformance"
	"github.com/annchain/OG/status"
	"time"

	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/og/syncer"
	"github.com/annchain/OG/rpc"
	"github.com/annchain/OG/wserver"
	"github.com/sirupsen/logrus"

	"github.com/annchain/OG/common/crypto"

	"strconv"

	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/og/fetcher"

	"github.com/annchain/OG/core"

	miner2 "github.com/annchain/OG/og/miner"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/performance"
	"github.com/annchain/OG/types"
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
	pm := &performance.PerformanceMonitor{}
	var rpcServer *rpc.RpcServer
	var cryptoType crypto.CryptoType

	switch viper.GetString("crypto.algorithm") {
	case "ed25519":
		cryptoType = crypto.CryptoTypeEd25519
	case "secp256k1":
		cryptoType = crypto.CryptoTypeSecp256k1
	default:
		panic("Unknown crypto algorithm: " + viper.GetString("crypto.algorithm"))
	}
	//set default signer
	crypto.Signer = crypto.NewSigner(cryptoType)

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
			NetworkId:   uint64(networkId),
			GenesisPath: io.FixPrefixPath(viper.GetString("datadir"), viper.GetString("dag.genesis_path")),
		},
	)
	if err != nil {
		logrus.WithError(err).Warning("Error occurred while initializing OG")
		panic(fmt.Sprintf("Error occurred while initializing OG %v", err))
	}
	feedBack := og.FeedBackMode
	if viper.GetBool("hub.disable_feedback") == true {
		feedBack = og.NormalMode
	}
	hub := og.NewHub(&og.HubConfig{
		OutgoingBufferSize:            viper.GetInt("hub.outgoing_buffer_size"),
		IncomingBufferSize:            viper.GetInt("hub.incoming_buffer_size"),
		MessageCacheExpirationSeconds: viper.GetInt("hub.message_cache_expiration_seconds"),
		MessageCacheMaxSize:           viper.GetInt("hub.message_cache_max_size"),
		MaxPeers:                      maxPeers,
		BroadCastMode:                 feedBack,
		DisableEncryptGossip:          viper.GetBool("hub.disable_encrypt_gossip"),
	})

	hub.StatusDataProvider = org

	n.Components = append(n.Components, org)
	n.Components = append(n.Components, hub)

	// Setup crypto algorithm
	types.Signer = crypto.Signer
	graphVerifier := &og.GraphVerifier{
		Dag:    org.Dag,
		TxPool: org.TxPool,
		//Buffer: txBuffer,
	}

	txFormatVerifier := &og.TxFormatVerifier{
		MaxTxHash:    types.HexToHash(viper.GetString("max_tx_hash")),
		MaxMinedHash: types.HexToHash(viper.GetString("max_mined_hash")),
	}
	if txFormatVerifier.MaxMinedHash == types.HexToHash("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") {
		txFormatVerifier.NoVerifyMindHash = true
		logrus.Info("no verify mined hash")
	}
	if txFormatVerifier.MaxTxHash == types.HexToHash("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") {
		txFormatVerifier.NoVerifyMaxTxHash = true
		logrus.Info("no verify max tx hash")
	}
	consensusVerifier := &og.ConsensusVerifier{}

	verifiers := []og.Verifier{graphVerifier, txFormatVerifier, consensusVerifier}

	txBuffer := og.NewTxBuffer(og.TxBufferConfig{
		Verifiers:                        verifiers,
		Dag:                              org.Dag,
		TxPool:                           org.TxPool,
		DependencyCacheExpirationSeconds: 10 * 60,
		DependencyCacheMaxSize:           20000,
		NewTxQueueSize:                   1,
		KnownCacheMaxSize:                30000,
		KnownCacheExpirationSeconds:      10 * 60,
		AddedToPoolQueueSize:             10000,
	})
	hub.IsReceivedHash = txBuffer.IsReceivedHash
	syncBuffer := syncer.NewSyncBuffer(syncer.SyncBufferConfig{
		TxPool:    org.TxPool,
		Dag:       org.Dag,
		Verifiers: verifiers,
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
	heighter := func() uint64 {
		return org.Dag.LatestSequencer().Height
	}
	hub.Fetcher = fetcher.New(org.Dag.GetSequencerByHash, heighter, syncBuffer.AddTxs, hub.RemovePeer)
	syncManager.CatchupSyncer = &syncer.CatchupSyncer{
		PeerProvider:           hub,
		NodeStatusDataProvider: org,
		Hub:                    hub,
		Downloader:             downloaderInstance,
		SyncMode:               downloader.FullSync,
		BootStrapNode:          bootNode,
	}
	syncManager.CatchupSyncer.Init()
	hub.Downloader = downloaderInstance

	messageHandler := og.NewIncomingMessageHandler(org, hub, 10000, time.Millisecond*40)

	m := &og.MessageRouter{
		PongHandler:               messageHandler,
		PingHandler:               messageHandler,
		BodiesRequestHandler:      messageHandler,
		BodiesResponseHandler:     messageHandler,
		HeaderRequestHandler:      messageHandler,
		SequencerHeaderHandler:    messageHandler,
		TxsRequestHandler:         messageHandler,
		TxsResponseHandler:        messageHandler,
		HeaderResponseHandler:     messageHandler,
		FetchByHashRequestHandler: messageHandler,
		GetMsgHandler:             messageHandler,
		ControlMsgHandler:         messageHandler,
		Hub:                       hub,
	}
	n.Components = append(n.Components, messageHandler)
	syncManager.IncrementalSyncer = syncer.NewIncrementalSyncer(
		&syncer.SyncerConfig{
			BatchTimeoutMilliSecond:                  20,
			AcquireTxQueueSize:                       1000,
			MaxBatchSize:                             5, //smaller
			AcquireTxDedupCacheMaxSize:               10000,
			AcquireTxDedupCacheExpirationSeconds:     60,
			BufferedIncomingTxCacheExpirationSeconds: 600,
			BufferedIncomingTxCacheMaxSize:           40000,
			FiredTxCacheExpirationSeconds:            600,
			FiredTxCacheMaxSize:                      10000,
			NewTxsChannelSize:                        15,
		}, m, org.TxPool.GetHashOrder, org.TxBuffer.IsKnownHash,
		heighter, syncManager.CatchupSyncer.CacheNewTxEnabled)
	org.TxPool.OnNewLatestSequencer = append(org.TxPool.OnNewLatestSequencer, org.NewLatestSequencerCh,
		syncManager.IncrementalSyncer.NewLatestSequencerCh)
	m.NewSequencerHandler = syncManager.IncrementalSyncer
	m.NewTxsHandler = syncManager.IncrementalSyncer
	m.NewTxHandler = syncManager.IncrementalSyncer
	m.FetchByHashResponseHandler = syncManager.IncrementalSyncer
	m.CampaignHandler = syncManager.IncrementalSyncer
	m.TermChangeHandler = syncManager.IncrementalSyncer
	m.ArchiveHandler = syncManager.IncrementalSyncer
	m.ActionTxHandler = syncManager.IncrementalSyncer
	messageHandler.TxEnable = syncManager.IncrementalSyncer.TxEnable
	syncManager.IncrementalSyncer.RemoveContrlMsgFromCache = messageHandler.RemoveControlMsgFromCache
	//syncManager.OnUpToDate = append(syncManager.OnUpToDate, syncer.UpToDateEventListener)
	//org.OnNodeSyncStatusChanged = append(org.OnNodeSyncStatusChanged, syncer.UpToDateEventListener)

	syncManager.IncrementalSyncer.OnNewTxiReceived = append(syncManager.IncrementalSyncer.OnNewTxiReceived, txBuffer.ReceivedNewTxsChan)

	txBuffer.Syncer = syncManager.IncrementalSyncer
	announcer := syncer.NewAnnouncer(m)
	txBuffer.Announcer = announcer
	n.Components = append(n.Components, syncManager)

	messageHandler32 := &og.IncomingMessageHandlerOG02{
		Hub: hub,
		Og:  org,
	}

	mr32 := &og.MessageRouterOG02{
		GetNodeDataMsgHandler: messageHandler32,
		GetReceiptsMsgHandler: messageHandler32,
		NodeDataMsgHandler:    messageHandler32,
	}

	// Setup Hub
	SetupCallbacks(m, hub)
	SetupCallbacksOG32(mr32, hub)

	miner := &miner2.PoWMiner{}
	txCreator := &og.TxCreator{
		Miner:              miner,
		TipGenerator:       og.NewFIFOTIpGenerator(org.TxPool, 6),
		MaxConnectingTries: 100,
		MaxTxHash:          txFormatVerifier.MaxTxHash,
		MaxMinedHash:       txFormatVerifier.MaxMinedHash,
		NoVerifyMindHash:   txFormatVerifier.NoVerifyMindHash,
		NoVerifyMaxTxHash:  txFormatVerifier.NoVerifyMaxTxHash,
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
		TxPool: org.TxPool,
		//TxBuffer:  txBuffer,
		Dag:       org.Dag,
		TxCreator: txCreator,
	}

	delegate.OnNewTxiGenerated = append(delegate.OnNewTxiGenerated, txBuffer.SelfGeneratedNewTxChan)
	genesisAccounts := parserGenesisAccounts(viper.GetString("annsensus.genesis_pk"))

	mode := viper.GetString("mode")
	if mode == "archive" {
		status.ArchiveMode = true
	}
	disableConsensus := viper.GetBool("annsensus.disable")
	//TODO temperary , delete this after demo
	//if archiveMode {
	//	disableConsensus = true
	//}
	campaign := viper.GetBool("annsensus.campaign")
	disableTermChange := viper.GetBool("annsensus.disable_term_change")
	partnerNum := viper.GetInt("annsensus.partner_number")
	//threshold := viper.GetInt("annsensus.threshold")
	sequencerTime := viper.GetInt("annsensus.sequencerTime")
	if sequencerTime == 0 {
		sequencerTime = 3000
	}
	consensFilePath := io.FixPrefixPath(viper.GetString("datadir"), viper.GetString("annsensus.consensus_path"))
	if consensFilePath == "" {
		panic("need path")
	}
	termChangeInterval := viper.GetInt("annsensus.term_change_interval")
	annSensus := annsensus.NewAnnSensus(termChangeInterval, disableConsensus, cryptoType, campaign,
		partnerNum, genesisAccounts, consensFilePath, disableTermChange)
	autoClientManager := &AutoClientManager{
		SampleAccounts:         core.GetSampleAccounts(),
		NodeStatusDataProvider: org,
	}

	// TODO
	// RegisterNewTxHandler is not for AnnSensus sending txs out.
	// Not suitable to be used here.
	autoClientManager.RegisterReceiver = annSensus.RegisterNewTxHandler
	accountIds := StringArrayToIntArray(viper.GetStringSlice("auto_client.tx.account_ids"))
	//coinBaseId := accountIds[0] + 100

	// init key vault from a file
	privFilePath := io.FixPrefixPath(viper.GetString("datadir"), "privkey")
	if !io.FileExists(privFilePath) {
		panic("Please generate a private key first")
	}

	decrpted, err := encryption.DecryptFileDummy(privFilePath, "")
	if err != nil {
		panic("failed to decrypt private key")
	}
	vault := encryption.NewVault(decrpted)

	myAcount := account.NewAccount(string(vault.Fetch()))
	autoClientManager.Init(
		accountIds,
		delegate,
		myAcount,
	)
	// TODO
	// set annsensus's private key to be coinbase.

	*consensusVerifier = og.ConsensusVerifier{
		VerifyTermChange: annSensus.VerifyTermChange,
		VerifySequencer:  annSensus.VerifySequencer,
		VerifyCampaign:   annSensus.VerifyCampaign,
	}
	annSensus.InitAccount(myAcount, time.Millisecond*time.Duration(sequencerTime),
		autoClientManager.JudgeNonce, txCreator, org.Dag, txBuffer.SelfGeneratedNewTxChan,
		syncManager.IncrementalSyncer.HandleNewTxi, hub)
	logrus.Info("my pk ", annSensus.MyAccount.PublicKey.String())
	hub.SetEncryptionKey(&annSensus.MyAccount.PrivateKey)
	n.Components = append(n.Components, autoClientManager)
	syncManager.OnUpToDate = append(syncManager.OnUpToDate, autoClientManager.UpToDateEventListener)
	if !disableConsensus {
		syncManager.OnUpToDate = append(syncManager.OnUpToDate, annSensus.UpdateEvent)
	}
	hub.OnNewPeerConnected = append(hub.OnNewPeerConnected, syncManager.CatchupSyncer.NewPeerConnectedEventListener)
	if !disableConsensus {
		hub.OnNewPeerConnected = append(hub.OnNewPeerConnected, annSensus.NewPeerConnectedEventListener)
	}

	//init msg requst id
	og.MsgCountInit()

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
		privKey := getNodePrivKey()
		p2pServer = NewP2PServer(privKey, bootNode)
		p2pServer.Protocols = append(p2pServer.Protocols, hub.SubProtocols...)
		hub.NodeInfo = p2pServer.NodeInfo

		n.Components = append(n.Components, p2pServer)
	}

	if rpcServer != nil {
		rpcServer.C.P2pServer = p2pServer
		rpcServer.C.Og = org
		rpcServer.C.TxBuffer = txBuffer
		rpcServer.C.TxCreator = txCreator
		// just for debugging, ignoring index OOR
		rpcServer.C.NewRequestChan = autoClientManager.Clients[0].ManualChan
		rpcServer.C.SyncerManager = syncManager
		rpcServer.C.AutoTxCli = autoClientManager
		rpcServer.C.AnnSensus = annSensus
		rpcServer.C.PerformanceMonitor = pm

	}

	if viper.GetBool("websocket.enabled") {
		wsServer := wserver.NewServer(fmt.Sprintf(":%d", viper.GetInt("websocket.port")))
		n.Components = append(n.Components, wsServer)
		org.TxPool.RegisterOnNewTxReceived(wsServer.NewTxReceivedChan, "wsServer.NewTxReceivedChan", true)
		org.TxPool.OnBatchConfirmed = append(org.TxPool.OnBatchConfirmed, wsServer.BatchConfirmedChan)
		pm.Register(wsServer)
	}
	if status.ArchiveMode {
		logrus.Info("archive mode")
	}

	//txMetrics
	txCounter := performance.NewTxCounter()

	org.TxPool.RegisterOnNewTxReceived(txCounter.NewTxReceivedChan, "txCounter.NewTxReceivedChan", true)
	org.TxPool.OnBatchConfirmed = append(org.TxPool.OnBatchConfirmed, txCounter.BatchConfirmedChan)
	delegate.OnNewTxiGenerated = append(delegate.OnNewTxiGenerated, txCounter.NewTxGeneratedChan)
	n.Components = append(n.Components, txCounter)

	//
	if !disableConsensus {
		org.Dag.OnConsensusTXConfirmed = annSensus.ConsensusTXConfirmed
	}

	n.Components = append(n.Components, annSensus)
	m.ConsensusDkgDealHandler = annSensus
	m.ConsensusDkgDealResponseHandler = annSensus
	m.ConsensusDkgSigSetsHandler = annSensus
	m.ConsensusProposalHandler = annSensus
	m.ConsensusPreCommitHandler = annSensus
	m.ConsensusPreVoteHandler = annSensus
	m.ConsensusDkgGenesisPublicKeyHandler = annSensus
	m.TermChangeResponseHandler = annSensus
	m.TermChangeRequestHandler = annSensus
	txBuffer.OnProposalSeqCh = annSensus.ProposalSeqChan

	if !disableConsensus {
		org.TxPool.OnNewLatestSequencer = append(org.TxPool.OnNewLatestSequencer, annSensus.NewLatestSequencer)
	}

	//annSensus.RegisterNewTxHandler(txBuffer.ReceivedNewTxChan)

	pm.Register(org.TxPool)
	pm.Register(syncManager)
	pm.Register(syncManager.IncrementalSyncer)
	pm.Register(txBuffer)
	pm.Register(messageHandler)
	pm.Register(hub)
	pm.Register(txCounter)
	pm.Register(annSensus)
	n.Components = append(n.Components, pm)
	ioPerformance := ioperformance.Init()
	n.Components = append(n.Components, ioPerformance)
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
	status.NodeStopped = true
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
	hub.CallbackRegistry[og.MessageTypeGetMsg] = m.RouteGetMsg
	hub.CallbackRegistry[og.MessageTypeSequencerHeader] = m.RouteSequencerHeader
	hub.CallbackRegistry[og.MessageTypeBodiesRequest] = m.RouteBodiesRequest
	hub.CallbackRegistry[og.MessageTypeBodiesResponse] = m.RouteBodiesResponse
	hub.CallbackRegistry[og.MessageTypeTxsRequest] = m.RouteTxsRequest
	hub.CallbackRegistry[og.MessageTypeTxsResponse] = m.RouteTxsResponse
	hub.CallbackRegistry[og.MessageTypeHeaderRequest] = m.RouteHeaderRequest
	hub.CallbackRegistry[og.MessageTypeHeaderResponse] = m.RouteHeaderResponse
	hub.CallbackRegistry[og.MessageTypeControl] = m.RouteControlMsg
	hub.CallbackRegistry[og.MessageTypeCampaign] = m.RouteCampaign
	hub.CallbackRegistry[og.MessageTypeTermChange] = m.RouteTermChange
	hub.CallbackRegistry[og.MessageTypeArchive] = m.RouteArchive
	hub.CallbackRegistry[og.MessageTypeActionTX] = m.RouteActionTx
	hub.CallbackRegistry[og.MessageTypeConsensusDkgDeal] = m.RouteConsensusDkgDeal
	hub.CallbackRegistry[og.MessageTypeConsensusDkgDealResponse] = m.RouteConsensusDkgDealResponse
	hub.CallbackRegistry[og.MessageTypeConsensusDkgSigSets] = m.RouteConsensusDkgSigSets
	hub.CallbackRegistry[og.MessageTypeConsensusDkgGenesisPublicKey] = m.RouteConsensusDkgGenesisPublicKey
	hub.CallbackRegistry[og.MessageTypeProposal] = m.RouteConsensusProposal
	hub.CallbackRegistry[og.MessageTypePreVote] = m.RouteConsensusPreVote
	hub.CallbackRegistry[og.MessageTypePreCommit] = m.RouteConsensusPreCommit

	hub.CallbackRegistry[og.MessageTypeTermChangeRequest] = m.RouteTermChangeRequest
	hub.CallbackRegistry[og.MessageTypeTermChangeResponse] = m.RouteTermChangeResponse
}

func SetupCallbacksOG32(m *og.MessageRouterOG02, hub *og.Hub) {
	hub.CallbackRegistryOG02[og.GetNodeDataMsg] = m.RouteGetNodeDataMsg
	hub.CallbackRegistryOG02[og.NodeDataMsg] = m.RouteNodeDataMsg
	hub.CallbackRegistryOG02[og.GetReceiptsMsg] = m.RouteGetReceiptsMsg
}
