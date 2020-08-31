package core

import (
	"github.com/annchain/OG/arefactor/consensus"
	"github.com/annchain/OG/arefactor/consts"
	"github.com/annchain/OG/arefactor/dummy"
	"github.com/annchain/OG/arefactor/og"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer"
	"github.com/annchain/OG/arefactor/performance"
	"github.com/annchain/OG/arefactor/rpc"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/latifrons/go-eventbus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"path"
	"time"
)

// OgNode is the basic entry point for all modules to start.
type OgNode struct {
	FolderConfig             FolderConfig
	components               []Component
	transportAccountProvider og.TransportAccountProvider
}

// InitDefault only set necessary data structures.
// to Init a node with components, use Setup
func (n *OgNode) InitDefault() {
	n.components = []Component{}
}

func (n *OgNode) Setup() {
	blockTime := time.Second * 1
	myId := viper.GetInt("id")

	ebus := &eventbus.EventBus{
		TimeoutControl: true, // for debugging
		Timeout:        time.Second * 5,
	}
	// reg events
	for eventCode, eventValue := range consts.EventCodeTextMap {
		ebus.RegisterEventType(int(eventCode), eventValue)
	}

	// reporter
	lowLevelReporter := getReporter()

	// load private info
	privateGenerator := &og.CachedPrivateGenerator{}

	// account management
	// check if file exists
	n.transportAccountProvider = &og.LocalTransportAccountProvider{
		PrivateGenerator:   privateGenerator,
		NetworkIdConverter: &og.OgNetworkIdConverter{},
		BackFilePath:       path.Join(n.FolderConfig.Private, "transport.key"),
		CryptoType:         transport_interface.CryptoTypeSecp256k1,
	}

	ogAddressConverter := &og.OgAddressConverter{}

	ledgerAccountProvider := &og.LocalLedgerAccountProvider{
		PrivateGenerator: privateGenerator,
		AddressConverter: ogAddressConverter,
		BackFilePath:     path.Join(n.FolderConfig.Private, "account.key"),
		CryptoType:       og_interface.CryptoTypeSecp256k1,
	}

	// bls account should be loaded along with the committee number.
	consensusAccountProvider := &dummy.DummyConsensusAccountProvider{
		BackFilePath: path.Join(n.FolderConfig.Private, "dummy_consensus.key"),
	}

	// load transport key
	ensureTransportAccountProvider(n.transportAccountProvider)

	// load account key (for consensus)
	ensureLedgerAccountProvider(ledgerAccountProvider)

	// load consensus key
	ensureConsensusAccountProvider(consensusAccountProvider)

	//consensusAccountProvider.Save()

	// ledger implementation
	ledger := &dummy.IntArrayLedger{
		EventBus:   ebus,
		DataPath:   n.FolderConfig.Data,
		ConfigPath: n.FolderConfig.Config,
		Reporter:   lowLevelReporter,
	}
	ledger.InitDefault()
	ledger.StaticSetup()
	//ledger.SaveConsensusCommittee()

	// load from ledger
	committeeProvider := loadLedgerCommittee(ledger, consensusAccountProvider)

	// consensus signer
	//consensusSigner := consensus.BlsSignatureCollector{}

	performanceReporter := &performance.SoccerdashReporter{
		Reporter: lowLevelReporter,
	}

	// low level transport (libp2p)
	cpPerformanceMonitor := getPerformanceMonitor(lowLevelReporter)
	cpTransport := getTransport(n.transportAccountProvider, performanceReporter)

	// peer relationship management
	cpCommunityManager := &og.DefaultCommunityManager{
		EventBus:              ebus,
		NodeInfoProvider:      nil,
		PhysicalCommunicator:  cpTransport,
		KnownPeerListFilePath: path.Join(n.FolderConfig.Config, "peers.lst"),
	}
	cpCommunityManager.InitDefault()
	cpCommunityManager.StaticSetup()

	proposalGenerator := &dummy.IntArrayProposalGenerator{
		Ledger:    ledger,
		BlockTime: blockTime,
		MyId:      myId,
	}
	proposalGenerator.InitDefault()

	cpController := &rpc.RpcController{
		Ledger:                    ledger,
		CpDefaultCommunityManager: cpCommunityManager,
	}

	// rpcs
	cpRpc := &rpc.RpcServer{
		Controller: cpController,
		Port:       viper.GetInt("rpc.port"),
	}

	cpRpc.InitDefault()

	cpContentFetcher := &ogsyncer.RandomPickerContentFetcher{
		EventBus:                ebus,
		ExpireDuration:          time.Minute * 5,
		MinimumIntervalDuration: time.Second * 5,
		MaxTryTimes:             10,
		Reporter:                lowLevelReporter,
	}

	cpContentFetcher.InitDefault()

	cpIntLedgerSyncer := &ogsyncer.IntLedgerSyncer{
		Ledger:         ledger,
		ContentFetcher: cpContentFetcher,
	}
	cpIntLedgerSyncer.InitDefault()

	// OG engine
	cpOgEngine := &og.OgEngine{
		EventBus:         ebus,
		Ledger:           ledger,
		CommunityManager: cpCommunityManager,
		NetworkId:        "1",
	}
	cpCommunityManager.NodeInfoProvider = cpOgEngine

	cpOgEngine.InitDefault()
	cpOgEngine.StaticSetup()

	n.components = append(n.components, cpTransport)
	n.components = append(n.components, cpPerformanceMonitor)
	n.components = append(n.components, cpCommunityManager)
	n.components = append(n.components, cpOgEngine)
	n.components = append(n.components, cpRpc)
	n.components = append(n.components, cpContentFetcher)
	n.components = append(n.components, cpIntLedgerSyncer)

	if viper.GetBool("features.consensus") {
		// consensus. Current all peers are Partner
		cpConsensusPartner := &consensus.Partner{
			EventBus:          ebus,
			Logger:            logrus.StandardLogger(),
			Reporter:          lowLevelReporter,
			ProposalGenerator: proposalGenerator,
			ProposalVerifier:  &dummy.DummyProposalVerifier{},
			CommitteeProvider: committeeProvider,
			ConsensusSigner: &dummy.DummyConsensusSigner{
				Id: myId,
			}, // should be replaced by bls signer
			ConsensusAccountProvider: consensusAccountProvider,
			Hasher:                   &consensus.SHA256Hasher{},
			Ledger:                   ledger,
			BlockTime:                blockTime,
			TimeoutTime:              blockTime + time.Second*3,
		}
		cpConsensusPartner.InitDefault()
		n.components = append(n.components, cpConsensusPartner)

		ebus.Subscribe(int(consts.NewIncomingMessageEvent), cpConsensusPartner)
	}

	// event registration
	// message senders
	ebus.Subscribe(int(consts.NewOutgoingMessageEvent), cpTransport)
	// message receivers
	ebus.Subscribe(int(consts.NewIncomingMessageEvent), cpOgEngine)
	ebus.Subscribe(int(consts.NewIncomingMessageEvent), cpCommunityManager)
	ebus.Subscribe(int(consts.NewIncomingMessageEvent), cpIntLedgerSyncer)

	// peer connected
	ebus.Subscribe(int(consts.PeerConnectedEvent), cpCommunityManager)

	// peer joined and left to the network cluster (protocol verified)
	ebus.Subscribe(int(consts.PeerJoinedEvent), cpOgEngine)
	ebus.Subscribe(int(consts.PeerJoinedEvent), cpContentFetcher)
	//cpCommunityManager.AddSubscriberPeerLeftEvent(cpSyncer)
	ebus.Subscribe(int(consts.UnknownNeededEvent), cpIntLedgerSyncer)

	// peer height provided
	ebus.Subscribe(int(consts.NewHeightDetectedEvent), cpContentFetcher)
	ebus.Subscribe(int(consts.NewHeightDetectedEvent), cpIntLedgerSyncer)
	ebus.Subscribe(int(consts.NewHeightBlockSyncedEvent), cpContentFetcher)

	// performance monitor registration
	cpPerformanceMonitor.Register(cpOgEngine)
}

func (n *OgNode) Start() {
	for _, component := range n.components {
		logrus.Infof("Starting %s", component.Name())
		component.Start()
		logrus.Infof("Started: %s", component.Name())

	}
	logrus.Info("OgNode Started")
}

func (n *OgNode) Stop() {
	for i := len(n.components) - 1; i >= 0; i-- {
		comp := n.components[i]
		logrus.Infof("Stopping %s", comp.Name())
		comp.Stop()
		logrus.Infof("Stopped: %s", comp.Name())
	}
	logrus.Info("OgNode Stopped")
}
