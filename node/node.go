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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/encryption"
	"github.com/annchain/OG/common/io"
	"github.com/annchain/OG/p2p/ioperformance"
	"github.com/annchain/OG/rpc"
	"github.com/annchain/OG/status"
	"github.com/annchain/OG/types/p2p_message"
	"time"

	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/og/syncer"
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

	// Crypto and signers
	var cryptoType crypto.CryptoType
	switch viper.GetString("crypto.algorithm") {
	case "ed25519":
		cryptoType = crypto.CryptoTypeEd25519
	case "secp256k1":
		cryptoType = crypto.CryptoTypeSecp256k1
	default:
		logrus.Fatal("Unknown crypto algorithm: " + viper.GetString("crypto.algorithm"))
	}
	//set default signer
	crypto.Signer = crypto.NewSigner(cryptoType)
	// Setup crypto algorithm
	if crypto.Signer.CanRecoverPubFromSig() {
		types.CanRecoverPubFromSig = true
	}
	// network id is configured either in config.toml or env variable
	networkId := viper.GetInt64("p2p.network_id")
	logrus.Infof("get network_id: %d", networkId)
	if networkId == 0 {
		networkId = defaultNetworkId
	}

	// OG init
	org, err := og.NewOg(
		og.OGConfig{
			NetworkId:   uint64(networkId),
			GenesisPath: viper.GetString("dag.genesis_path"),
		},
	)
	if err != nil {
		logrus.WithError(err).Warning("Error occurred while initializing OG")
		logrus.WithError(err).Fatal(fmt.Sprintf("Error occurred while initializing OG %v", err))
	}

	// Hub
	maxPeers := viper.GetInt("p2p.max_peers")
	if maxPeers == 0 {
		maxPeers = defaultMaxPeers
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

	// let og be the status source of hub to provide chain info to other peers
	hub.StatusDataProvider = org

	n.Components = append(n.Components, org)
	n.Components = append(n.Components, hub)

	// my account
	// init key vault from a file
	privFilePath := io.FixPrefixPath(viper.GetString("datadir"), "privkey")
	if !io.FileExists(privFilePath) {
		if viper.GetBool("genkey") {
			logrus.Info("We will generate a private key for you. Please store it carefully.")
			priv, _ := account.GenAccount()
			account.SavePrivateKey(privFilePath, priv.String())
		} else {
			logrus.Fatal("Please generate a private key under " + privFilePath + " , or specify --genkey to let us generate one for you")
		}
	}

	decrpted, err := encryption.DecryptFileDummy(privFilePath, "")
	if err != nil {
		logrus.WithError(err).Fatal("failed to decrypt private key")
	}
	vault := encryption.NewVault(decrpted)

	myAcount := account.NewAccount(string(vault.Fetch()))

	// p2p server
	privKey := getNodePrivKey()
	// isBootNode must be set to false if you need a centralized server to collect and dispatch bootstrap
	if viper.GetBool("p2p.enabled") && viper.GetString("p2p.bootstrap_nodes") == "" {

		// get my url and then send to centralized bootstrap server if there is no bootstrap server specified
		nodeURL := getOnodeURL(privKey)
		buildBootstrap(networkId, nodeURL, &myAcount.PublicKey)
	}

	var p2pServer *p2p.Server
	if viper.GetBool("p2p.enabled") {
		isBootNode := viper.GetBool("p2p.bootstrap_node")
		p2pServer = NewP2PServer(privKey, isBootNode)

		p2pServer.Protocols = append(p2pServer.Protocols, hub.SubProtocols...)
		hub.NodeInfo = p2pServer.NodeInfo

		n.Components = append(n.Components, p2pServer)
	}

	// transaction verifiers
	graphVerifier := &og.GraphVerifier{
		Dag:    org.Dag,
		TxPool: org.TxPool,
		//Buffer: txBuffer,
	}

	txFormatVerifier := &og.TxFormatVerifier{
		MaxTxHash:    common.HexToHash(viper.GetString("max_tx_hash")),
		MaxMinedHash: common.HexToHash(viper.GetString("max_mined_hash")),
	}
	if txFormatVerifier.MaxMinedHash == common.HexToHash("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") {
		txFormatVerifier.NoVerifyMindHash = true
		logrus.Info("no verify mined hash")
	}
	if txFormatVerifier.MaxTxHash == common.HexToHash("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") {
		txFormatVerifier.NoVerifyMaxTxHash = true
		logrus.Info("no verify max tx hash")
	}
	txFormatVerifier.NoVerifySignatrue = viper.GetBool("tx_buffer.no_verify_signature")
	//verify format first , set address and then verify graph
	verifiers := []og.Verifier{txFormatVerifier, graphVerifier}

	// txBuffer
	txBuffer := og.NewTxBuffer(og.TxBufferConfig{
		Verifiers:                        verifiers,
		Dag:                              org.Dag,
		TxPool:                           org.TxPool,
		DependencyCacheExpirationSeconds: 1 * 60 * 60,
		DependencyCacheMaxSize:           100000,
		NewTxQueueSize:                   viper.GetInt("tx_buffer.new_tx_queue_size"),
		KnownCacheMaxSize:                100000,
		KnownCacheExpirationSeconds:      1 * 60 * 60,
		AddedToPoolQueueSize:             10000,
		TestNoVerify:                     viper.GetBool("tx_buffer.test_no_verify"),
	})
	// let txBuffer judge whether the tx is received previously
	hub.IsReceivedHash = txBuffer.IsReceivedHash
	org.TxBuffer = txBuffer
	n.Components = append(n.Components, txBuffer)

	// syncBuffer
	syncBuffer := syncer.NewSyncBuffer(syncer.SyncBufferConfig{
		TxPool:    org.TxPool,
		Dag:       org.Dag,
		Verifiers: verifiers,
	})
	n.Components = append(n.Components, syncBuffer)

	isBootNode := viper.GetBool("p2p.bootstrap_node")
	// syncManager
	syncManager := syncer.NewSyncManager(syncer.SyncManagerConfig{
		Mode:           downloader.FullSync,
		ForceSyncCycle: uint(viper.GetInt("hub.sync_cycle_ms")),
		BootstrapNode:  isBootNode,
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
		BootStrapNode:          isBootNode,
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
			AcquireTxDedupCacheExpirationSeconds:     600,
			BufferedIncomingTxCacheExpirationSeconds: 3600,
			BufferedIncomingTxCacheMaxSize:           100000,
			FiredTxCacheExpirationSeconds:            3600,
			FiredTxCacheMaxSize:                      30000,
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
		TipGenerator:       org.TxPool, //og.NewFIFOTIpGenerator(org.TxPool, 6),
		MaxConnectingTries: 100,
		MaxTxHash:          txFormatVerifier.MaxTxHash,
		MaxMinedHash:       txFormatVerifier.MaxMinedHash,
		NoVerifyMindHash:   txFormatVerifier.NoVerifyMindHash,
		NoVerifyMaxTxHash:  txFormatVerifier.NoVerifyMaxTxHash,
		DebugNodeId:        viper.GetInt("debug.node_id"),
		GraphVerifier:      graphVerifier,
		GetStateRoot:       org.TxPool,
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
		Dag:              org.Dag,
		TxCreator:        txCreator,
		InsertSyncBuffer: syncBuffer.AddTxs,
	}

	delegate.OnNewTxiGenerated = append(delegate.OnNewTxiGenerated, txBuffer.SelfGeneratedNewTxChan)
	delegate.ReceivedNewTxsChan = txBuffer.ReceivedNewTxsChan
	genesisAccounts := parserGenesisAccounts(viper.GetString("annsensus.genesis_pk"))

	mode := viper.GetString("mode")
	if mode == "archive" {
		status.ArchiveMode = true
	}

	autoClientManager := &AutoClientManager{
		SampleAccounts:         core.GetSampleAccounts(),
		NodeStatusDataProvider: org,
	}

	disableConsensus := viper.GetBool("annsensus.disable")
	//TODO temperary , delete this after demo
	//if archiveMode {
	//	disableConsensus = true
	//}

	var annSensus *annsensus.AnnSensus
	if !disableConsensus {
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
			logrus.Fatal("need path")
		}
		termChangeInterval := viper.GetInt("annsensus.term_change_interval")
		annSensus = annsensus.NewAnnSensus(termChangeInterval, disableConsensus, cryptoType, campaign,
			partnerNum, genesisAccounts, consensFilePath, disableTermChange)
		// TODO
		// RegisterNewTxHandler is not for AnnSensus sending txs out.
		// Not suitable to be used here.
		autoClientManager.RegisterReceiver = annSensus.RegisterNewTxHandler
		// TODO
		// set annsensus's private key to be coinbase.

		consensusVerifier := &og.ConsensusVerifier{
			VerifyTermChange: annSensus.VerifyTermChange,
			VerifySequencer:  annSensus.VerifySequencer,
			VerifyCampaign:   annSensus.VerifyCampaign,
		}
		txBuffer.Verifiers = append(txBuffer.Verifiers, consensusVerifier)

		annSensus.InitAccount(myAcount, time.Millisecond*time.Duration(sequencerTime),
			autoClientManager.JudgeNonce, txCreator, org.Dag, txBuffer.SelfGeneratedNewTxChan,
			syncManager.IncrementalSyncer.HandleNewTxi, hub)
		logrus.Info("my pk ", annSensus.MyAccount.PublicKey.String())
		hub.SetEncryptionKey(&annSensus.MyAccount.PrivateKey)

		syncManager.OnUpToDate = append(syncManager.OnUpToDate, annSensus.UpdateEvent)
		hub.OnNewPeerConnected = append(hub.OnNewPeerConnected, annSensus.NewPeerConnectedEventListener)
		org.Dag.OnConsensusTXConfirmed = annSensus.ConsensusTXConfirmed

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

		org.TxPool.OnNewLatestSequencer = append(org.TxPool.OnNewLatestSequencer, annSensus.NewLatestSequencer)
		pm.Register(annSensus)
	}

	accountIds := StringArrayToIntArray(viper.GetStringSlice("auto_client.tx.account_ids"))
	//coinBaseId := accountIds[0] + 100

	autoClientManager.Init(
		accountIds,
		delegate,
		myAcount,
	)

	n.Components = append(n.Components, autoClientManager)
	syncManager.OnUpToDate = append(syncManager.OnUpToDate, autoClientManager.UpToDateEventListener)
	hub.OnNewPeerConnected = append(hub.OnNewPeerConnected, syncManager.CatchupSyncer.NewPeerConnectedEventListener)

	//init msg requst id
	p2p_message.MsgCountInit()

	// DataLoader
	dataLoader := &og.DataLoader{
		Dag:    org.Dag,
		TxPool: org.TxPool,
	}
	n.Components = append(n.Components, dataLoader)

	n.Components = append(n.Components, m)
	org.Manager = m

	// rpc server
	var rpcServer *rpc.RpcServer
	if viper.GetBool("rpc.enabled") {
		rpcServer = rpc.NewRpcServer(viper.GetString("rpc.port"))
		n.Components = append(n.Components, rpcServer)
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
		if !disableConsensus {
			rpcServer.C.AnnSensus = annSensus
		}

		rpcServer.C.PerformanceMonitor = pm
	}

	// websocket server
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

	//annSensus.RegisterNewTxHandler(txBuffer.ReceivedNewTxChan)

	pm.Register(org.TxPool)
	pm.Register(syncManager)
	pm.Register(syncManager.IncrementalSyncer)
	pm.Register(txBuffer)
	pm.Register(messageHandler)
	pm.Register(hub)
	pm.Register(txCounter)

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
	hub.CallbackRegistry[p2p_message.MessageTypePing] = m.RoutePing
	hub.CallbackRegistry[p2p_message.MessageTypePong] = m.RoutePong
	hub.CallbackRegistry[p2p_message.MessageTypeFetchByHashRequest] = m.RouteFetchByHashRequest
	hub.CallbackRegistry[p2p_message.MessageTypeFetchByHashResponse] = m.RouteFetchByHashResponse
	hub.CallbackRegistry[p2p_message.MessageTypeNewTx] = m.RouteNewTx
	hub.CallbackRegistry[p2p_message.MessageTypeNewTxs] = m.RouteNewTxs
	hub.CallbackRegistry[p2p_message.MessageTypeNewSequencer] = m.RouteNewSequencer
	hub.CallbackRegistry[p2p_message.MessageTypeGetMsg] = m.RouteGetMsg
	hub.CallbackRegistry[p2p_message.MessageTypeSequencerHeader] = m.RouteSequencerHeader
	hub.CallbackRegistry[p2p_message.MessageTypeBodiesRequest] = m.RouteBodiesRequest
	hub.CallbackRegistry[p2p_message.MessageTypeBodiesResponse] = m.RouteBodiesResponse
	hub.CallbackRegistry[p2p_message.MessageTypeTxsRequest] = m.RouteTxsRequest
	hub.CallbackRegistry[p2p_message.MessageTypeTxsResponse] = m.RouteTxsResponse
	hub.CallbackRegistry[p2p_message.MessageTypeHeaderRequest] = m.RouteHeaderRequest
	hub.CallbackRegistry[p2p_message.MessageTypeHeaderResponse] = m.RouteHeaderResponse
	hub.CallbackRegistry[p2p_message.MessageTypeControl] = m.RouteControlMsg
	hub.CallbackRegistry[p2p_message.MessageTypeCampaign] = m.RouteCampaign
	hub.CallbackRegistry[p2p_message.MessageTypeTermChange] = m.RouteTermChange
	hub.CallbackRegistry[p2p_message.MessageTypeArchive] = m.RouteArchive
	hub.CallbackRegistry[p2p_message.MessageTypeActionTX] = m.RouteActionTx
	hub.CallbackRegistry[p2p_message.MessageTypeConsensusDkgDeal] = m.RouteConsensusDkgDeal
	hub.CallbackRegistry[p2p_message.MessageTypeConsensusDkgDealResponse] = m.RouteConsensusDkgDealResponse
	hub.CallbackRegistry[p2p_message.MessageTypeConsensusDkgSigSets] = m.RouteConsensusDkgSigSets
	hub.CallbackRegistry[p2p_message.MessageTypeConsensusDkgGenesisPublicKey] = m.RouteConsensusDkgGenesisPublicKey
	hub.CallbackRegistry[p2p_message.MessageTypeProposal] = m.RouteConsensusProposal
	hub.CallbackRegistry[p2p_message.MessageTypePreVote] = m.RouteConsensusPreVote
	hub.CallbackRegistry[p2p_message.MessageTypePreCommit] = m.RouteConsensusPreCommit

	hub.CallbackRegistry[p2p_message.MessageTypeTermChangeRequest] = m.RouteTermChangeRequest
	hub.CallbackRegistry[p2p_message.MessageTypeTermChangeResponse] = m.RouteTermChangeResponse
}

func SetupCallbacksOG32(m *og.MessageRouterOG02, hub *og.Hub) {
	hub.CallbackRegistryOG02[p2p_message.GetNodeDataMsg] = m.RouteGetNodeDataMsg
	hub.CallbackRegistryOG02[p2p_message.NodeDataMsg] = m.RouteNodeDataMsg
	hub.CallbackRegistryOG02[p2p_message.GetReceiptsMsg] = m.RouteGetReceiptsMsg
}
