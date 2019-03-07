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
package miner

import (
	"github.com/annchain/OG/types"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestPoW(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	t.Parallel()

	tx := types.SampleTx()
	miner := PoWMiner{}

	responseChan := make(chan uint64)
	start := time.Now()
	go miner.StartMine(tx, types.HexToHash("0x00FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 0, responseChan)

	c, ok := <-responseChan
	logrus.Infof("time: %d ms", time.Since(start).Nanoseconds()/1000000)
	assert.Equal(t, ok, true)
	logrus.Info(c)

}
