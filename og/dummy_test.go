package og

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/og/protocol_message"
	"github.com/annchain/gcache"
	"github.com/sirupsen/logrus"
)

type dummyDag struct {
	dmap map[common.Hash]protocol_message.Txi
}

func (d *dummyDag) GetHeight() uint64 {
	return 0
}

func (d *dummyDag) GetLatestNonce(addr common.Address) (uint64, error) {
	return 0, nil
}

func (d *dummyDag) GetSequencerByHeight(id uint64) *protocol_message.Sequencer {
	return nil
}

func (d *dummyDag) GetSequencerByHash(hash common.Hash) *protocol_message.Sequencer {
	return nil
}

func (d *dummyDag) GetBalance(address common.Address, tokenId int32) *math.BigInt {
	return math.NewBigInt(0)
}

func (d *dummyDag) GetTxByNonce(addr common.Address, nonce uint64) protocol_message.Txi {
	return nil
}

func (d *dummyDag) GetTxisByNumber(id uint64) protocol_message.Txis {
	return nil
}

func (d *dummyDag) GetTestTxisByNumber(id uint64) protocol_message.Txis {
	return nil
}

func (d *dummyDag) LatestSequencer() *protocol_message.Sequencer {
	return nil
}

func (d *dummyDag) GetSequencer(hash common.Hash, id uint64) *protocol_message.Sequencer {
	return nil
}

func (d *dummyDag) Genesis() *protocol_message.Sequencer {
	return nil
}

func (d *dummyDag) init() {
	d.dmap = make(map[common.Hash]protocol_message.Txi)
	tx := sampleTx("0x00", []string{})
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummyDag) GetTx(hash common.Hash) protocol_message.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

func sampleTx(selfHash string, parentsHash []string) *protocol_message.Tx {
	tx := &protocol_message.Tx{TxBase: protocol_message.TxBase{
		ParentsHash: common.Hashes{},
		Type:        protocol_message.TxBaseTypeNormal,
		Hash:        common.HexToHash(selfHash),
	},
	}
	for _, h := range parentsHash {
		tx.ParentsHash = append(tx.ParentsHash, common.HexToHash(h))
	}
	return tx
}

type dummySyncer struct {
	dmap                map[common.Hash]protocol_message.Txi
	buffer              *TxBuffer
	acquireTxDedupCache gcache.Cache
}

func (d *dummySyncer) ClearQueue() {
	for k := range d.dmap {
		delete(d.dmap, k)
	}
}

func (d *dummySyncer) SyncHashList(seqHash common.Hash) {
	return
}

func (d *dummySyncer) Know(tx protocol_message.Txi) {
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummySyncer) IsCachedHash(hash common.Hash) bool {
	return false
}

func (d *dummySyncer) Enqueue(hash *common.Hash, childHash common.Hash, b bool) {
	if _, err := d.acquireTxDedupCache.Get(*hash); err == nil {
		logrus.WithField("Hash", hash).Debugf("duplicate sync task")
		return
	}
	d.acquireTxDedupCache.Set(hash, struct{}{})

	if v, ok := d.dmap[*hash]; ok {
		<-ffchan.NewTimeoutSenderShort(d.buffer.ReceivedNewTxChan, v, "test").C
		logrus.WithField("Hash", hash).Infof("syncer added tx")
		logrus.WithField("Hash", hash).Infof("syncer returned tx")
	} else {
		logrus.WithField("Hash", hash).Infof("syncer does not know tx")
	}

}

type dummyTxPool struct {
	dmap map[common.Hash]protocol_message.Txi
}

func (d *dummyTxPool) GetLatestNonce(addr common.Address) (uint64, error) {
	return 0, fmt.Errorf("not supported")
}

func (p *dummyTxPool) IsBadSeq(seq *protocol_message.Sequencer) error {
	return nil
}

func (d *dummyTxPool) RegisterOnNewTxReceived(c chan protocol_message.Txi, s string, b bool) {
	return
}

func (d *dummyTxPool) GetMaxWeight() uint64 {
	return 0
}

func (d *dummyTxPool) GetByNonce(addr common.Address, nonce uint64) protocol_message.Txi {
	return nil
}

func (d *dummyTxPool) init() {
	d.dmap = make(map[common.Hash]protocol_message.Txi)
	tx := sampleTx("0x01", []string{"0x00"})
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummyTxPool) Get(hash common.Hash) protocol_message.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

func (d *dummyTxPool) AddRemoteTx(tx protocol_message.Txi, b bool) error {
	d.dmap[tx.GetTxHash()] = tx
	return nil
}

func (d *dummyTxPool) IsLocalHash(hash common.Hash) bool {
	return false
}

type dummyVerifier struct{}

func (d *dummyVerifier) Verify(t protocol_message.Txi) bool {
	return true
}

func (d *dummyVerifier) Name() string {
	return "dumnmy verifier"
}

func (d *dummyVerifier) String() string {
	return d.Name()
}

func (d *dummyVerifier) Independent() bool {
	return false
}
