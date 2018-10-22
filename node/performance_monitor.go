package node

import (
	"github.com/sirupsen/logrus"
	"runtime"
	"time"
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
		//runtime.SetBlockProfileRate(1)

		for !p.quit {
			fields := logrus.Fields{}
			for _, ch := range p.reporters {
				fields[ch.Name()] = ch.GetBenchmarks()
			}
			// add additional fields
			fields["goroutines"] = runtime.NumGoroutine()

			logrus.WithFields(fields).Info("Performance")
			//pprof.Lookup("block").WriteTo(os.Stdout, 1)

			time.Sleep(time.Second * 15)
		}
	}()
}

func (p *PerformanceMonitor) Stop() {
	p.quit = true
}

func (PerformanceMonitor) Name() string {
	return "PerformanceMonitor"
}
