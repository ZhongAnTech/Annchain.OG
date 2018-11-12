package performance

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
			fields := logrus.Fields(p.CollectData())
			logrus.WithFields(fields).Info("Performance")
			//pprof.Lookup("block").WriteTo(os.Stdout, 1)

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


func (p *PerformanceMonitor) CollectData() map[string]interface{}{
	data := make(map[string]interface{})
	for _, ch := range p.reporters {
		data[ch.Name()] = ch.GetBenchmarks()
	}
	// add additional fields
	data["goroutines"] = runtime.NumGoroutine()

	return data
}