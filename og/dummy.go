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
	var keys []types.Hash
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

func (p *dummyTxPoolParents) GetLatestNonce(addr types.Address) (uint64, error) {
	return 0, fmt.Errorf("not supported")
}

func (p *dummyTxPoolParents) RegisterOnNewTxReceived(c chan types.Txi) {
	return
}

func (p *dummyTxPoolParents) Init() {
	p.poolMap = make(map[types.Hash]types.Txi)
}

func (p *dummyTxPoolParents) Get(hash types.Hash) types.Txi {
	return p.poolMap[hash]
}

func (p *dummyTxPoolParents) AddRemoteTx(tx types.Txi) error {
	p.poolMap[tx.GetTxHash()] = tx
	return nil
}
