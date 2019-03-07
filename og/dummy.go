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
package og

import (
	"fmt"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type dummyTxPoolRandomTx struct {
}

func (p *dummyTxPoolRandomTx) GetRandomTips(n int) (v []types.Txi) {
	for i := 0; i < n; i++ {
		v = append(v, types.RandomTx())
	}
	return
}

type DummyTxPoolMiniTx struct {
	poolMap map[types.Hash]types.Txi
	tipsMap map[types.Hash]types.Txi
}

func (d *DummyTxPoolMiniTx) Init() {
	d.poolMap = make(map[types.Hash]types.Txi)
	d.tipsMap = make(map[types.Hash]types.Txi)
}

func (p *DummyTxPoolMiniTx) GetRandomTips(n int) (v []types.Txi) {
	indices := generateRandomIndices(n, len(p.tipsMap))
	// slice of keys
	var keys types.Hashes
	for k := range p.tipsMap {
		keys = append(keys, k)
	}
	for i := range indices {
		v = append(v, p.tipsMap[keys[i]])
	}
	return v
}

func (p *DummyTxPoolMiniTx) Add(v types.Txi) {
	p.tipsMap[v.GetTxHash()] = v

	for _, parentHash := range v.Parents() {
		if vp, ok := p.tipsMap[parentHash]; ok {
			delete(p.tipsMap, parentHash)
			p.poolMap[parentHash] = vp
		}
	}
	logrus.Infof("added tx %s to tip. current pool size: tips: %d pool: %d",
		v.String(), len(p.tipsMap), len(p.poolMap))
}

type dummyTxPoolParents struct {
	poolMap map[types.Hash]types.Txi
}

func (p *dummyTxPoolParents) IsLocalHash(h types.Hash) bool {
	return false
}

func (p *dummyTxPoolParents) GetLatestNonce(addr types.Address) (uint64, error) {
	return 0, fmt.Errorf("not supported")
}

func (p *dummyTxPoolParents) RegisterOnNewTxReceived(c chan types.Txi, s string, b bool) {
	return
}

func (p *dummyTxPoolParents) Init() {
	p.poolMap = make(map[types.Hash]types.Txi)
}

func (p *dummyTxPoolParents) Get(hash types.Hash) types.Txi {
	return p.poolMap[hash]
}

func (p *dummyTxPoolParents) AddRemoteTx(tx types.Txi, b bool) error {
	p.poolMap[tx.GetTxHash()] = tx
	return nil
}

func (p *dummyTxPoolParents) GetMaxWeight() uint64 {
	return 0
}
