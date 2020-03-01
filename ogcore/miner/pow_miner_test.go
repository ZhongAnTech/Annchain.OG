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
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/types"

	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestPoW(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	t.Parallel()

	miner := PoWMiner{}

	acc := account.RandomAccount()

	tx := &types.Tx{
		Hash: common.Hash{},
		ParentsHash: common.Hashes{
			common.HexToHashNoError("0x0003"),
			common.HexToHashNoError("0x0004"),
		},
		MineNonce:    0,
		AccountNonce: 100,
		From:         common.HexToAddressNoError("0x0001"),
		To:           common.HexToAddressNoError("0x0002"),
		Value:        math.NewBigInt(50),
		TokenId:      0,
		Data:         []byte{1, 2, 3, 4},
		PublicKey:    acc.PublicKey,
		Signature:    crypto.Signature{},
		Height:       10,
		Weight:       10,
	}
	responseChan := make(chan uint64)
	start := time.Now()
	go miner.Mine(tx, common.HexToHashNoError("0x0000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 0, responseChan)

	c, ok := <-responseChan
	logrus.Infof("time: %d ms", time.Since(start).Nanoseconds()/1000000)
	assert.Equal(t, ok, true)
	logrus.Info(c)

}
