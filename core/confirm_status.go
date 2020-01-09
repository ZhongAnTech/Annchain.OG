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
package core

import (
	"fmt"
	"sync"
	"time"
)

type ConfirmStatus struct {
	TxNum       uint64
	ConfirmNum  uint64
	Confirm     time.Duration
	mu          sync.RWMutex
	initTime    time.Time
	RefreshTime time.Duration
}

type ConfirmInfo struct {
	ConfirmTime string `json:"confirm_time"`
	ConfirmRate string `json:"confirm_rate"`
}

func (c *ConfirmStatus) AddTxNum() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.initTime) > c.RefreshTime {
		c.TxNum = 0
		c.ConfirmNum = 0
		c.Confirm = time.Duration(0)
		c.initTime = time.Now()
	}
	c.TxNum++
}

func (c *ConfirmStatus) AddConfirm(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Confirm += d
	c.ConfirmNum++
}

func (c *ConfirmStatus) GetInfo() *ConfirmInfo {
	var info = &ConfirmInfo{}
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.TxNum != 0 {
		rate := float64(c.ConfirmNum*100) / float64(c.TxNum)
		if rate == 1 {
			info.ConfirmRate = "100%"
		} else if rate < 98 {
			info.ConfirmRate = fmt.Sprintf("%2d", int(rate)) + "%"
		} else {
			info.ConfirmRate = fmt.Sprintf("%3f", rate) + "%"
		}
	}
	if c.ConfirmNum != 0 {
		info.ConfirmTime = (c.Confirm / time.Duration(c.ConfirmNum)).String()
	}
	return info
}
