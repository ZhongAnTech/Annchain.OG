package txmaker

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
	"github.com/annchain/OG/og/types"

	"github.com/sirupsen/logrus"
)

type dummyTxPoolRandomTx struct {
}

func (p *dummyTxPoolRandomTx) IsBadSeq(seq *types.Sequencer) error {
	return nil
}

func (p *dummyTxPoolRandomTx) GetRandomTips(n int) (v []types.Txi) {
	for i := 0; i < n; i++ {
		v = append(v, archive.RandomTx())
	}
	return
}

func (P *dummyTxPoolRandomTx) GetByNonce(addr common.Address, nonce uint64) types.Txi {
	return nil
}

type dummyTxPoolMiniTx struct {
	poolMap map[common.Hash]types.Txi
	tipsMap map[common.Hash]types.Txi
}

func (d *dummyTxPoolMiniTx) IsBadSeq(seq *types.Sequencer) error {
	return nil
}

func (d *dummyTxPoolMiniTx) Init() {
	d.poolMap = make(map[common.Hash]types.Txi)
	d.tipsMap = make(map[common.Hash]types.Txi)
}

func (P *dummyTxPoolMiniTx) GetByNonce(addr common.Address, nonce uint64) types.Txi {
	return nil
}

func (p *dummyTxPoolMiniTx) GetRandomTips(n int) (v []types.Txi) {
	indices := math.GenerateRandomIndices(n, len(p.tipsMap))
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

func (p *dummyTxPoolMiniTx) Add(v types.Txi) {
	p.tipsMap[v.GetTxHash()] = v

	for _, parentHash := range v.GetParents() {
		if vp, ok := p.tipsMap[parentHash]; ok {
			delete(p.tipsMap, parentHash)
			p.poolMap[parentHash] = vp
		}
	}
	logrus.Infof("added tx %s to tip. current pool size: tips: %d pool: %d",
		v.String(), len(p.tipsMap), len(p.poolMap))
}

type dummyTxPoolParents struct {
	poolMap map[common.Hash]types.Txi
}

func (p *dummyTxPoolParents) IsLocalHash(h common.Hash) bool {
	return false
}

func (p *dummyTxPoolParents) IsBadSeq(seq *types.Sequencer) error {
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
