package node

import (
	"github.com/sirupsen/logrus"
	"time"
)

type PerformanceReporter interface {
	Name() string
	GetBenchmarks() map[string]int
}

type PerformanceMonitor struct {
	reporters []PerformanceReporter
	quit      bool
}

func (p *PerformanceMonitor) Register(holder PerformanceReporter) {
	p.reporters = append(p.reporters, holder)
}

func (p *PerformanceMonitor) Start() {
	go func() {
		p.quit = false
		for !p.quit {
			fields := logrus.Fields{}
			for _, ch := range p.reporters {
				fields[ch.Name()] = ch.GetBenchmarks()
			}
			logrus.WithFields(fields).Warn("Performance")
			time.Sleep(time.Second * 5)
		}
	}()
}

func (p *PerformanceMonitor) Stop() {
	p.quit = true
}

func (PerformanceMonitor) Name() string {
	return "PerformanceMonitor"
}
