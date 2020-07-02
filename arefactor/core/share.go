package core

import (
	"github.com/annchain/OG/arefactor/committee"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/og"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/performance"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	LogDir     = "log"
	DataDir    = "data"
	ConfigDir  = "config"
	PrivateDir = "private"
)

func getPerformanceMonitor() *performance.PerformanceMonitor {
	hostname := utilfuncs.GetHostName()
	reporter := &performance.SoccerdashReporter{
		Id:         hostname + viper.GetString("id"),
		IpPort:     viper.GetString("report.address"),
		BufferSize: viper.GetInt("report.buffer_size"),
	}
	reporter.InitDefault()

	pm := &performance.PerformanceMonitor{
		Reporters: []performance.PerformanceReporter{
			reporter,
		},
	}
	return pm
}

func ensureLedgerAccountProvider(accountProvider og_interface.LedgerAccountProvider) {
	_, err := accountProvider.ProvideAccount()
	if err != nil {
		// ledger account not exists
		// generate or
		if !viper.GetBool("genkey") {
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
		if !viper.GetBool("genkey") {
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

func getTransport(accountProvider og.TransportAccountProvider) *transport.PhysicalCommunicator {
	account, err := accountProvider.ProvideAccount()
	if err != nil {
		logrus.WithError(err).Fatal("failed to provide transport account")
	}

	hostname := utilfuncs.GetHostName()
	reporter := &performance.SoccerdashReporter{
		Id:         hostname + viper.GetString("id"),
		IpPort:     viper.GetString("report.address_log"),
		BufferSize: viper.GetInt("report.buffer_size"),
	}
	reporter.InitDefault()

	p2p := &transport.PhysicalCommunicator{
		Port:            viper.GetInt("p2p.port"),
		PrivateKey:      account.PrivateKey,
		ProtocolId:      viper.GetString("p2p.network_id"),
		NetworkReporter: reporter,
	}
	p2p.InitDefault()
	// load known peers
	return p2p
}

func loadLedgerCommittee(ledger *og.IntArrayLedger, provider *og.LocalLedgerAccountProvider) consensus_interface.CommitteeProvider {
	ledgerCommittee := ledger.CurrentCommittee()
	blsCommitteeProvider := &committee.BlsCommitteeProvider{}

	account, err := provider.ProvideAccount()
	if err != nil {
		logrus.WithError(err).Fatal("failed to provide ledger account")
	}
	var members []consensus_interface.CommitteeMember
	for _, v := range ledgerCommittee.Peers {
		members = append(members, *v)
	}

	blsCommitteeProvider.InitCommittee(ledgerCommittee.Version, members, account.Address.AddressString())

	return blsCommitteeProvider
}
