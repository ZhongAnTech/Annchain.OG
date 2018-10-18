package node

import (
	"github.com/sirupsen/logrus"
	"time"
	"runtime"
	"runtime/pprof"
	"os"
)

type PerformanceReporter interface {
	Name() string
	GetBenchmarks() map[string]interface{}
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
			// add additional fields
			fields["goroutines"] = runtime.NumGoroutine()

			logrus.WithFields(fields).Info("Performance")
			pprof.Lookup("block").WriteTo(os.Stdout, 1)

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
