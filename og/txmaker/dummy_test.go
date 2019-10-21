package txmaker

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/protocol/ogmessage"

	"github.com/sirupsen/logrus"
)

type dummyTxPoolRandomTx struct {
}

func (p *dummyTxPoolRandomTx) IsBadSeq(seq *ogmessage.Sequencer) error {
	return nil
}

func (p *dummyTxPoolRandomTx) GetRandomTips(n int) (v []ogmessage.Txi) {
	for i := 0; i < n; i++ {
		v = append(v, ogmessage.RandomTx())
	}
	return
}

func (P *dummyTxPoolRandomTx) GetByNonce(addr common.Address, nonce uint64) ogmessage.Txi {
	return nil
}

type dummyTxPoolMiniTx struct {
	poolMap map[common.Hash]ogmessage.Txi
	tipsMap map[common.Hash]ogmessage.Txi
}

func (d *dummyTxPoolMiniTx) IsBadSeq(seq *ogmessage.Sequencer) error {
	return nil
}

func (d *dummyTxPoolMiniTx) Init() {
	d.poolMap = make(map[common.Hash]ogmessage.Txi)
	d.tipsMap = make(map[common.Hash]ogmessage.Txi)
}

func (P *dummyTxPoolMiniTx) GetByNonce(addr common.Address, nonce uint64) ogmessage.Txi {
	return nil
}

func (p *dummyTxPoolMiniTx) GetRandomTips(n int) (v []ogmessage.Txi) {
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

func (p *dummyTxPoolMiniTx) Add(v ogmessage.Txi) {
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
	poolMap map[common.Hash]ogmessage.Txi
}

func (p *dummyTxPoolParents) IsLocalHash(h common.Hash) bool {
	return false
}

func (p *dummyTxPoolParents) IsBadSeq(seq *ogmessage.Sequencer) error {
	return nil
}

func (P *dummyTxPoolParents) GetByNonce(addr common.Address, nonce uint64) ogmessage.Txi {
	return nil
}

func (p *dummyTxPoolParents) GetLatestNonce(addr common.Address) (uint64, error) {
	return 0, fmt.Errorf("not supported")
}

func (p *dummyTxPoolParents) RegisterOnNewTxReceived(c chan ogmessage.Txi, s string, b bool) {
	return
}

func (p *dummyTxPoolParents) Init() {
	p.poolMap = make(map[common.Hash]ogmessage.Txi)
}

func (p *dummyTxPoolParents) Get(hash common.Hash) ogmessage.Txi {
	return p.poolMap[hash]
}

func (p *dummyTxPoolParents) AddRemoteTx(tx ogmessage.Txi, b bool) error {
	p.poolMap[tx.GetTxHash()] = tx
	return nil
}

func (p *dummyTxPoolParents) GetMaxWeight() uint64 {
	return 0
}
