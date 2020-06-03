package core

import (
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/performance"
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
		Id:         hostname,
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
