package core

import (
	"github.com/sirupsen/logrus"
)

// SoloNode is the basic entrypoint for all modules to start.
type SoloNode struct {
	components []Component
}

// InitDefault only set necessary data structures.
// to Init a node with components, use Setup
func (n *SoloNode) InitDefault() {
	n.components = []Component{}
}

func (n *SoloNode) Setup() {
	n.components = append(n.components, getPerformanceMonitor())
}

func (n *SoloNode) Start() {
	for _, component := range n.components {
		logrus.Infof("Starting %s", component.Name())
		component.Start()
		logrus.Infof("Started: %s", component.Name())

	}
	logrus.Info("SoloNode Started")
}
func (n *SoloNode) Stop() {
	for i := len(n.components) - 1; i >= 0; i-- {
		comp := n.components[i]
		logrus.Infof("Stopping %s", comp.Name())
		comp.Stop()
		logrus.Infof("Stopped: %s", comp.Name())
	}
	logrus.Info("SoloNode Stopped")
}
