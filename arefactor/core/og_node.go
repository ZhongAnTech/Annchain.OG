package core

import (
	"github.com/annchain/OG/arefactor/consensus"
	"github.com/annchain/OG/arefactor/dummy"
	"github.com/annchain/OG/arefactor/og"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer"
	"github.com/annchain/OG/arefactor/performance"
	"github.com/annchain/OG/arefactor/rpc"
	"github.com/annchain/OG/arefactor/transport_interface"
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
	blockTime := time.Second * 3
	myId := viper.GetInt("id")
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

	cpSyncer := &ogsyncer.IntSyncer{
		Ledger: ledger,
	}
	cpSyncer.InitDefault()

	// OG engine
	cpOgEngine := &og.OgEngine{
		Ledger:           ledger,
		CommunityManager: cpCommunityManager,
	}
	cpCommunityManager.NodeInfoProvider = cpOgEngine

	cpOgEngine.InitDefault()
	cpOgEngine.StaticSetup()

	n.components = append(n.components, cpTransport)
	n.components = append(n.components, cpPerformanceMonitor)
	n.components = append(n.components, cpCommunityManager)
	n.components = append(n.components, cpOgEngine)
	n.components = append(n.components, cpRpc)
	n.components = append(n.components, cpSyncer)

	if viper.GetBool("features.consensus") {
		// consensus. Current all peers are Partner
		cpConsensusPartner := &consensus.Partner{
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
		}
		cpConsensusPartner.InitDefault()
		n.components = append(n.components, cpConsensusPartner)
		cpConsensusPartner.AddSubscriberNewOutgoingMessageEvent(cpTransport)
		cpTransport.AddSubscriberNewIncomingMessageEvent(cpConsensusPartner)
	}

	// event registration

	// message senders
	cpOgEngine.AddSubscriberNewOutgoingMessageEvent(cpTransport)
	cpCommunityManager.AddSubscriberNewOutgoingMessageEvent(cpTransport)
	cpSyncer.AddSubscriberNewOutgoingMessageEvent(cpTransport)

	// message receivers
	cpTransport.AddSubscriberNewIncomingMessageEvent(cpOgEngine)
	cpTransport.AddSubscriberNewIncomingMessageEvent(cpCommunityManager)
	cpTransport.AddSubscriberNewIncomingMessageEvent(cpSyncer)

	// peer connected
	cpTransport.AddSubscriberPeerConnectedEvent(cpCommunityManager)

	// peer joined and left to the network cluster (protocol verified)
	cpCommunityManager.AddSubscriberPeerJoinedEvent(cpOgEngine)
	cpCommunityManager.AddSubscriberPeerLeftEvent(cpSyncer)

	// peer height provided
	cpOgEngine.AddSubscriberNewHeightDetectedEvent(cpSyncer)

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
