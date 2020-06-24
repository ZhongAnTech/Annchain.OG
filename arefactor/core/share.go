package core

import (
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/og"
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

func getTransport(accountHolder og.TransportAccountHolder) *transport.PhysicalCommunicator {
	account, err := accountHolder.ProvideAccount()
	if err != nil {
		// account not exists
		// generate or
		if !viper.GetBool("genkey") {
			logrus.WithError(err).Fatal("failed to read key file. You may generate one by specifying --genkey flag")
		}
		// generate
		account, err = accountHolder.Generate(nil)
		if err != nil {
			logrus.WithError(err).Fatal("failed to generate transport account")
		}
		err = accountHolder.Save()
		if err != nil {
			logrus.WithError(err).Fatal("failed to store account. account may lost after reboot so we quit.")
		}
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
