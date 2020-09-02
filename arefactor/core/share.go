package core

import (
	"github.com/annchain/OG/arefactor/committee"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/dummy"
	"github.com/annchain/OG/arefactor/og"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/performance"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/commongo/utilfuncs"
	"github.com/latifrons/go-eventbus"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	LogDir     = "log"
	DataDir    = "data"
	ConfigDir  = "config"
	PrivateDir = "private"
)

func getReporter() *soccerdash.Reporter {
	hostname := utilfuncs.GetHostName()
	reporter := &soccerdash.Reporter{
		Id:            hostname + viper.GetString("id"),
		TargetAddress: viper.GetString("report.address"),
		BufferSize:    100, //viper.GetInt("report.buffer_size"),
		Logger:        nil,
		//Logger:        logrus.StandardLogger(),

	}
	return reporter
}

func getPerformanceMonitor(reporter *soccerdash.Reporter) *performance.PerformanceMonitor {
	performanceReporter := &performance.SoccerdashReporter{
		Reporter: reporter,
	}
	pm := &performance.PerformanceMonitor{
		Reporters: []performance.PerformanceReporter{
			performanceReporter,
		},
	}
	return pm
}

func ensureLedgerAccountProvider(accountProvider og_interface.LedgerAccountProvider) {
	_, err := accountProvider.ProvideAccount()
	if err != nil {
		// ledger account not exists
		// generate or
		if !viper.GetBool("gen.key") {
			logrus.WithError(err).Fatal("failed to read ledger key file. You may generate one by specifying --genkey flag")
		}
		// generate
		_, err = accountProvider.Generate()
		if err != nil {
			logrus.WithError(err).Fatal("failed to generate ledger account")
		}
		err = accountProvider.Save()
		if err != nil {
			logrus.WithError(err).Fatal("failed to store ledger account. account may lost after reboot so we quit.")
		}
	}
}

func ensureTransportAccountProvider(accountProvider og.TransportAccountProvider) {
	_, err := accountProvider.ProvideAccount()
	if err != nil {
		// account not exists
		// generate or
		if !viper.GetBool("gen.key") {
			logrus.WithError(err).Fatal("failed to read transport key file. You may generate one by specifying --genkey flag")
		}
		// generate
		_, err = accountProvider.Generate()
		if err != nil {
			logrus.WithError(err).Fatal("failed to generate transport account")
		}
		err = accountProvider.Save()
		if err != nil {
			logrus.WithError(err).Fatal("failed to store account. account may lost after reboot so we quit.")
		}
	}
}

func ensureConsensusAccountProvider(provider consensus_interface.ConsensusAccountProvider) {
	_, err := provider.ProvideAccount()
	if err != nil {
		logrus.WithError(err).Fatal("failed to load consensus account. You must own bls key from discussion first")
	}
}

func getTransport(accountProvider og.TransportAccountProvider,
	reporter performance.PerformanceReporter,
	ebus *eventbus.EventBus,
) *transport.PhysicalCommunicator {
	account, err := accountProvider.ProvideAccount()
	if err != nil {
		logrus.WithError(err).Fatal("failed to provide transport account")
	}

	p2p := &transport.PhysicalCommunicator{
		EventBus:        ebus,
		Port:            viper.GetInt("p2p.port"),
		PrivateKey:      account.PrivateKey,
		ProtocolId:      viper.GetString("p2p.network_id"),
		NetworkReporter: reporter,
	}
	p2p.InitDefault()
	// load known peers
	return p2p
}

func loadLedgerCommittee(ledger *dummy.IntArrayLedger, provider consensus_interface.ConsensusAccountProvider) consensus_interface.CommitteeProvider {
	ledgerCommittee := ledger.CurrentCommittee()
	//blsCommitteeProvider := &committee.BlsCommitteeProvider{}
	committeeProvider := &committee.PlainBftCommitteeProvider{}

	account, err := provider.ProvideAccount()
	if err != nil {
		logrus.WithError(err).Fatal("failed to provide ledger account")
	}
	var members []consensus_interface.CommitteeMember
	for _, v := range ledgerCommittee.Peers {
		members = append(members, *v)
	}
	committeeProvider.InitCommittee(ledgerCommittee.Version, members, account)

	return committeeProvider
}
