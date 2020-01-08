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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og/types"

	"math"
)

type PoWMiner struct {
}

func (m *PoWMiner) MineTx(tx *types.Tx, targetMax common.Hash, start uint64) common.Hash {
}

func (m *PoWMiner) StartMine(tx types.Txi, targetMax common.Hash, start uint64, responseChan chan uint64) {
	// do brute force
	//
	var i uint64
	for i = start; i <= math.MaxUint64; i++ {
		tx.SetMineNonce(i)
		//logrus.Debugf("%10d %s %s", i, tx.Hash().Hex(), targetMax.Hex())
		if tx.CalcMinedHash().Cmp(targetMax) < 0 {
			//logrus.Debugf("Hash found: %s with %d", tx.CalcMinedHash().Hex(), i)
			responseChan <- i
			return
		}
	}

}
func (m *PoWMiner) Stop() {
}
