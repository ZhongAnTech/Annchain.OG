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
	"fmt"
	"github.com/annchain/OG/arefactor/common/goroutine"
	"runtime"
	"time"
)

type PerformanceReporter interface {
	Report(key string, value interface{})
}

type PerformanceDataProvider interface {
	Name() string
	GetBenchmarks() map[string]interface{}
}

type PerformanceMonitor struct {
	dataProviders []PerformanceDataProvider
	quit          bool
	Reporters     []PerformanceReporter
}

func (p *PerformanceMonitor) Register(dataProvider PerformanceDataProvider) {
	p.dataProviders = append(p.dataProviders, dataProvider)
}

func (p *PerformanceMonitor) Start() {
	goroutine.New(func() {
		p.quit = false
		//runtime.SetBlockProfileRate(1)

		for !p.quit {
			fmt.Println("reporting")
			for key, value := range p.CollectData() {
				for _, reporter := range p.Reporters {
					reporter.Report(key, value)
				}
			}

			time.Sleep(time.Second * 10)
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
	for _, ch := range p.dataProviders {
		data[ch.Name()] = ch.GetBenchmarks()
	}
	// add additional fields
	data["goroutines"] = runtime.NumGoroutine()
	data["goroutineNUmbers"] = goroutine.GetGoRoutineNum()
	return data
}
