package core

import (
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/performance"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Node is the basic entrypoint for all modules to start.
type Node struct {
	components []Component
}

// InitDefault only set necessary data structures.
// to Init a node with components, use Setup
func (n *Node) InitDefault() {
	n.components = []Component{}
}

func (n *Node) Setup() {

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

	n.components = append(n.components, pm)
}

func (n *Node) Start() {
	for _, component := range n.components {
		logrus.Infof("Starting %s", component.Name())
		component.Start()
		logrus.Infof("Started: %s", component.Name())

	}
	logrus.Info("Node Started")
}
func (n *Node) Stop() {
	for i := len(n.components) - 1; i >= 0; i-- {
		comp := n.components[i]
		logrus.Infof("Stopping %s", comp.Name())
		comp.Stop()
		logrus.Infof("Stopped: %s", comp.Name())
	}
	logrus.Info("Node Stopped")
}
