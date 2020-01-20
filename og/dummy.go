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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
)

type dummyTxPoolRandomTx struct {
}

func (p *dummyTxPoolRandomTx) GetRandomTips(n int) (v []types.Txi) {
	for i := 0; i < n; i++ {
		v = append(v, tx_types.RandomTx())
	}
	return
}

func (P *dummyTxPoolRandomTx) GetByNonce(addr common.Address, nonce uint64) types.Txi {
	return nil
}

func (p *dummyTxPoolRandomTx) IsBadSeq(seq *tx_types.Sequencer) error {
	return nil
}

type DummyTxPoolMiniTx struct {
	poolMap map[common.Hash]types.Txi
	tipsMap map[common.Hash]types.Txi
}

func (d *DummyTxPoolMiniTx) Init() {
	d.poolMap = make(map[common.Hash]types.Txi)
	d.tipsMap = make(map[common.Hash]types.Txi)
}

func (P *DummyTxPoolMiniTx) GetByNonce(addr common.Address, nonce uint64) types.Txi {
	return nil
}

func (p *DummyTxPoolMiniTx) GetRandomTips(n int) (v []types.Txi) {
	indices := generateRandomIndices(n, len(p.tipsMap))
	// slice of keys
	var keys common.Hashes
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

func (p *DummyTxPoolMiniTx) IsBadSeq(seq *tx_types.Sequencer) error {
	return nil
}

type dummyTxPoolParents struct {
	poolMap map[common.Hash]types.Txi
}

func (p *dummyTxPoolParents) IsLocalHash(h common.Hash) bool {
	return false
}

func (p *dummyTxPoolParents) IsBadSeq(seq *tx_types.Sequencer) error {
	return nil
}

func (P *dummyTxPoolParents) GetByNonce(addr common.Address, nonce uint64) types.Txi {
	return nil
}

func (p *dummyTxPoolParents) GetLatestNonce(addr common.Address) (uint64, error) {
	return 0, fmt.Errorf("not supported")
}

func (p *dummyTxPoolParents) RegisterOnNewTxReceived(c chan types.Txi, s string, b bool) {
	return
}

func (p *dummyTxPoolParents) Init() {
	p.poolMap = make(map[common.Hash]types.Txi)
}

func (p *dummyTxPoolParents) Get(hash common.Hash) types.Txi {
	return p.poolMap[hash]
}

func (p *dummyTxPoolParents) AddRemoteTx(tx types.Txi, b bool) error {
	p.poolMap[tx.GetTxHash()] = tx
	return nil
}

func (p *dummyTxPoolParents) GetMaxWeight() uint64 {
	return 0
}
