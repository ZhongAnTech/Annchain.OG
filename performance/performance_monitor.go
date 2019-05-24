// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package performance

import (
	"github.com/annchain/OG/common/goroutine"
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
	goroutine.NewRoutine( func() {
		p.quit = false
		//runtime.SetBlockProfileRate(1)

		for !p.quit {
			fields := logrus.Fields(p.CollectData())
			logrus.WithFields(fields).Info("Performance")
			//pprof.Lookup("block").WriteTo(os.Stdout, 1)

			time.Sleep(time.Second * 5)
		}
	})
}

func (p *PerformanceMonitor) Stop() {
	p.quit = true
}

func (PerformanceMonitor) Name() string {
	return "PerformanceMonitor"
}

func (p *PerformanceMonitor) CollectData() map[string]interface{} {
	data := make(map[string]interface{})
	for _, ch := range p.reporters {
		data[ch.Name()] = ch.GetBenchmarks()
	}
	// add additional fields
	data["goroutines"] = runtime.NumGoroutine()
    data["goroutineNUmbers"] = goroutine.GetGoRoutineNum()
	return data
}
